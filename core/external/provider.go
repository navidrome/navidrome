package external

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/random"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
	"golang.org/x/sync/errgroup"
)

const (
	maxSimilarArtists  = 100
	refreshDelay       = 5 * time.Second
	refreshTimeout     = 15 * time.Second
	refreshQueueLength = 2000
)

type Provider interface {
	UpdateAlbumInfo(ctx context.Context, id string) (*model.Album, error)
	UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error)
	ArtistRadio(ctx context.Context, id string, count int) (model.MediaFiles, error)
	TopSongs(ctx context.Context, artist string, count int) (model.MediaFiles, error)
	ArtistImage(ctx context.Context, id string) (*url.URL, error)
	AlbumImage(ctx context.Context, id string) (*url.URL, error)
}

type provider struct {
	ds          model.DataStore
	ag          Agents
	artistQueue refreshQueue[auxArtist]
	albumQueue  refreshQueue[auxAlbum]
}

type auxAlbum struct {
	model.Album
}

// Name returns the appropriate album name for external API calls
// based on the DevPreserveUnicodeInExternalCalls configuration option
func (a *auxAlbum) Name() string {
	if conf.Server.DevPreserveUnicodeInExternalCalls {
		return a.Album.Name
	}
	return str.Clear(a.Album.Name)
}

type auxArtist struct {
	model.Artist
}

// Name returns the appropriate artist name for external API calls
// based on the DevPreserveUnicodeInExternalCalls configuration option
func (a *auxArtist) Name() string {
	if conf.Server.DevPreserveUnicodeInExternalCalls {
		return a.Artist.Name
	}
	return str.Clear(a.Artist.Name)
}

type Agents interface {
	agents.AlbumInfoRetriever
	agents.AlbumImageRetriever
	agents.ArtistBiographyRetriever
	agents.ArtistMBIDRetriever
	agents.ArtistImageRetriever
	agents.ArtistSimilarRetriever
	agents.ArtistTopSongsRetriever
	agents.ArtistURLRetriever
}

func NewProvider(ds model.DataStore, agents Agents) Provider {
	e := &provider{ds: ds, ag: agents}
	e.artistQueue = newRefreshQueue(context.TODO(), e.populateArtistInfo)
	e.albumQueue = newRefreshQueue(context.TODO(), e.populateAlbumInfo)
	return e
}

func (e *provider) getAlbum(ctx context.Context, id string) (auxAlbum, error) {
	var entity interface{}
	entity, err := model.GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return auxAlbum{}, err
	}

	var album auxAlbum
	switch v := entity.(type) {
	case *model.Album:
		album.Album = *v
	case *model.MediaFile:
		return e.getAlbum(ctx, v.AlbumID)
	default:
		return auxAlbum{}, model.ErrNotFound
	}

	return album, nil
}

func (e *provider) UpdateAlbumInfo(ctx context.Context, id string) (*model.Album, error) {
	album, err := e.getAlbum(ctx, id)
	if err != nil {
		log.Info(ctx, "Not found", "id", id)
		return nil, err
	}

	updatedAt := V(album.ExternalInfoUpdatedAt)
	albumName := album.Name()
	if updatedAt.IsZero() {
		log.Debug(ctx, "AlbumInfo not cached. Retrieving it now", "updatedAt", updatedAt, "id", id, "name", albumName)
		album, err = e.populateAlbumInfo(ctx, album)
		if err != nil {
			return nil, err
		}
	}

	// If info is expired, trigger a populateAlbumInfo in the background
	if time.Since(updatedAt) > conf.Server.DevAlbumInfoTimeToLive {
		log.Debug("Found expired cached AlbumInfo, refreshing in the background", "updatedAt", album.ExternalInfoUpdatedAt, "name", albumName)
		e.albumQueue.enqueue(&album)
	}

	return &album.Album, nil
}

