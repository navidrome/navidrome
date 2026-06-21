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

// songQuery represents a normalized query for matching a song to library tracks.
type songQuery struct {
	title      string
	artist     string
	artistMBID string
	album      string
	albumMBID  string
	durationMs uint32
}

// matchScore combines title/album similarity with metadata specificity for ranking matches.
type matchScore struct {
	titleSimilarity   float64
	durationProximity float64
	preferredMatch    bool
	albumSimilarity   float64
	specificityLevel  int
}

// betterThan returns true if this score beats another.
func (s matchScore) betterThan(other matchScore) bool {
	if s.titleSimilarity != other.titleSimilarity {
		return s.titleSimilarity > other.titleSimilarity
	}
	if s.durationProximity != other.durationProximity {
		return s.durationProximity > other.durationProximity
	}
	if s.preferredMatch != other.preferredMatch {
		return s.preferredMatch
	}
	if s.specificityLevel != other.specificityLevel {
		return s.specificityLevel > other.specificityLevel
	}
	return s.albumSimilarity > other.albumSimilarity
}

// sanitizedTrack holds pre-sanitized fields for a media file, avoiding redundant sanitization
// when the same track is scored against multiple queries. The `mf` field is a pointer to avoid
// copying the large MediaFile struct into each entry of the sanitized slice.
type sanitizedTrack struct {
	mf         *model.MediaFile
	title      string
	artist     string
	album      string
	artistMBID string // resolved from the artist table; mf.MbzArtistID is not populated on the bulk path
}

func newSanitizedTrack(mf *model.MediaFile, artistMBID string) sanitizedTrack {
	return sanitizedTrack{
		mf:         mf,
		title:      str.SanitizeFieldForSorting(mf.Title),
		artist:     str.SanitizeFieldForSortingNoArticle(mf.Artist),
		album:      str.SanitizeFieldForSorting(mf.Album),
		artistMBID: artistMBID,
	}
}

// computeSpecificityLevel determines how well query metadata matches a track (0-5).
// The track's title, artist, and album fields must be pre-sanitized, and artistMBID
// must hold the resolved artist MBID.
func computeSpecificityLevel(q songQuery, t sanitizedTrack, albumThreshold float64) int {
	if q.artistMBID != "" && q.albumMBID != "" &&
		t.artistMBID == q.artistMBID && t.mf.MbzAlbumID == q.albumMBID {
		return 5
	}
	if q.artistMBID != "" && q.album != "" &&
		t.artistMBID == q.artistMBID && similarityRatio(t.album, q.album) >= albumThreshold {
		return 4
	}
	if q.artist != "" && q.album != "" &&
		t.artist == q.artist && similarityRatio(t.album, q.album) >= albumThreshold {
		return 3
	}
	if q.artistMBID != "" && t.artistMBID == q.artistMBID {
		return 2
	}
	if q.artist != "" && t.artist == q.artist {
		return 1
	}
	return 0
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
	byArtist := groupQueriesByArtist(songs, result)
	if len(byArtist) == 0 {
		return nil
	}

	resolved, err := m.resolveArtists(ctx, byArtist)
	if err != nil || len(resolved.allIDs) == 0 {
		return err
	}

	tracks, err := m.fetchTracksCreditedTo(ctx, resolved.allIDs)
	if err != nil {
		return err
	}

	tracksByQuery := resolved.bucketTracks(tracks)
	threshold := float64(conf.Server.Matcher.FuzzyThreshold) / 100.0
	for artist, queries := range byArtist {
		sanitized := tracksByQuery[artist]
		// Each song is matched independently by index, so two songs with the same
		// (title, artist) but different durations can resolve to different tracks.
		for _, iq := range queries {
			if mf, found := m.findBestMatch(iq.query, sanitized, threshold); found {
				result[iq.index] = mf
			}
		}
	}
	return nil
}

// groupQueriesByArtist buckets the still-unmatched title queries by sanitized artist name.
// Songs without an artist are skipped: title matching needs one to scope the library query.
func groupQueriesByArtist(songs []agents.Song, result map[int]model.MediaFile) map[string][]indexedQuery {
	byArtist := map[string][]indexedQuery{}
	for i, s := range songs {
		if _, done := result[i]; done {
			continue
		}
		artist := str.SanitizeFieldForSortingNoArticle(s.Artist)
		if artist == "" {
			continue
		}
		byArtist[artist] = append(byArtist[artist], indexedQuery{index: i, query: songQuery{
			title:      str.SanitizeFieldForSorting(s.Name),
			artist:     artist,
			artistMBID: s.ArtistMBID,
			album:      str.SanitizeFieldForSorting(s.Album),
			albumMBID:  s.AlbumMBID,
			durationMs: s.Duration,
		}})
	}
	return byArtist
}

