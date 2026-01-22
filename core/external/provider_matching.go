package external

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
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

	indices := e.indexTracksByKeys(res)
	return e.matchQueriesAgainstIndices(queries, indices), nil
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

type trackIndices struct {
	byTitleArtistMBIDAlbumMBID map[string]model.MediaFile
	byTitleArtistMBIDAlbum     map[string]model.MediaFile
	byTitleArtistAlbum         map[string]model.MediaFile
	byTitleArtistMBID          map[string]model.MediaFile
	byTitleArtist              map[string]model.MediaFile
	byTitle                    map[string]model.MediaFile
}

func (e *provider) indexTracksByKeys(tracks model.MediaFiles) *trackIndices {
	indices := &trackIndices{
		byTitleArtistMBIDAlbumMBID: map[string]model.MediaFile{},
		byTitleArtistMBIDAlbum:     map[string]model.MediaFile{},
		byTitleArtistAlbum:         map[string]model.MediaFile{},
		byTitleArtistMBID:          map[string]model.MediaFile{},
		byTitleArtist:              map[string]model.MediaFile{},
		byTitle:                    map[string]model.MediaFile{},
	}

	for _, mf := range tracks {
		title := str.SanitizeFieldForSorting(mf.Title)
		artist := str.SanitizeFieldForSorting(mf.Artist)
		album := str.SanitizeFieldForSorting(mf.Album)
		artistMBID := mf.MbzArtistID
		albumMBID := mf.MbzAlbumID

		e.indexTrackBySpecificity(indices, title, artist, album, artistMBID, albumMBID, mf)
	}
	return indices
}

func (e *provider) indexTrackBySpecificity(indices *trackIndices, title, artist, album, artistMBID, albumMBID string, mf model.MediaFile) {
	addIfNotExists := func(m map[string]model.MediaFile, key string) {
		if _, ok := m[key]; !ok {
			m[key] = mf
		}
	}

	if artistMBID != "" && albumMBID != "" {
		addIfNotExists(indices.byTitleArtistMBIDAlbumMBID, title+"|"+artistMBID+"|"+albumMBID)
	}
	if artistMBID != "" && album != "" {
		addIfNotExists(indices.byTitleArtistMBIDAlbum, title+"|"+artistMBID+"|"+album)
	}
	if artist != "" && album != "" {
		addIfNotExists(indices.byTitleArtistAlbum, title+"|"+artist+"|"+album)
	}
	if artistMBID != "" {
		addIfNotExists(indices.byTitleArtistMBID, title+"|"+artistMBID)
	}
	if artist != "" {
		addIfNotExists(indices.byTitleArtist, title+"|"+artist)
	}
	addIfNotExists(indices.byTitle, title)
}

func (e *provider) matchQueriesAgainstIndices(queries []songQuery, indices *trackIndices) map[string]model.MediaFile {
	matches := map[string]model.MediaFile{}

	for _, q := range queries {
		if mf, found := e.findBestMatch(q, indices); found {
			// Use composite key (title+artist) to preserve matches for duplicate titles
			key := q.title + "|" + q.artist
			if _, ok := matches[key]; !ok {
				matches[key] = mf
			}
		}
	}
	return matches
}

func (e *provider) findBestMatch(q songQuery, indices *trackIndices) (model.MediaFile, bool) {
	// Try most specific matches first
	lookupFuncs := []func() (model.MediaFile, bool){
		func() (model.MediaFile, bool) {
			if q.artistMBID != "" && q.albumMBID != "" {
				mf, ok := indices.byTitleArtistMBIDAlbumMBID[q.title+"|"+q.artistMBID+"|"+q.albumMBID]
				return mf, ok
			}
			return model.MediaFile{}, false
		},
		func() (model.MediaFile, bool) {
			if q.artistMBID != "" && q.album != "" {
				mf, ok := indices.byTitleArtistMBIDAlbum[q.title+"|"+q.artistMBID+"|"+q.album]
				return mf, ok
			}
			return model.MediaFile{}, false
		},
		func() (model.MediaFile, bool) {
			if q.artist != "" && q.album != "" {
				mf, ok := indices.byTitleArtistAlbum[q.title+"|"+q.artist+"|"+q.album]
				return mf, ok
			}
			return model.MediaFile{}, false
		},
		func() (model.MediaFile, bool) {
			if q.artistMBID != "" {
				mf, ok := indices.byTitleArtistMBID[q.title+"|"+q.artistMBID]
				return mf, ok
			}
			return model.MediaFile{}, false
		},
		func() (model.MediaFile, bool) {
			if q.artist != "" {
				mf, ok := indices.byTitleArtist[q.title+"|"+q.artist]
				return mf, ok
			}
			return model.MediaFile{}, false
		},
		func() (model.MediaFile, bool) {
			mf, ok := indices.byTitle[q.title]
			return mf, ok
		},
	}

	for _, lookup := range lookupFuncs {
		if mf, found := lookup(); found {
			return mf, true
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