func (e *provider) populateAlbumInfo(ctx context.Context, album auxAlbum) (auxAlbum, error) {
	start := time.Now()
	albumName := album.Name()
	info, err := e.ag.GetAlbumInfo(ctx, albumName, album.AlbumArtist, album.MbzAlbumID)
	if errors.Is(err, agents.ErrNotFound) {
		return album, nil
	}
	if err != nil {
		log.Error("Error refreshing AlbumInfo", "id", album.ID, "name", albumName, "artist", album.AlbumArtist,
			"elapsed", time.Since(start), err)
		return album, err
	}

	album.ExternalInfoUpdatedAt = P(time.Now())
	album.ExternalUrl = info.URL

	if info.Description != "" {
		album.Description = info.Description
	}

	images, err := e.ag.GetAlbumImages(ctx, albumName, album.AlbumArtist, album.MbzAlbumID)
	if err == nil && len(images) > 0 {
		sort.Slice(images, func(i, j int) bool {
			return images[i].Size > images[j].Size
		})

		album.LargeImageUrl = images[0].URL

		if len(images) >= 2 {
			album.MediumImageUrl = images[1].URL
		}

		if len(images) >= 3 {
			album.SmallImageUrl = images[2].URL
		}
	}

	err = e.ds.Album(ctx).UpdateExternalInfo(&album.Album)
	if err != nil {
		log.Error(ctx, "Error trying to update album external information", "id", album.ID, "name", albumName,
			"elapsed", time.Since(start), err)
	} else {
		log.Trace(ctx, "AlbumInfo collected", "album", album, "elapsed", time.Since(start))
	}

	return album, nil
}

func (e *provider) getArtist(ctx context.Context, id string) (auxArtist, error) {
	var entity interface{}
	entity, err := model.GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return auxArtist{}, err
	}

	var artist auxArtist
	switch v := entity.(type) {
	case *model.Artist:
		artist.Artist = *v
	case *model.MediaFile:
		return e.getArtist(ctx, v.ArtistID)
	case *model.Album:
		return e.getArtist(ctx, v.AlbumArtistID)
	default:
		return auxArtist{}, model.ErrNotFound
	}
	return artist, nil
}

func (e *provider) UpdateArtistInfo(ctx context.Context, id string, similarCount int, includeNotPresent bool) (*model.Artist, error) {
	artist, err := e.refreshArtistInfo(ctx, id)
	if err != nil {
		return nil, err
	}

	err = e.loadSimilar(ctx, &artist, similarCount, includeNotPresent)
	return &artist.Artist, err
}

func (e *provider) refreshArtistInfo(ctx context.Context, id string) (auxArtist, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return auxArtist{}, err
	}

	// If we don't have any info, retrieves it now
	updatedAt := V(artist.ExternalInfoUpdatedAt)
	artistName := artist.Name()
	if updatedAt.IsZero() {
		log.Debug(ctx, "ArtistInfo not cached. Retrieving it now", "updatedAt", updatedAt, "id", id, "name", artistName)
		artist, err = e.populateArtistInfo(ctx, artist)
		if err != nil {
			return auxArtist{}, err
		}
	}

	// If info is expired, trigger a populateArtistInfo in the background
	if time.Since(updatedAt) > conf.Server.DevArtistInfoTimeToLive {
		log.Debug("Found expired cached ArtistInfo, refreshing in the background", "updatedAt", updatedAt, "name", artistName)
		e.artistQueue.enqueue(&artist)
	}
	return artist, nil
}

