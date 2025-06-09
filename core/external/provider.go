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
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
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
	SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error)
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
	Name string
}

type auxArtist struct {
	model.Artist
	Name string
}

type Agents interface {
	agents.AlbumInfoRetriever
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
		album.Name = str.Clear(v.Name)
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
	if updatedAt.IsZero() {
		log.Debug(ctx, "AlbumInfo not cached. Retrieving it now", "updatedAt", updatedAt, "id", id, "name", album.Name)
		album, err = e.populateAlbumInfo(ctx, album)
		if err != nil {
			return nil, err
		}
	}

	// If info is expired, trigger a populateAlbumInfo in the background
	if time.Since(updatedAt) > conf.Server.DevAlbumInfoTimeToLive {
		log.Debug("Found expired cached AlbumInfo, refreshing in the background", "updatedAt", album.ExternalInfoUpdatedAt, "name", album.Name)
		e.albumQueue.enqueue(&album)
	}

	return &album.Album, nil
}

func (e *provider) populateAlbumInfo(ctx context.Context, album auxAlbum) (auxAlbum, error) {
	start := time.Now()
	info, err := e.ag.GetAlbumInfo(ctx, album.Name, album.AlbumArtist, album.MbzAlbumID)
	if errors.Is(err, agents.ErrNotFound) {
		return album, nil
	}
	if err != nil {
		log.Error("Error refreshing AlbumInfo", "id", album.ID, "name", album.Name, "artist", album.AlbumArtist,
			"elapsed", time.Since(start), err)
		return album, err
	}

	album.ExternalInfoUpdatedAt = P(time.Now())
	album.ExternalUrl = info.URL

	if info.Description != "" {
		album.Description = info.Description
	}

	if len(info.Images) > 0 {
		sort.Slice(info.Images, func(i, j int) bool {
			return info.Images[i].Size > info.Images[j].Size
		})

		album.LargeImageUrl = info.Images[0].URL

		if len(info.Images) >= 2 {
			album.MediumImageUrl = info.Images[1].URL
		}

		if len(info.Images) >= 3 {
			album.SmallImageUrl = info.Images[2].URL
		}
	}

	err = e.ds.Album(ctx).UpdateExternalInfo(&album.Album)
	if err != nil {
		log.Error(ctx, "Error trying to update album external information", "id", album.ID, "name", album.Name,
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
		artist.Name = str.Clear(v.Name)
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
	if updatedAt.IsZero() {
		log.Debug(ctx, "ArtistInfo not cached. Retrieving it now", "updatedAt", updatedAt, "id", id, "name", artist.Name)
		artist, err = e.populateArtistInfo(ctx, artist)
		if err != nil {
			return auxArtist{}, err
		}
	}

	// If info is expired, trigger a populateArtistInfo in the background
	if time.Since(updatedAt) > conf.Server.DevArtistInfoTimeToLive {
		log.Debug("Found expired cached ArtistInfo, refreshing in the background", "updatedAt", updatedAt, "name", artist.Name)
		e.artistQueue.enqueue(&artist)
	}
	return artist, nil
}

func (e *provider) populateArtistInfo(ctx context.Context, artist auxArtist) (auxArtist, error) {
	start := time.Now()
	// Get MBID first, if it is not yet available
	if artist.MbzArtistID == "" {
		mbid, err := e.ag.GetArtistMBID(ctx, artist.ID, artist.Name)
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
		log.Warn(ctx, "ArtistInfo update canceled", "elapsed", "id", artist.ID, "name", artist.Name, time.Since(start), ctx.Err())
		return artist, ctx.Err()
	}

	artist.ExternalInfoUpdatedAt = P(time.Now())
	err := e.ds.Artist(ctx).UpdateExternalInfo(&artist.Artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artist external information", "id", artist.ID, "name", artist.Name,
			"elapsed", time.Since(start), err)
	} else {
		log.Trace(ctx, "ArtistInfo collected", "artist", artist, "elapsed", time.Since(start))
	}
	return artist, nil
}

func (e *provider) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	e.callGetSimilar(ctx, e.ag, &artist, 15, false)
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	weightedSongs := random.NewWeightedChooser[model.MediaFile]()
	addArtist := func(a model.Artist, weightedSongs *random.WeightedChooser[model.MediaFile], count, artistWeight int) error {
		if utils.IsCtxDone(ctx) {
			log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
			return ctx.Err()
		}

		topCount := max(count, 20)
		topSongs, err := e.getMatchingTopSongs(ctx, e.ag, &auxArtist{Name: a.Name, Artist: a}, topCount)
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

	info, err := e.ag.GetAlbumInfo(ctx, album.Name, album.AlbumArtist, album.MbzAlbumID)
	if err != nil {
		switch {
		case errors.Is(err, agents.ErrNotFound):
			log.Trace(ctx, "Album not found in agent", "albumID", id, "name", album.Name, "artist", album.AlbumArtist)
			return nil, model.ErrNotFound
		case errors.Is(err, context.Canceled):
			log.Debug(ctx, "GetAlbumInfo call canceled", err)
		default:
			log.Warn(ctx, "Error getting album info from agent", "albumID", id, "name", album.Name, "artist", album.AlbumArtist, err)
		}

		return nil, err
	}

	if info == nil {
		log.Warn(ctx, "Agent returned nil info without error", "albumID", id, "name", album.Name, "artist", album.AlbumArtist)
		return nil, model.ErrNotFound
	}

	// Return the biggest image
	var img agents.ExternalImage
	for _, i := range info.Images {
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
	songs, err := agent.GetArtistTopSongs(ctx, artist.ID, artist.Name, artist.MbzArtistID, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get top songs for artist %s: %w", artist.Name, err)
	}

	mbidMatches, err := e.loadTracksByMBID(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by MBID: %w", err)
	}
	titleMatches, err := e.loadTracksByTitle(ctx, songs, artist, mbidMatches)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks by title: %w", err)
	}

	log.Trace(ctx, "Top Songs loaded", "name", artist.Name, "numSongs", len(songs), "numMBIDMatches", len(mbidMatches), "numTitleMatches", len(titleMatches))
	mfs := e.selectTopSongs(songs, mbidMatches, titleMatches, count)

	if len(mfs) == 0 {
		log.Debug(ctx, "No matching top songs found", "name", artist.Name)
	} else {
		log.Debug(ctx, "Found matching top songs", "name", artist.Name, "numSongs", len(mfs))
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

func (e *provider) loadTracksByTitle(ctx context.Context, songs []agents.Song, artist *auxArtist, mbidMatches map[string]model.MediaFile) (map[string]model.MediaFile, error) {
	titleMap := map[string]string{}
	for _, s := range songs {
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

func (e *provider) selectTopSongs(songs []agents.Song, byMBID, byTitle map[string]model.MediaFile, count int) model.MediaFiles {
	var mfs model.MediaFiles
	for _, t := range songs {
		if len(mfs) == count {
			break
		}
		if t.MBID != "" {
			if mf, ok := byMBID[t.MBID]; ok {
				mfs = append(mfs, mf)
				continue
			}
		}
		if mf, ok := byTitle[str.SanitizeFieldForSorting(t.Name)]; ok {
			mfs = append(mfs, mf)
		}
	}
	return mfs
}

func (e *provider) callGetURL(ctx context.Context, agent agents.ArtistURLRetriever, artist *auxArtist) {
	artisURL, err := agent.GetArtistURL(ctx, artist.ID, artist.Name, artist.MbzArtistID)
	if err != nil {
		return
	}
	artist.ExternalUrl = artisURL
}

func (e *provider) callGetBiography(ctx context.Context, agent agents.ArtistBiographyRetriever, artist *auxArtist) {
	bio, err := agent.GetArtistBiography(ctx, artist.ID, str.Clear(artist.Name), artist.MbzArtistID)
	if err != nil {
		return
	}
	bio = str.SanitizeText(bio)
	bio = strings.ReplaceAll(bio, "\n", " ")
	artist.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
}

func (e *provider) callGetImage(ctx context.Context, agent agents.ArtistImageRetriever, artist *auxArtist) {
	images, err := agent.GetArtistImages(ctx, artist.ID, artist.Name, artist.MbzArtistID)
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
	similar, err := agent.GetSimilarArtists(ctx, artist.ID, artist.Name, artist.MbzArtistID, limit)
	if len(similar) == 0 || err != nil {
		return
	}
	start := time.Now()
	sa, err := e.mapSimilarArtists(ctx, similar, includeNotPresent)
	log.Debug(ctx, "Mapped Similar Artists", "artist", artist.Name, "numSimilar", len(sa), "elapsed", time.Since(start))
	if err != nil {
		return
	}
	artist.SimilarArtists = sa
}

func (e *provider) mapSimilarArtists(ctx context.Context, similar []agents.Artist, includeNotPresent bool) (model.Artists, error) {
	var result model.Artists
	var notPresent []string

	artistNames := slice.Map(similar, func(artist agents.Artist) string { return artist.Name })

	// Query all artists at once
	clauses := slice.Map(artistNames, func(name string) squirrel.Sqlizer {
		return squirrel.Like{"artist.name": name}
	})
	artists, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Or(clauses),
	})
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	artistMap := make(map[string]model.Artist)
	for _, artist := range artists {
		artistMap[artist.Name] = artist
	}

	// Process the similar artists
	for _, s := range similar {
		if artist, found := artistMap[s.Name]; found {
			result = append(result, artist)
		} else {
			notPresent = append(notPresent, s.Name)
		}
	}

	// Then fill up with non-present artists
	if includeNotPresent {
		for _, s := range notPresent {
			// Let the ID empty to indicate that the artist is not present in the DB
			sa := model.Artist{Name: s}
			result = append(result, sa)
		}
	}

	return result, nil
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
	artist := &auxArtist{
		Artist: artists[0],
		Name:   str.Clear(artists[0].Name),
	}
	return artist, nil
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
		log.Error("Error loading similar artists", "id", artist.ID, "name", artist.Name, err)
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
