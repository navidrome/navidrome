package matcher

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
	"github.com/xrash/smetrics"
)

// Matcher matches agent song results to local library tracks.
type Matcher struct {
	ds model.DataStore
}

// New creates a new Matcher with the given DataStore.
func New(ds model.DataStore) *Matcher {
	return &Matcher{ds: ds}
}

// MatchSongs matches agent songs to library tracks and returns up to count
// tracks in the input's order. See the package documentation for the matching
// algorithm.
//
// Each library track appears at most once, unless the same input song is
// repeated: identical input songs intentionally yield repeated output tracks,
// while distinct songs that resolve to the same track are deduplicated. Songs
// that cannot be matched are skipped.
func (m *Matcher) MatchSongs(ctx context.Context, songs []agents.Song, count int) (model.MediaFiles, error) {
	if len(songs) == 0 {
		return nil, nil
	}
	matches, err := m.resolveMatches(ctx, songs)
	if err != nil {
		return nil, err
	}
	return orderAndDedup(songs, matches, count), nil
}

// MatchSongsIndexed matches agent songs to library tracks and returns a map from
// input-song index to matched track, letting callers correlate results back to
// the input slice. Unmatched songs are omitted from the map. Unlike MatchSongs,
// results are not deduplicated. See the package documentation for the matching
// algorithm.
func (m *Matcher) MatchSongsIndexed(ctx context.Context, songs []agents.Song) (map[int]model.MediaFile, error) {
	if len(songs) == 0 {
		return nil, nil
	}
	return m.resolveMatches(ctx, songs)
}

// resolveMatches resolves each input song to its best-matching library track,
// keyed by the song's index. Loaders run in priority order (ID > MBID > ISRC >
// Title); each only fills indices not already matched by a higher-priority loader.
func (m *Matcher) resolveMatches(ctx context.Context, songs []agents.Song) (map[int]model.MediaFile, error) {
	result := make(map[int]model.MediaFile, len(songs))
	if err := m.matchByID(ctx, songs, result); err != nil {
		return nil, fmt.Errorf("failed to match tracks by ID: %w", err)
	}
	if err := m.matchByMBID(ctx, songs, result); err != nil {
		return nil, fmt.Errorf("failed to match tracks by MBID: %w", err)
	}
	if err := m.matchByISRC(ctx, songs, result); err != nil {
		return nil, fmt.Errorf("failed to match tracks by ISRC: %w", err)
	}
	// The title phase is best-effort: a DB failure there must not discard the exact
	// matches already found by the higher-priority phases. Only surface it as fatal
	// when nothing matched at all.
	if err := m.matchByTitle(ctx, songs, result); err != nil {
		if len(result) == 0 {
			return nil, fmt.Errorf("failed to match tracks by title: %w", err)
		}
		log.Warn(ctx, "Title matching failed; returning matches from exact phases only", err)
	}
	return result, nil
}

// matchByID fills result with direct ID matches.
func (m *Matcher) matchByID(ctx context.Context, songs []agents.Song, result map[int]model.MediaFile) error {
	var ids []string
	for _, s := range songs {
		if s.ID != "" {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	res, err := m.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"media_file.id": ids},
			squirrel.Eq{"missing": false},
		},
	})
	if err != nil {
		return err
	}
	byID := make(map[string]model.MediaFile, len(res))
	for _, mf := range res {
		byID[mf.ID] = mf // media_file.id is unique, so no dedup needed
	}
	for i, s := range songs {
		if s.ID == "" {
			continue
		}
		if mf, ok := byID[s.ID]; ok {
			result[i] = mf
		}
	}
	return nil
}

// matchByMBID fills result with MusicBrainz Recording ID matches, skipping
// songs already matched by a higher-priority loader.
func (m *Matcher) matchByMBID(ctx context.Context, songs []agents.Song, result map[int]model.MediaFile) error {
	var mbids []string
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		if s.MBID != "" {
			mbids = append(mbids, s.MBID)
		}
	}
	if len(mbids) == 0 {
		return nil
	}
	res, err := m.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"mbz_recording_id": mbids},
			squirrel.Eq{"missing": false},
		},
	})
	if err != nil {
		return err
	}
	byMBID := make(map[string]model.MediaFile, len(res))
	for _, mf := range res {
		if id := mf.MbzRecordingID; id != "" {
			if _, ok := byMBID[id]; !ok {
				byMBID[id] = mf
			}
		}
	}
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		if s.MBID == "" {
			continue
		}
		if mf, ok := byMBID[s.MBID]; ok {
			result[i] = mf
		}
	}
	return nil
}

