package external

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
	"github.com/xrash/smetrics"
)

// matchSongsToLibrary matches agent song results to local library tracks
func (e *provider) matchSongsToLibrary(ctx context.Context, songs []agents.Song, count int) (model.MediaFiles, error) {
	idMatches, err := e.loadTracksByID(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by ID: %w", err)
	}
	mbidMatches, err := e.loadTracksByMBID(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by MBID: %w", err)
	}
	titleMatches, err := e.loadTracksByTitleAndArtist(ctx, songs, idMatches, mbidMatches)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by title: %w", err)
	}

	return e.selectBestMatchingSongs(songs, idMatches, mbidMatches, titleMatches, count), nil
}

func (e *provider) loadTracksByID(ctx context.Context, songs []agents.Song) (map[string]model.MediaFile, error) {
	var ids []string
	for _, s := range songs {
		if s.ID != "" {
			ids = append(ids, s.ID)
		}
	}
	matches := map[string]model.MediaFile{}
	if len(ids) == 0 {
		return matches, nil
	}
	res, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"media_file.id": ids},
			squirrel.Eq{"missing": false},
		},
	})
	if err != nil {
		return matches, err
	}
	for _, mf := range res {
		if _, ok := matches[mf.ID]; !ok {
			matches[mf.ID] = mf
		}
	}
	return matches, nil
}

func (e *provider) loadTracksByMBID(ctx context.Context, songs []agents.Song) (map[string]model.MediaFile, error) {
	var mbids []string
	for _, s := range songs {
		if s.MBID != "" {
			mbids = append(mbids, s.MBID)
		}
	}
	matches := map[string]model.MediaFile{}
	if len(mbids) == 0 {
		return matches, nil
	}
	res, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"mbz_recording_id": mbids},
			squirrel.Eq{"missing": false},
		},
	})
	if err != nil {
		return matches, err
	}
	for _, mf := range res {
		if id := mf.MbzRecordingID; id != "" {
			if _, ok := matches[id]; !ok {
				matches[id] = mf
			}
		}
	}
	return matches, nil
}

type songQuery struct {
	title      string
	artist     string
	artistMBID string
	album      string
	albumMBID  string
}

// loadTracksByTitleAndArtist loads tracks matching by title with optional artist/album filtering.
// Uses per-song artist/album info when available for more precise matching.
// Falls back to fuzzy matching for unmatched queries if configured.
// Returns a map keyed by sanitized title for compatibility with selectBestMatchingSongs.
func (e *provider) loadTracksByTitleAndArtist(ctx context.Context, songs []agents.Song, idMatches, mbidMatches map[string]model.MediaFile) (map[string]model.MediaFile, error) {
	queries := e.buildTitleQueries(songs, idMatches, mbidMatches)
	matches := map[string]model.MediaFile{}
	if len(queries) == 0 {
		return matches, nil
	}

	res, err := e.queryTracksByTitles(ctx, queries)
	if err != nil {
		return matches, err
	}

	// Build index from query results
	matcher := newTrackMatcher()
	for _, mf := range res {
		matcher.index(mf)
	}

	// Find best match for each query
	for _, q := range queries {
		if mf, found := matcher.find(q); found {
			key := q.title + "|" + q.artist
			if _, ok := matches[key]; !ok {
				matches[key] = mf
			}
		}
	}

	// Find unmatched queries and try fuzzy matching
	var unmatched []songQuery
	for _, q := range queries {
		key := q.title + "|" + q.artist
		if _, ok := matches[key]; !ok {
			unmatched = append(unmatched, q)
		}
	}

	if len(unmatched) > 0 {
		fuzzyMatches, err := e.fuzzyMatchUnmatched(ctx, unmatched, matches)
		if err == nil && len(fuzzyMatches) > 0 {
			for k, v := range fuzzyMatches {
				matches[k] = v
			}
		}
	}

	return matches, nil
}