func (e *provider) populateArtistInfo(ctx context.Context, artist auxArtist) (auxArtist, error) {
	start := time.Now()
	// Get MBID first, if it is not yet available
	artistName := artist.Name()
	if artist.MbzArtistID == "" {
		mbid, err := e.ag.GetArtistMBID(ctx, artist.ID, artistName)
		if mbid != "" && err == nil {
			artist.MbzArtistID = mbid
		}
	}

	// Call all registered agents and collect information
	g := errgroup.Group{}
	g.SetLimit(2)
	g.Go(func() error { e.callGetImage(ctx, e.ag, &artist); return nil })
	g.Go(func() error { e.callGetBiography(ctx, e.ag, &artist); return nil })
	g.Go(func() error { e.callGetURL(ctx, e.ag, &artist); return nil })
	g.Go(func() error { e.callGetSimilar(ctx, e.ag, &artist, maxSimilarArtists, true); return nil })
	_ = g.Wait()

	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "ArtistInfo update canceled", "id", artist.ID, "name", artistName, "elapsed", time.Since(start), ctx.Err())
		return artist, ctx.Err()
	}

	artist.ExternalInfoUpdatedAt = P(time.Now())
	err := e.ds.Artist(ctx).UpdateExternalInfo(&artist.Artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artist external information", "id", artist.ID, "name", artistName,
			"elapsed", time.Since(start), err)
	} else {
		log.Trace(ctx, "ArtistInfo collected", "artist", artist, "elapsed", time.Since(start))
	}
	return artist, nil
}

func (e *provider) ArtistRadio(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	e.callGetSimilar(ctx, e.ag, &artist, 15, false)
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "ArtistRadio call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	weightedSongs := random.NewWeightedChooser[model.MediaFile]()
	addArtist := func(a model.Artist, weightedSongs *random.WeightedChooser[model.MediaFile], count, artistWeight int) error {
		if utils.IsCtxDone(ctx) {
			log.Warn(ctx, "ArtistRadio call canceled", ctx.Err())
			return ctx.Err()
		}

		topCount := max(count, 20)
		topSongs, err := e.getMatchingTopSongs(ctx, e.ag, &auxArtist{Artist: a}, topCount)
		if err != nil {
			log.Warn(ctx, "Error getting artist's top songs", "artist", a.Name, err)
			return nil
		}

		weight := topCount * (4 + artistWeight)
		for _, mf := range topSongs {
			weightedSongs.Add(mf, weight)
			weight -= 4
		}
		return nil
	}

	err = addArtist(artist.Artist, weightedSongs, count, 10)
	if err != nil {
		return nil, err
	}
	for _, a := range artist.SimilarArtists {
		err := addArtist(a, weightedSongs, count, 0)
		if err != nil {
			return nil, err
		}
	}

	var similarSongs model.MediaFiles
	for len(similarSongs) < count && weightedSongs.Size() > 0 {
		s, err := weightedSongs.Pick()
		if err != nil {
			log.Warn(ctx, "Error getting weighted song", err)
			continue
		}
		similarSongs = append(similarSongs, s)
	}

	return similarSongs, nil
}

func (e *provider) ArtistImage(ctx context.Context, id string) (*url.URL, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	e.callGetImage(ctx, e.ag, &artist)
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "ArtistImage call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	imageUrl := artist.ArtistImageUrl()
	if imageUrl == "" {
		return nil, model.ErrNotFound
	}
	return url.Parse(imageUrl)
}

func (e *provider) AlbumImage(ctx context.Context, id string) (*url.URL, error) {
	album, err := e.getAlbum(ctx, id)
	if err != nil {
		return nil, err
	}

	albumName := album.Name()
	images, err := e.ag.GetAlbumImages(ctx, albumName, album.AlbumArtist, album.MbzAlbumID)
	if err != nil {
		switch {
		case errors.Is(err, agents.ErrNotFound):
			log.Trace(ctx, "Album not found in agent", "albumID", id, "name", albumName, "artist", album.AlbumArtist)
			return nil, model.ErrNotFound
		case errors.Is(err, context.Canceled):
			log.Debug(ctx, "GetAlbumImages call canceled", err)
		default:
			log.Warn(ctx, "Error getting album images from agent", "albumID", id, "name", albumName, "artist", album.AlbumArtist, err)
		}
		return nil, err
	}

	if len(images) == 0 {
		log.Warn(ctx, "Agent returned no images without error", "albumID", id, "name", albumName, "artist", album.AlbumArtist)
		return nil, model.ErrNotFound
	}

	// Return the biggest image
	var img agents.ExternalImage
	for _, i := range images {
		if img.Size <= i.Size {
			img = i
		}
	}
	if img.URL == "" {
		return nil, model.ErrNotFound
	}
	return url.Parse(img.URL)
}