// matchByISRC fills result with ISRC tag matches, skipping songs already
// matched by a higher-priority loader.
func (m *Matcher) matchByISRC(ctx context.Context, songs []agents.Song, result map[int]model.MediaFile) error {
	var isrcs []string
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		if s.ISRC != "" {
			isrcs = append(isrcs, s.ISRC)
		}
	}
	if len(isrcs) == 0 {
		return nil
	}
	res, err := m.ds.MediaFile(ctx).GetAllByTags(model.TagISRC, isrcs, model.QueryOptions{
		Filters: squirrel.Eq{"missing": false},
		Sort:    "starred desc, rating desc, year asc, compilation asc",
	})
	if err != nil {
		return err
	}
	byISRC := make(map[string]model.MediaFile, len(res))
	for _, mf := range res {
		for _, isrc := range mf.Tags.Values(model.TagISRC) {
			if _, ok := byISRC[isrc]; !ok {
				byISRC[isrc] = mf
			}
		}
	}
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		if s.ISRC == "" {
			continue
		}
		if mf, ok := byISRC[s.ISRC]; ok {
			result[i] = mf
		}
	}
	return nil
}

// queryArtist is one of a song's artists. A non-empty id is matched directly, skipping name/MBID
// resolution; name is pre-sanitized (article-stripped).
type queryArtist struct {
	id   string
	name string
	mbid string
}

// songQuery represents a normalized query for matching a song to library tracks.
type songQuery struct {
	title      string
	artists    []queryArtist
	album      string
	albumMBID  string
	durationMs uint32
}

// matchScore combines title/album similarity with metadata specificity for ranking matches.
type matchScore struct {
	titleSimilarity   float64
	durationProximity float64
	preferredMatch    bool
	specificityLevel  int
	artistOverlap     int
	albumSimilarity   float64
}

// betterThan returns true if this score beats another.
// Identity signals (specificity, overlap) outrank the taste signal (preferred).
func (s matchScore) betterThan(other matchScore) bool {
	if s.titleSimilarity != other.titleSimilarity {
		return s.titleSimilarity > other.titleSimilarity
	}
	if s.durationProximity != other.durationProximity {
		return s.durationProximity > other.durationProximity
	}
	if s.specificityLevel != other.specificityLevel {
		return s.specificityLevel > other.specificityLevel
	}
	if s.artistOverlap != other.artistOverlap {
		return s.artistOverlap > other.artistOverlap
	}
	if s.preferredMatch != other.preferredMatch {
		return s.preferredMatch
	}
	return s.albumSimilarity > other.albumSimilarity
}

// sanitizedTrack holds pre-sanitized fields for a media file, avoiding redundant sanitization
// when the same track is scored against multiple queries. The `mf` field is a pointer to avoid
// copying the large MediaFile struct into each entry of the sanitized slice.
type sanitizedTrack struct {
	mf          *model.MediaFile
	title       string
	artist      string
	album       string
	artistIDs   map[string]struct{} // query's owned artist IDs this track credits; an ID match is the strongest identity signal
	artistMBIDs map[string]struct{} // MBIDs of those artists (artist table; mf.MbzArtistID is not populated on the bulk path)
}

func newSanitizedTrack(mf *model.MediaFile, artistIDs, artistMBIDs map[string]struct{}) sanitizedTrack {
	return sanitizedTrack{
		mf:          mf,
		title:       str.SanitizeFieldForSorting(mf.Title),
		artist:      str.SanitizeFieldForSortingNoArticle(mf.Artist),
		album:       str.SanitizeFieldForSorting(mf.Album),
		artistIDs:   artistIDs,
		artistMBIDs: artistMBIDs,
	}
}