func (e *provider) buildTitleQueries(songs []agents.Song, idMatches, mbidMatches map[string]model.MediaFile) []songQuery {
	var queries []songQuery
	for _, s := range songs {
		// Skip if already matched by ID or MBID
		if s.ID != "" && idMatches[s.ID].ID != "" {
			continue
		}
		if s.MBID != "" && mbidMatches[s.MBID].ID != "" {
			continue
		}
		queries = append(queries, songQuery{
			title:      str.SanitizeFieldForSorting(s.Name),
			artist:     str.SanitizeFieldForSorting(s.Artist),
			artistMBID: s.ArtistMBID,
			album:      str.SanitizeFieldForSorting(s.Album),
			albumMBID:  s.AlbumMBID,
		})
	}
	return queries
}

func (e *provider) queryTracksByTitles(ctx context.Context, queries []songQuery) (model.MediaFiles, error) {
	titleSet := map[string]struct{}{}
	for _, q := range queries {
		titleSet[q.title] = struct{}{}
	}

	titleFilters := squirrel.Or{}
	for title := range titleSet {
		titleFilters = append(titleFilters, squirrel.Like{"order_title": title})
	}

	return e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			titleFilters,
			squirrel.Eq{"missing": false},
		},
		Sort: "starred desc, rating desc, year asc, compilation asc",
	})
}

// keyFunc builds a lookup key from track metadata. Returns empty string if required fields are missing.
type keyFunc func(title, artist, album, artistMBID, albumMBID string) string

// Key builders - ordered from most to least specific. Empty string means "skip this level".
func keyTitleArtistMBIDAlbumMBID(t, _, _, am, alm string) string {
	if am == "" || alm == "" {
		return ""
	}
	return t + "|" + am + "|" + alm
}

func keyTitleArtistMBIDAlbum(t, _, al, am, _ string) string {
	if am == "" || al == "" {
		return ""
	}
	return t + "|" + am + "|" + al
}

func keyTitleArtistAlbum(t, a, al, _, _ string) string {
	if a == "" || al == "" {
		return ""
	}
	return t + "|" + a + "|" + al
}

func keyTitleArtistMBID(t, _, _, am, _ string) string {
	if am == "" {
		return ""
	}
	return t + "|" + am
}

func keyTitleArtist(t, a, _, _, _ string) string {
	if a == "" {
		return ""
	}
	return t + "|" + a
}

func keyTitleOnly(t, _, _, _, _ string) string {
	return t
}

// matchLevel defines one level of matching specificity
type matchLevel struct {
	index   map[string]model.MediaFile
	makeKey keyFunc
}

// trackMatcher indexes tracks at multiple specificity levels for efficient lookup
type trackMatcher struct {
	levels []matchLevel
}

// newTrackMatcher creates a matcher with levels ordered from most to least specific
func newTrackMatcher() *trackMatcher {
	return &trackMatcher{
		levels: []matchLevel{
			{make(map[string]model.MediaFile), keyTitleArtistMBIDAlbumMBID},
			{make(map[string]model.MediaFile), keyTitleArtistMBIDAlbum},
			{make(map[string]model.MediaFile), keyTitleArtistAlbum},
			{make(map[string]model.MediaFile), keyTitleArtistMBID},
			{make(map[string]model.MediaFile), keyTitleArtist},
			{make(map[string]model.MediaFile), keyTitleOnly},
		},
	}
}

// index adds a track to all applicable specificity levels
func (m *trackMatcher) index(mf model.MediaFile) {
	title := str.SanitizeFieldForSorting(mf.Title)
	artist := str.SanitizeFieldForSorting(mf.Artist)
	album := str.SanitizeFieldForSorting(mf.Album)

	for i := range m.levels {
		if key := m.levels[i].makeKey(title, artist, album, mf.MbzArtistID, mf.MbzAlbumID); key != "" {
			if _, exists := m.levels[i].index[key]; !exists {
				m.levels[i].index[key] = mf
			}
		}
	}
}

