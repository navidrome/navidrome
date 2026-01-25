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

// matchSongsToLibrary matches agent song results to local library tracks using a multi-phase
// matching algorithm that prioritizes accuracy over recall.
//
// # Algorithm Overview
//
// The algorithm matches songs from external agents (Last.fm, Deezer, etc.) to tracks in the
// local music library using three matching strategies in priority order:
//
//  1. Direct ID match: Songs with an ID field are matched directly to MediaFiles by ID
//  2. MusicBrainz Recording ID (MBID) match: Songs with MBID are matched to tracks with
//     matching mbz_recording_id
//  3. Title+Artist fuzzy match: Remaining songs are matched using fuzzy string comparison
//     with metadata specificity scoring
//
// # Matching Priority
//
// When selecting the final result, matches are prioritized in order: ID > MBID > Title+Artist.
// This ensures that more reliable identifiers take precedence over fuzzy text matching.
//
// # Fuzzy Matching Details
//
// For title+artist matching, the algorithm uses Jaro-Winkler similarity (threshold configurable
// via SimilarSongsMatchThreshold, default 85%). Matches are ranked by:
//
//  1. Title similarity (Jaro-Winkler score, 0.0-1.0)
//  2. Specificity level (0-5, based on metadata precision):
//     - Level 5: Title + Artist MBID + Album MBID (most specific)
//     - Level 4: Title + Artist MBID + Album name (fuzzy)
//     - Level 3: Title + Artist name + Album name (fuzzy)
//     - Level 2: Title + Artist MBID
//     - Level 1: Title + Artist name
//     - Level 0: Title only
//  3. Album similarity (Jaro-Winkler, as final tiebreaker)
//
// # Examples
//
// Example 1 - MBID Priority:
//
//	Agent returns: {Name: "Paranoid Android", MBID: "abc-123", Artist: "Radiohead"}
//	Library has: [
//	  {ID: "t1", Title: "Paranoid Android", MbzRecordingID: "abc-123"},
//	  {ID: "t2", Title: "Paranoid Android", Artist: "Radiohead"},
//	]
//	Result: t1 (MBID match takes priority over title+artist)
//
// Example 2 - Specificity Ranking:
//
//	Agent returns: {Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"}
//	Library has: [
//	  {ID: "t1", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101"},           // Level 1
//	  {ID: "t2", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"},      // Level 3
//	]
//	Result: t2 (Level 3 beats Level 1 due to album match)
//
// Example 3 - Fuzzy Title Matching:
//
//	Agent returns: {Name: "Bohemian Rhapsody", Artist: "Queen"}
//	Library has: {ID: "t1", Title: "Bohemian Rhapsody - Remastered", Artist: "Queen"}
//	With threshold=85%: Match succeeds (similarity ~0.87)
//	With threshold=100%: No match (not exact)
//
// # Parameters
//
//   - ctx: Context for database operations
//   - songs: Slice of agent.Song results from external providers
//   - count: Maximum number of matches to return
//
// # Returns
//
// Returns up to 'count' MediaFiles from the library that best match the input songs,
// preserving the original order from the agent. Songs that cannot be matched are skipped.
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

// loadTracksByID fetches MediaFiles from the library using direct ID matching.
// It extracts all non-empty ID fields from the input songs and performs a single
// batch query to the database. Returns a map keyed by MediaFile ID for O(1) lookup.
// Only non-missing files are returned.
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

// loadTracksByMBID fetches MediaFiles from the library using MusicBrainz Recording IDs.
// It extracts all non-empty MBID fields from the input songs and performs a single
// batch query against the mbz_recording_id column. Returns a map keyed by MBID for
// O(1) lookup. Only non-missing files are returned.
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

// songQuery represents a normalized query for matching a song to library tracks.
// All string fields are sanitized (lowercased, diacritics removed) for comparison.
// This struct is used internally by loadTracksByTitleAndArtist to group queries by artist.
type songQuery struct {
	title      string // Sanitized song title
	artist     string // Sanitized artist name (without articles like "The")
	artistMBID string // MusicBrainz Artist ID (optional, for higher specificity matching)
	album      string // Sanitized album name (optional, for specificity scoring)
	albumMBID  string // MusicBrainz Album ID (optional, for highest specificity matching)
}

// matchScore combines title/album similarity with metadata specificity for ranking matches
type matchScore struct {
	titleSimilarity  float64 // 0.0-1.0 (Jaro-Winkler)
	albumSimilarity  float64 // 0.0-1.0 (Jaro-Winkler), used as tiebreaker
	specificityLevel int     // 0-5 (higher = more specific metadata match)
}

// betterThan returns true if this score beats another.
// Comparison order: title similarity > specificity level > album similarity
func (s matchScore) betterThan(other matchScore) bool {
	if s.titleSimilarity != other.titleSimilarity {
		return s.titleSimilarity > other.titleSimilarity
	}
	if s.specificityLevel != other.specificityLevel {
		return s.specificityLevel > other.specificityLevel
	}
	return s.albumSimilarity > other.albumSimilarity
}

// computeSpecificityLevel determines how well query metadata matches a track (0-5).
// Higher values indicate more specific matches (MBIDs > names > title only).
// Uses fuzzy matching for album names with the same threshold as title matching.
func computeSpecificityLevel(q songQuery, mf model.MediaFile, albumThreshold float64) int {
	title := str.SanitizeFieldForSorting(mf.Title)
	artist := str.SanitizeFieldForSortingNoArticle(mf.Artist)
	album := str.SanitizeFieldForSorting(mf.Album)

	// Level 5: Title + Artist MBID + Album MBID (most specific)
	if q.artistMBID != "" && q.albumMBID != "" &&
		mf.MbzArtistID == q.artistMBID && mf.MbzAlbumID == q.albumMBID {
		return 5
	}
	// Level 4: Title + Artist MBID + Album name (fuzzy)
	if q.artistMBID != "" && q.album != "" &&
		mf.MbzArtistID == q.artistMBID && similarityRatio(album, q.album) >= albumThreshold {
		return 4
	}
	// Level 3: Title + Artist name + Album name (fuzzy)
	if q.artist != "" && q.album != "" &&
		artist == q.artist && similarityRatio(album, q.album) >= albumThreshold {
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

// findBestMatch finds the best matching track using combined title/album similarity and specificity scoring.
// A track must meet the threshold for title similarity, then the best match is chosen by:
// 1. Highest title similarity
// 2. Highest specificity level
// 3. Highest album similarity (as final tiebreaker)
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

		// Compute album similarity for tiebreaking (0.0 if no album in query)
		var albumSim float64
		if q.album != "" {
			trackAlbum := str.SanitizeFieldForSorting(mf.Album)
			albumSim = similarityRatio(q.album, trackAlbum)
		}

		score := matchScore{
			titleSimilarity:  titleSim,
			albumSimilarity:  albumSim,
			specificityLevel: computeSpecificityLevel(q, mf, threshold),
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