// computeSpecificityLevel determines how well query metadata matches a track (0-5), taking the best
// level achievable across any of the query's artists. Fields must be pre-sanitized.
//
// A query artist counts as an identity match when the track credits its resolved Navidrome ID (the
// strongest signal, our own primary key) or its MBID; that identity then unlocks the album tiers.
// Name matching is the lowest fallback for an artist with no identity match (e.g. a cover credited
// to a different artist by the same name).
func computeSpecificityLevel(q songQuery, t sanitizedTrack, albumThreshold float64) int {
	best := 0
	albumOK := q.album != "" && similarityRatio(t.album, q.album) >= albumThreshold
	for _, a := range q.artists {
		_, idMember := t.artistIDs[a.id]
		_, mbidMember := t.artistMBIDs[a.mbid]
		identity := (a.id != "" && idMember) || (a.mbid != "" && mbidMember)
		level := 0
		switch {
		case identity && q.albumMBID != "" && t.mf.MbzAlbumID == q.albumMBID:
			level = 5
		case identity && q.album != "" && albumOK:
			level = 4
		case a.name != "" && q.album != "" && t.artist == a.name && albumOK:
			level = 3
		case identity:
			level = 2
		case a.name != "" && t.artist == a.name:
			level = 1
		}
		if level > best {
			best = level
		}
	}
	return best
}

// indexedQuery pairs a normalized songQuery with the index of the input song
// it came from, so title matches can be written back to result by index.
type indexedQuery struct {
	index int
	query songQuery
}

// matchByTitle fills result with fuzzy title+artist matches, skipping songs
// already matched by a higher-priority loader.
func (m *Matcher) matchByTitle(ctx context.Context, songs []agents.Song, result map[int]model.MediaFile) error {
	queries := groupQueries(songs, result)
	if len(queries) == 0 {
		return nil
	}

	resolved, err := m.resolveArtists(ctx, queries)
	if err != nil || len(resolved.allIDs) == 0 {
		return err
	}

	tracks, err := m.fetchTracksCreditedTo(ctx, resolved.allIDs)
	if err != nil {
		return err
	}

	tracksByQuery := resolved.bucketTracks(tracks)
	threshold := float64(conf.Server.Matcher.FuzzyThreshold) / 100.0
	for _, iq := range queries {
		sanitized := tracksByQuery[iq.index]
		if mf, found := m.findBestMatch(iq.query, sanitized, threshold); found {
			result[iq.index] = mf
		}
	}
	return nil
}

// groupQueries builds one normalized title query per still-unmatched song, carrying its full
// artist set. An artist is usable if it carries a Navidrome ID or a non-empty sanitized name;
// songs with no usable artist are skipped (the title phase needs at least one to scope the query).
func groupQueries(songs []agents.Song, result map[int]model.MediaFile) []indexedQuery {
	var queries []indexedQuery
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		var artists []queryArtist
		for _, a := range s.Artists {
			name := str.SanitizeFieldForSortingNoArticle(a.Name)
			if a.ID == "" && name == "" && a.MBID == "" {
				continue
			}
			artists = append(artists, queryArtist{id: a.ID, name: name, mbid: a.MBID})
		}
		if len(artists) == 0 {
			continue
		}
		queries = append(queries, indexedQuery{index: i, query: songQuery{
			title:      str.SanitizeFieldForSorting(s.Name),
			artists:    artists,
			album:      str.SanitizeFieldForSorting(s.Album),
			albumMBID:  s.AlbumMBID,
			durationMs: s.Duration,
		}})
	}
	return queries
}

// resolvedArtists holds the agent artists resolved to artist-table rows, keyed by the query index
// that owns them. Routing is always by stable artist ID, never by name.
type resolvedArtists struct {
	byQuery map[int]map[string]struct{} // query index -> set of resolved artist IDs
	mbid    map[string]string           // artist ID -> its MBID (from the artist table)
	allIDs  []string                    // every resolved artist ID, for the track lookup
}