// resolvedArtists holds the agent artists resolved to artist-table rows. Everything routes by
// stable artist ID, never by name, so MBID-resolved artists whose order name differs from the
// query name are not misrouted.
type resolvedArtists struct {
	byQuery map[string]map[string]struct{} // sanitized query name -> set of resolved artist IDs
	mbid    map[string]string              // artist ID -> its MBID (the real one, from the artist table)
	allIDs  []string                       // every resolved artist ID, for the track lookup
}

// resolveArtists resolves the queries' artists against the artist table (by sort name or
// agent-provided MBID) and records, for each query, which artist IDs it owns.
func (m *Matcher) resolveArtists(ctx context.Context, byArtist map[string][]indexedQuery) (resolvedArtists, error) {
	names := make([]string, 0, len(byArtist))
	mbidToQueries := make(map[string][]string, len(byArtist)) // agent ArtistMBID -> query names that supplied it
	for name, queries := range byArtist {
		names = append(names, name)
		for _, iq := range queries {
			if iq.query.artistMBID != "" {
				mbidToQueries[iq.query.artistMBID] = append(mbidToQueries[iq.query.artistMBID], name)
			}
		}
	}

	filter := squirrel.Or{squirrel.Eq{"order_artist_name": names}}
	if len(mbidToQueries) > 0 {
		filter = append(filter, squirrel.Eq{"mbz_artist_id": slices.Collect(maps.Keys(mbidToQueries))})
	}
	artists, err := m.ds.Artist(ctx).GetAll(model.QueryOptions{Filters: filter})
	if err != nil {
		return resolvedArtists{}, err
	}

	res := resolvedArtists{
		byQuery: make(map[string]map[string]struct{}, len(byArtist)),
		mbid:    make(map[string]string, len(artists)),
		allIDs:  make([]string, 0, len(artists)),
	}
	for _, a := range artists {
		res.mbid[a.ID] = a.MbzArtistID
		res.allIDs = append(res.allIDs, a.ID)
		// An artist belongs to a query if its order name matches the query name, or if its MBID
		// matches one a query supplied. The same MBID can come from several queries (agent aliases),
		// so every one of them owns the artist.
		res.own(a.OrderArtistName, a.ID)
		if a.MbzArtistID != "" {
			for _, name := range mbidToQueries[a.MbzArtistID] {
				res.own(name, a.ID)
			}
		}
	}
	return res, nil
}

// own records that the named query owns the given artist ID. A name that is not a query simply
// gets its own (unused) entry.
func (r resolvedArtists) own(name, artistID string) {
	if r.byQuery[name] == nil {
		r.byQuery[name] = map[string]struct{}{}
	}
	r.byQuery[name][artistID] = struct{}{}
}

// bucketTracks groups tracks by query name, at most once per query even when a track credits
// several of that query's artists, so the same track is not scored twice. The participants JSON
// on each track carries artist IDs but not their MBID, so the MBID comes from r.mbid instead.
func (r resolvedArtists) bucketTracks(tracks []model.MediaFile) map[string][]sanitizedTrack {
	// Invert byQuery once so each participant maps straight to the queries that own it, instead of
	// scanning every query per participant.
	queriesByArtist := make(map[string][]string)
	for name, ids := range r.byQuery {
		for id := range ids {
			queriesByArtist[id] = append(queriesByArtist[id], name)
		}
	}

	byQuery := make(map[string][]sanitizedTrack, len(r.byQuery))
	added := make(map[string]map[string]struct{}, len(r.byQuery)) // query name -> set of track IDs already bucketed
	for i := range tracks {
		for _, p := range tracks[i].Participants[model.RoleArtist] {
			mbid, isResolved := r.mbid[p.ID]
			if !isResolved {
				continue
			}
			for _, name := range queriesByArtist[p.ID] {
				if added[name] == nil {
					added[name] = map[string]struct{}{}
				}
				if _, dup := added[name][tracks[i].ID]; dup {
					continue
				}
				added[name][tracks[i].ID] = struct{}{}
				byQuery[name] = append(byQuery[name], newSanitizedTrack(&tracks[i], mbid))
			}
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
func (m *Matcher) findBestMatch(q songQuery, sanitizedTracks []sanitizedTrack, threshold float64) (model.MediaFile, bool) {
	var bestMatch model.MediaFile
	bestScore := matchScore{titleSimilarity: -1}
	found := false

	preferStarred := conf.Server.Matcher.PreferStarred
	for _, t := range sanitizedTracks {
		titleSim := similarityRatio(q.title, t.title)

		if titleSim < threshold {
			continue
		}

		var albumSim float64
		if q.album != "" {
			albumSim = similarityRatio(q.album, t.album)
		}

		score := matchScore{
			titleSimilarity:   titleSim,
			durationProximity: durationProximity(q.durationMs, t.mf.Duration),
			preferredMatch:    preferStarred && isPreferredTrack(t.mf),
			albumSimilarity:   albumSim,
			specificityLevel:  computeSpecificityLevel(q, t, threshold),
		}

		if score.betterThan(bestScore) {
			bestScore = score
			bestMatch = *t.mf
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
			if s != songs[prevIdx] {
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