// find returns the best match for a query, trying most specific levels first
func (m *trackMatcher) find(q songQuery) (model.MediaFile, bool) {
	for _, level := range m.levels {
		if key := level.makeKey(q.title, q.artist, q.album, q.artistMBID, q.albumMBID); key != "" {
			if mf, ok := level.index[key]; ok {
				return mf, true
			}
		}
	}
	return model.MediaFile{}, false
}

func (e *provider) selectBestMatchingSongs(songs []agents.Song, byID, byMBID, byTitleArtist map[string]model.MediaFile, count int) model.MediaFiles {
	var mfs model.MediaFiles
	for _, t := range songs {
		if len(mfs) == count {
			break
		}
		// Try ID match first
		if t.ID != "" {
			if mf, ok := byID[t.ID]; ok {
				mfs = append(mfs, mf)
				continue
			}
		}
		// Try MBID match second
		if t.MBID != "" {
			if mf, ok := byMBID[t.MBID]; ok {
				mfs = append(mfs, mf)
				continue
			}
		}
		// Fall back to title+artist match (composite key preserves duplicate titles)
		key := str.SanitizeFieldForSorting(t.Name) + "|" + str.SanitizeFieldForSorting(t.Artist)
		if mf, ok := byTitleArtist[key]; ok {
			mfs = append(mfs, mf)
		}
	}
	return mfs
}

// similarityRatio calculates the similarity between two strings using Jaro-Winkler algorithm.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
// Jaro-Winkler is well-suited for matching song titles because it gives higher scores
// when strings share a common prefix (e.g., "Song Title" vs "Song Title - Remastered").
func similarityRatio(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	// JaroWinkler params: boostThreshold=0.7, prefixSize=4
	return smetrics.JaroWinkler(a, b, 0.7, 4)
}

// fuzzyMatchUnmatched performs fuzzy title matching for songs that didn't match exactly.
// It queries tracks by artist and then fuzzy-matches titles within that artist's catalog.
func (e *provider) fuzzyMatchUnmatched(ctx context.Context, unmatched []songQuery, existingMatches map[string]model.MediaFile) (map[string]model.MediaFile, error) {
	// Skip fuzzy matching if threshold is 100 (exact match only)
	threshold := float64(conf.Server.SimilarSongsMatchThreshold) / 100.0
	if threshold >= 1.0 {
		return nil, nil
	}

	// Group unmatched queries by artist
	byArtist := map[string][]songQuery{}
	for _, q := range unmatched {
		if q.artist == "" {
			continue
		}
		// Skip if already matched
		key := q.title + "|" + q.artist
		if _, ok := existingMatches[key]; ok {
			continue
		}
		byArtist[q.artist] = append(byArtist[q.artist], q)
	}

	if len(byArtist) == 0 {
		return nil, nil
	}

	matches := map[string]model.MediaFile{}
	for artist, queries := range byArtist {
		// Query all tracks by this artist
		tracks, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.And{
				squirrel.Eq{"order_artist_name": artist},
				squirrel.Eq{"missing": false},
			},
			Sort: "starred desc, rating desc, year asc, compilation asc",
		})
		if err != nil {
			continue
		}

		// Fuzzy match each query against artist's tracks
		for _, q := range queries {
			if mf, found := e.findFuzzyTitleMatch(q.title, tracks, threshold); found {
				key := q.title + "|" + q.artist
				matches[key] = mf
			}
		}
	}
	return matches, nil
}

// findFuzzyTitleMatch finds the best fuzzy match for a title within a set of tracks.
func (e *provider) findFuzzyTitleMatch(title string, tracks model.MediaFiles, threshold float64) (model.MediaFile, bool) {
	var bestMatch model.MediaFile
	bestRatio := 0.0

	for _, mf := range tracks {
		trackTitle := str.SanitizeFieldForSorting(mf.Title)
		ratio := similarityRatio(title, trackTitle)
		if ratio >= threshold && ratio > bestRatio {
			bestMatch = mf
			bestRatio = ratio
		}
	}
	return bestMatch, bestRatio >= threshold
}