// resolveArtists resolves every artist of every query to artist-table rows. Artists that carry a
// Navidrome ID are owned directly (no name/MBID lookup). The remaining names/MBIDs are resolved in
// one batched query. Ownership is recorded per query index.
func (m *Matcher) resolveArtists(ctx context.Context, queries []indexedQuery) (resolvedArtists, error) {
	res := resolvedArtists{
		byQuery: make(map[int]map[string]struct{}, len(queries)),
		mbid:    make(map[string]string),
	}
	allIDs := map[string]struct{}{} // de-dupe across fast-path + resolved

	// One pending entry per non-ID artist (carrying the query that owns it). ID-bearing artists
	// take the fast-path and are owned directly.
	type pendingArtist struct {
		name, mbid string
		query      int
	}
	var pending []pendingArtist
	for _, iq := range queries {
		for _, a := range iq.query.artists {
			if a.id != "" {
				addToSet(res.byQuery, iq.index, a.id) // ID fast-path: own directly
				allIDs[a.id] = struct{}{}
				continue
			}
			pending = append(pending, pendingArtist{name: a.name, mbid: a.mbid, query: iq.index})
		}
	}
	// query indices that supplied each order name / each MBID (skip the empty key — an artist may
	// have only one of name/mbid).
	nameToQueries := map[string][]int{}
	mbidToQueries := map[string][]int{}
	for _, p := range pending {
		if p.name != "" {
			nameToQueries[p.name] = append(nameToQueries[p.name], p.query)
		}
		if p.mbid != "" {
			mbidToQueries[p.mbid] = append(mbidToQueries[p.mbid], p.query)
		}
	}

	// Query the artist table for name/MBID artists AND for the fast-path IDs (so their MBIDs are
	// available for specificity scoring).
	var filter squirrel.Or
	if len(nameToQueries) > 0 {
		filter = append(filter, squirrel.Eq{"order_artist_name": slices.Collect(maps.Keys(nameToQueries))})
	}
	if len(mbidToQueries) > 0 {
		filter = append(filter, squirrel.Eq{"mbz_artist_id": slices.Collect(maps.Keys(mbidToQueries))})
	}
	if len(allIDs) > 0 {
		filter = append(filter, squirrel.Eq{"id": slices.Collect(maps.Keys(allIDs))})
	}
	if len(filter) > 0 {
		artists, err := m.ds.Artist(ctx).GetAll(model.QueryOptions{Filters: filter})
		if err != nil {
			return resolvedArtists{}, err
		}
		for _, a := range artists {
			res.mbid[a.ID] = a.MbzArtistID
			allIDs[a.ID] = struct{}{}
			for _, idx := range nameToQueries[a.OrderArtistName] {
				addToSet(res.byQuery, idx, a.ID)
			}
			if a.MbzArtistID != "" {
				for _, idx := range mbidToQueries[a.MbzArtistID] {
					addToSet(res.byQuery, idx, a.ID)
				}
			}
		}
	}

	res.allIDs = slices.Collect(maps.Keys(allIDs))
	return res, nil
}

func addToSet(m map[int]map[string]struct{}, k int, v string) {
	if m[k] == nil {
		m[k] = map[string]struct{}{}
	}
	m[k][v] = struct{}{}
}

// scoredTrack is a candidate track for a query; overlap is how many of the query's distinct
// artist IDs the track credits.
type scoredTrack struct {
	sanitizedTrack
	overlap int
}

// queryAccum tallies, for one track against one query, the overlap count, the credited artists'
// owned IDs, and their MBIDs.
type queryAccum struct {
	overlap int
	ids     map[string]struct{}
	mbids   map[string]struct{}
}

func (r resolvedArtists) bucketTracks(tracks []model.MediaFile) map[int][]scoredTrack {
	queriesByArtist := make(map[string][]int)
	for idx, ids := range r.byQuery {
		for id := range ids {
			queriesByArtist[id] = append(queriesByArtist[id], idx)
		}
	}

	byQuery := make(map[int][]scoredTrack, len(r.byQuery))
	for i := range tracks {
		acc := map[int]*queryAccum{}
		credited := map[string]struct{}{}
		for _, p := range tracks[i].Participants[model.RoleArtist] {
			if _, dup := credited[p.ID]; dup {
				continue
			}
			owners, owned := queriesByArtist[p.ID]
			if !owned {
				continue
			}
			credited[p.ID] = struct{}{}
			mbid := r.mbid[p.ID] // "" if not in the artist-table result
			for _, idx := range owners {
				a := acc[idx]
				if a == nil {
					a = &queryAccum{ids: map[string]struct{}{}, mbids: map[string]struct{}{}}
					acc[idx] = a
				}
				a.overlap++
				a.ids[p.ID] = struct{}{}
				if mbid != "" {
					a.mbids[mbid] = struct{}{}
				}
			}
		}
		for idx, a := range acc {
			byQuery[idx] = append(byQuery[idx], scoredTrack{
				sanitizedTrack: newSanitizedTrack(&tracks[i], a.ids, a.mbids),
				overlap:        a.overlap,
			})
		}
	}
	return byQuery
}