func (e *provider) TopSongs(ctx context.Context, artistName string, count int) (model.MediaFiles, error) {
	artist, err := e.findArtistByName(ctx, artistName)
	if err != nil {
		log.Error(ctx, "Artist not found", "name", artistName, err)
		return nil, nil
	}

	songs, err := e.getMatchingTopSongs(ctx, e.ag, artist, count)
	if err != nil {
		switch {
		case errors.Is(err, agents.ErrNotFound):
			log.Trace(ctx, "TopSongs not found", "name", artistName)
			return nil, model.ErrNotFound
		case errors.Is(err, context.Canceled):
			log.Debug(ctx, "TopSongs call canceled", err)
		default:
			log.Warn(ctx, "Error getting top songs from agent", "artist", artistName, err)
		}

		return nil, err
	}
	return songs, nil
}

func (e *provider) getMatchingTopSongs(ctx context.Context, agent agents.ArtistTopSongsRetriever, artist *auxArtist, count int) (model.MediaFiles, error) {
	artistName := artist.Name()
	songs, err := agent.GetArtistTopSongs(ctx, artist.ID, artistName, artist.MbzArtistID, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get top songs for artist %s: %w", artistName, err)
	}

	idMatches, err := e.loadTracksByID(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by ID: %w", err)
	}
	mbidMatches, err := e.loadTracksByMBID(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by MBID: %w", err)
	}
	titleMatches, err := e.loadTracksByTitle(ctx, songs, artist, idMatches, mbidMatches)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by title: %w", err)
	}

	log.Trace(ctx, "Top Songs loaded", "name", artistName, "numSongs", len(songs), "numIDMatches", len(idMatches), "numMBIDMatches", len(mbidMatches), "numTitleMatches", len(titleMatches))
	mfs := e.selectTopSongs(songs, idMatches, mbidMatches, titleMatches, count)

	if len(mfs) == 0 {
		log.Debug(ctx, "No matching top songs found", "name", artistName)
	} else {
		log.Debug(ctx, "Found matching top songs", "name", artistName, "numSongs", len(mfs))
	}

	return mfs, nil
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

func (e *provider) loadTracksByTitle(ctx context.Context, songs []agents.Song, artist *auxArtist, idMatches, mbidMatches map[string]model.MediaFile) (map[string]model.MediaFile, error) {
	titleMap := map[string]string{}
	for _, s := range songs {
		// Skip if already matched by ID or MBID
		if s.ID != "" && idMatches[s.ID].ID != "" {
			continue
		}
		if s.MBID != "" && mbidMatches[s.MBID].ID != "" {
			continue
		}
		sanitized := str.SanitizeFieldForSorting(s.Name)
		titleMap[sanitized] = s.Name
	}
	matches := map[string]model.MediaFile{}
	if len(titleMap) == 0 {
		return matches, nil
	}
	titleFilters := squirrel.Or{}
	for sanitized := range titleMap {
		titleFilters = append(titleFilters, squirrel.Like{"order_title": sanitized})
	}

	res, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Or{
				squirrel.Eq{"artist_id": artist.ID},
				squirrel.Eq{"album_artist_id": artist.ID},
			},
			titleFilters,
			squirrel.Eq{"missing": false},
		},
		Sort: "starred desc, rating desc, year asc, compilation asc ",
	})
	if err != nil {
		return matches, err
	}
	for _, mf := range res {
		sanitized := str.SanitizeFieldForSorting(mf.Title)
		if _, ok := matches[sanitized]; !ok {
			matches[sanitized] = mf
		}
	}
	return matches, nil
}

func (e *provider) selectTopSongs(songs []agents.Song, byID, byMBID, byTitle map[string]model.MediaFile, count int) model.MediaFiles {
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
		// Fall back to title match
		if mf, ok := byTitle[str.SanitizeFieldForSorting(t.Name)]; ok {
			mfs = append(mfs, mf)
		}
	}
	return mfs
}

