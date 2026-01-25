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

// matchScore combines title similarity with metadata specificity for ranking matches
type matchScore struct {
	titleSimilarity  float64 // 0.0-1.0 (Jaro-Winkler)
	specificityLevel int     // 0-5 (higher = more specific metadata match)
}

// betterThan returns true if this score beats another.
// Primary comparison: title similarity. Secondary: specificity level.
func (s matchScore) betterThan(other matchScore) bool {
	if s.titleSimilarity != other.titleSimilarity {
		return s.titleSimilarity > other.titleSimilarity
	}
	return s.specificityLevel > other.specificityLevel
}

// computeSpecificityLevel determines how well query metadata matches a track (0-5).
// Higher values indicate more specific matches (MBIDs > names > title only).
func computeSpecificityLevel(q songQuery, mf model.MediaFile) int {
	title := str.SanitizeFieldForSorting(mf.Title)
	artist := str.SanitizeFieldForSortingNoArticle(mf.Artist)
	album := str.SanitizeFieldForSorting(mf.Album)

	// Level 5: Title + Artist MBID + Album MBID (most specific)
	if q.artistMBID != "" && q.albumMBID != "" &&
		mf.MbzArtistID == q.artistMBID && mf.MbzAlbumID == q.albumMBID {
		return 5
	}
	// Level 4: Title + Artist MBID + Album name
	if q.artistMBID != "" && q.album != "" &&
		mf.MbzArtistID == q.artistMBID && album == q.album {
		return 4
	}
	// Level 3: Title + Artist name + Album name
	if q.artist != "" && q.album != "" &&
		artist == q.artist && album == q.album {
		return 3
	}
	// Level 2: Title + Artist MBID
	if q.artistMBID != "" && mf.MbzArtistID == q.artistMBID {
		return 2
	}
	// Level 1: Title + Artist name
	if q.artist != "" && artist == q.artist {
		return 1
	}
	// Level 0: Title only match (but for fuzzy, title matched via similarity)
	// Check if at least the title matches exactly
	if title == q.title {
		return 0
	}
	return -1 // No exact title match, but could still be a fuzzy match
}

// loadTracksByTitleAndArtist loads tracks matching by title with optional artist/album filtering.
// Uses a unified scoring approach that combines title similarity (Jaro-Winkler) with
// metadata specificity (MBIDs, album names) for both exact and fuzzy matches.
// Returns a map keyed by "title|artist" for compatibility with selectBestMatchingSongs.
func (e *provider) loadTracksByTitleAndArtist(ctx context.Context, songs []agents.Song, idMatches, mbidMatches map[string]model.MediaFile) (map[string]model.MediaFile, error) {
	queries := e.buildTitleQueries(songs, idMatches, mbidMatches)
	if len(queries) == 0 {
		return map[string]model.MediaFile{}, nil
	}

	threshold := float64(conf.Server.SimilarSongsMatchThreshold) / 100.0

	// Group queries by artist for efficient DB access
	byArtist := map[string][]songQuery{}
	for _, q := range queries {
		if q.artist != "" {
			byArtist[q.artist] = append(byArtist[q.artist], q)
		}
	}

	matches := map[string]model.MediaFile{}
	for artist, artistQueries := range byArtist {
		// Single DB query per artist - get all their tracks
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

		// Find best match for each query using unified scoring
		for _, q := range artistQueries {
			if mf, found := e.findBestMatch(q, tracks, threshold); found {
				key := q.title + "|" + q.artist
				if _, exists := matches[key]; !exists {
					matches[key] = mf
				}
			}
		}
	}
	return matches, nil
}

// findBestMatch finds the best matching track using combined title similarity and specificity scoring.
// A track must meet the threshold for title similarity, then the best match is chosen by:
// 1. Highest title similarity
// 2. Highest specificity level (as tiebreaker)
func (e *provider) findBestMatch(q songQuery, tracks model.MediaFiles, threshold float64) (model.MediaFile, bool) {
	var bestMatch model.MediaFile
	bestScore := matchScore{titleSimilarity: -1}
	found := false

	for _, mf := range tracks {
		trackTitle := str.SanitizeFieldForSorting(mf.Title)
		titleSim := similarityRatio(q.title, trackTitle)

		if titleSim < threshold {
			continue
		}

		score := matchScore{
			titleSimilarity:  titleSim,
			specificityLevel: computeSpecificityLevel(q, mf),
		}

		if score.betterThan(bestScore) {
			bestScore = score
			bestMatch = mf
			found = true
		}
	}
	return bestMatch, found
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
			artist:     str.SanitizeFieldForSortingNoArticle(s.Artist),
			artistMBID: s.ArtistMBID,
			album:      str.SanitizeFieldForSorting(s.Album),
			albumMBID:  s.AlbumMBID,
		})
	}
	return queries
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
		key := str.SanitizeFieldForSorting(t.Name) + "|" + str.SanitizeFieldForSortingNoArticle(t.Artist)
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