// fetchTracksCreditedTo fetches every non-missing track credited to any of the given artists as
// the main artist (role='artist', not albumartist — that avoids tribute/compilation false
// positives). The non-correlated id IN (subquery) materializes the matching ids once from the
// media_file_artists(artist_id) covering index, far cheaper than a correlated EXISTS that re-runs
// per row. That form isn't expressible via the repository's role filters, so the raw squirrel.Expr
// keeps the media_file_artists schema knowledge here; a dedicated repository method would be the
// cleaner home if this is reused.
func (m *Matcher) fetchTracksCreditedTo(ctx context.Context, artistIDs []string) (model.MediaFiles, error) {
	if len(artistIDs) == 0 {
		return nil, nil
	}
	args := slice.Map(artistIDs, func(id string) any { return id })
	return m.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Expr(
				"media_file.id IN (SELECT media_file_id FROM media_file_artists "+
					"WHERE role = 'artist' AND artist_id IN ("+squirrel.Placeholders(len(artistIDs))+"))", args...),
			squirrel.Eq{"missing": false},
		},
		Sort: "starred desc, rating desc, year asc, compilation asc",
	})
}

// durationProximity returns a score from 0.0 to 1.0 indicating how close the track's duration
// is to the target. Returns 1.0 if durationMs is 0 (unknown).
func durationProximity(durationMs uint32, mediaFileDurationSec float32) float64 {
	if durationMs == 0 {
		return 1.0
	}
	durationSec := float64(durationMs) / 1000.0
	diff := math.Abs(durationSec - float64(mediaFileDurationSec))
	return 1.0 / (1.0 + diff)
}

// findBestMatch finds the best matching track using combined title/album similarity and specificity scoring.
func (m *Matcher) findBestMatch(q songQuery, candidates []scoredTrack, threshold float64) (model.MediaFile, bool) {
	var bestMatch model.MediaFile
	bestScore := matchScore{titleSimilarity: -1}
	found := false

	preferStarred := conf.Server.Matcher.PreferStarred
	for _, c := range candidates {
		titleSim := similarityRatio(q.title, c.title)
		if titleSim < threshold {
			continue
		}
		var albumSim float64
		if q.album != "" {
			albumSim = similarityRatio(q.album, c.album)
		}
		score := matchScore{
			titleSimilarity:   titleSim,
			durationProximity: durationProximity(q.durationMs, c.mf.Duration),
			preferredMatch:    preferStarred && isPreferredTrack(c.mf),
			albumSimilarity:   albumSim,
			specificityLevel:  computeSpecificityLevel(q, c.sanitizedTrack, threshold),
			artistOverlap:     c.overlap,
		}
		if score.betterThan(bestScore) {
			bestScore = score
			bestMatch = *c.mf
			found = true
		}
	}
	return bestMatch, found
}

func isPreferredTrack(mf *model.MediaFile) bool {
	return mf.Starred || mf.Rating >= 4
}

// orderAndDedup builds the final ordered result from the per-index matches,
// applying the count limit and deduplication. A library track is added at most
// once unless the same input song appears more than once (callers rely on that
// 1:1 positional behavior for identical duplicate inputs).
func orderAndDedup(songs []agents.Song, matches map[int]model.MediaFile, count int) model.MediaFiles {
	mfs := make(model.MediaFiles, 0, len(songs))
	addedBy := make(map[string]int, len(songs))

	for i, s := range songs {
		if len(mfs) == count {
			break
		}
		mf, found := matches[i]
		if !found {
			continue
		}
		if prevIdx, alreadyAdded := addedBy[mf.ID]; alreadyAdded {
			if !s.Equals(songs[prevIdx]) {
				continue
			}
		} else {
			addedBy[mf.ID] = i
		}
		mfs = append(mfs, mf)
	}
	return mfs
}

// similarityRatio calculates the similarity between two strings using Jaro-Winkler algorithm.
func similarityRatio(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	return smetrics.JaroWinkler(a, b, 0.7, 4)
}