func (e *provider) callGetURL(ctx context.Context, agent agents.ArtistURLRetriever, artist *auxArtist) {
	artisURL, err := agent.GetArtistURL(ctx, artist.ID, artist.Name(), artist.MbzArtistID)
	if err != nil {
		return
	}
	artist.ExternalUrl = artisURL
}

func (e *provider) callGetBiography(ctx context.Context, agent agents.ArtistBiographyRetriever, artist *auxArtist) {
	bio, err := agent.GetArtistBiography(ctx, artist.ID, artist.Name(), artist.MbzArtistID)
	if err != nil {
		return
	}
	bio = str.SanitizeText(bio)
	bio = strings.ReplaceAll(bio, "\n", " ")
	artist.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
}

func (e *provider) callGetImage(ctx context.Context, agent agents.ArtistImageRetriever, artist *auxArtist) {
	images, err := agent.GetArtistImages(ctx, artist.ID, artist.Name(), artist.MbzArtistID)
	if err != nil {
		return
	}
	sort.Slice(images, func(i, j int) bool { return images[i].Size > images[j].Size })

	if len(images) >= 1 {
		artist.LargeImageUrl = images[0].URL
	}
	if len(images) >= 2 {
		artist.MediumImageUrl = images[1].URL
	}
	if len(images) >= 3 {
		artist.SmallImageUrl = images[2].URL
	}
}

func (e *provider) callGetSimilar(ctx context.Context, agent agents.ArtistSimilarRetriever, artist *auxArtist,
	limit int, includeNotPresent bool) {
	artistName := artist.Name()
	similar, err := agent.GetSimilarArtists(ctx, artist.ID, artistName, artist.MbzArtistID, limit)
	if len(similar) == 0 || err != nil {
		return
	}
	start := time.Now()
	sa, err := e.mapSimilarArtists(ctx, similar, limit, includeNotPresent)
	log.Debug(ctx, "Mapped Similar Artists", "artist", artistName, "numSimilar", len(sa), "elapsed", time.Since(start))
	if err != nil {
		return
	}
	artist.SimilarArtists = sa
}

func (e *provider) mapSimilarArtists(ctx context.Context, similar []agents.Artist, limit int, includeNotPresent bool) (model.Artists, error) {
	var result model.Artists
	var notPresent []string

	// Load artists by ID (highest priority)
	idMatches, err := e.loadArtistsByID(ctx, similar)
	if err != nil {
		return nil, err
	}

	// Load artists by MBID (second priority)
	mbidMatches, err := e.loadArtistsByMBID(ctx, similar, idMatches)
	if err != nil {
		return nil, err
	}

	// Load artists by name (lowest priority, fallback)
	nameMatches, err := e.loadArtistsByName(ctx, similar, idMatches, mbidMatches)
	if err != nil {
		return nil, err
	}

	count := 0

	// Process the similar artists using priority: ID → MBID → Name
	for _, s := range similar {
		if count >= limit {
			break
		}
		// Try ID match first
		if s.ID != "" {
			if artist, found := idMatches[s.ID]; found {
				result = append(result, artist)
				count++
				continue
			}
		}
		// Try MBID match second
		if s.MBID != "" {
			if artist, found := mbidMatches[s.MBID]; found {
				result = append(result, artist)
				count++
				continue
			}
		}
		// Fall back to name match
		if artist, found := nameMatches[s.Name]; found {
			result = append(result, artist)
			count++
		} else {
			notPresent = append(notPresent, s.Name)
		}
	}

	// Then fill up with non-present artists
	if includeNotPresent && count < limit {
		for _, s := range notPresent {
			// Let the ID empty to indicate that the artist is not present in the DB
			sa := model.Artist{Name: s}
			result = append(result, sa)

			count++
			if count >= limit {
				break
			}
		}
	}

	return result, nil
}

func (e *provider) loadArtistsByID(ctx context.Context, similar []agents.Artist) (map[string]model.Artist, error) {
	var ids []string
	for _, s := range similar {
		if s.ID != "" {
			ids = append(ids, s.ID)
		}
	}
	matches := map[string]model.Artist{}
	if len(ids) == 0 {
		return matches, nil
	}
	res, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"artist.id": ids},
	})
	if err != nil {
		return matches, err
	}
	for _, a := range res {
		if _, ok := matches[a.ID]; !ok {
			matches[a.ID] = a
		}
	}
	return matches, nil
}

func (e *provider) loadArtistsByMBID(ctx context.Context, similar []agents.Artist, idMatches map[string]model.Artist) (map[string]model.Artist, error) {
	var mbids []string
	for _, s := range similar {
		// Skip if already matched by ID
		if s.ID != "" && idMatches[s.ID].ID != "" {
			continue
		}
		if s.MBID != "" {
			mbids = append(mbids, s.MBID)
		}
	}
	matches := map[string]model.Artist{}
	if len(mbids) == 0 {
		return matches, nil
	}
	res, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"mbz_artist_id": mbids},
	})
	if err != nil {
		return matches, err
	}
	for _, a := range res {
		if id := a.MbzArtistID; id != "" {
			if _, ok := matches[id]; !ok {
				matches[id] = a
			}
		}
	}
	return matches, nil
}

func (e *provider) loadArtistsByName(ctx context.Context, similar []agents.Artist, idMatches map[string]model.Artist, mbidMatches map[string]model.Artist) (map[string]model.Artist, error) {
	var names []string
	for _, s := range similar {
		// Skip if already matched by ID or MBID
		if s.ID != "" && idMatches[s.ID].ID != "" {
			continue
		}
		if s.MBID != "" && mbidMatches[s.MBID].ID != "" {
			continue
		}
		names = append(names, s.Name)
	}
	matches := map[string]model.Artist{}
	if len(names) == 0 {
		return matches, nil
	}
	clauses := slice.Map(names, func(name string) squirrel.Sqlizer {
		return squirrel.Like{"artist.name": name}
	})
	res, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Or(clauses),
	})
	if err != nil {
		return matches, err
	}
	for _, a := range res {
		if _, ok := matches[a.Name]; !ok {
			matches[a.Name] = a
		}
	}
	return matches, nil
}

func (e *provider) findArtistByName(ctx context.Context, artistName string) (*auxArtist, error) {
	artists, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Like{"artist.name": artistName},
		Max:     1,
	})
	if err != nil {
		return nil, err
	}
	if len(artists) == 0 {
		return nil, model.ErrNotFound
	}
	return &auxArtist{Artist: artists[0]}, nil
}

func (e *provider) loadSimilar(ctx context.Context, artist *auxArtist, count int, includeNotPresent bool) error {
	var ids []string
	for _, sa := range artist.SimilarArtists {
		if sa.ID == "" {
			continue
		}
		ids = append(ids, sa.ID)
	}

	similar, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"artist.id": ids},
	})
	if err != nil {
		log.Error("Error loading similar artists", "id", artist.ID, "name", artist.Name(), err)
		return err
	}

	// Use a map and iterate through original array, to keep the same order
	artistMap := make(map[string]model.Artist)
	for _, sa := range similar {
		artistMap[sa.ID] = sa
	}

	var loaded model.Artists
	for _, sa := range artist.SimilarArtists {
		if len(loaded) >= count {
			break
		}
		la, ok := artistMap[sa.ID]
		if !ok {
			if !includeNotPresent {
				continue
			}
			la = sa
			la.ID = ""
		}
		loaded = append(loaded, la)
	}
	artist.SimilarArtists = loaded
	return nil
}

type refreshQueue[T any] chan<- *T

func newRefreshQueue[T any](ctx context.Context, processFn func(context.Context, T) (T, error)) refreshQueue[T] {
	queue := make(chan *T, refreshQueueLength)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(refreshDelay):
				ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
				select {
				case item := <-queue:
					_, _ = processFn(ctx, *item)
					cancel()
				case <-ctx.Done():
					cancel()
				}
			}
		}
	}()
	return queue
}

func (q *refreshQueue[T]) enqueue(item *T) {
	select {
	case *q <- item:
	default: // It is ok to miss a refresh request
	}
}
