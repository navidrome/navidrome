package core

import (
	"context"
	"errors"
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

type ExternalMetadata interface {
	UpdateAlbumInfo(ctx context.Context, id string) (*model.Album, error)
	UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error)
	SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error)
	TopSongs(ctx context.Context, artist string, count int) (model.MediaFiles, error)
	ArtistImage(ctx context.Context, id string) (*url.URL, error)
	AlbumImage(ctx context.Context, id string) (*url.URL, error)
}

type externalMetadata struct {
	ds          model.DataStore
	ag          *agents.Agents
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

func NewExternalMetadata(ds model.DataStore, agents *agents.Agents) ExternalMetadata {
	e := &externalMetadata{ds: ds, ag: agents}
	e.artistQueue = newRefreshQueue(context.TODO(), e.populateArtistInfo)
	e.albumQueue = newRefreshQueue(context.TODO(), e.populateAlbumInfo)
	return e
}

func (e *externalMetadata) getAlbum(ctx context.Context, id string) (auxAlbum, error) {
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

func (e *externalMetadata) UpdateAlbumInfo(ctx context.Context, id string) (*model.Album, error) {
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

func (e *externalMetadata) populateAlbumInfo(ctx context.Context, album auxAlbum) (auxAlbum, error) {
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

func (e *externalMetadata) getArtist(ctx context.Context, id string) (auxArtist, error) {
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

func (e *externalMetadata) UpdateArtistInfo(ctx context.Context, id string, similarCount int, includeNotPresent bool) (*model.Artist, error) {
	artist, err := e.refreshArtistInfo(ctx, id)
	if err != nil {
		return nil, err
	}

	err = e.loadSimilar(ctx, &artist, similarCount, includeNotPresent)
	return &artist.Artist, err
}

func (e *externalMetadata) refreshArtistInfo(ctx context.Context, id string) (auxArtist, error) {
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

func (e *externalMetadata) populateArtistInfo(ctx context.Context, artist auxArtist) (auxArtist, error) {
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

func (e *externalMetadata) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
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

func (e *externalMetadata) ArtistImage(ctx context.Context, id string) (*url.URL, error) {
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
		return nil, agents.ErrNotFound
	}
	return url.Parse(imageUrl)
}

func (e *externalMetadata) AlbumImage(ctx context.Context, id string) (*url.URL, error) {
	album, err := e.getAlbum(ctx, id)
	if err != nil {
		return nil, err
	}

	info, err := e.ag.GetAlbumInfo(ctx, album.Name, album.AlbumArtist, album.MbzAlbumID)
	if errors.Is(err, agents.ErrNotFound) {
		return nil, err
	}
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "AlbumImage call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	// Return the biggest image
	var img agents.ExternalImage
	for _, i := range info.Images {
		if img.Size <= i.Size {
			img = i
		}
	}
	if img.URL == "" {
		return nil, agents.ErrNotFound
	}
	return url.Parse(img.URL)
}

func (e *externalMetadata) TopSongs(ctx context.Context, artistName string, count int) (model.MediaFiles, error) {
	artist, err := e.findArtistByName(ctx, artistName)
	if err != nil {
		log.Error(ctx, "Artist not found", "name", artistName, err)
		return nil, nil
	}

	return e.getMatchingTopSongs(ctx, e.ag, artist, count)
}

func (e *externalMetadata) getMatchingTopSongs(ctx context.Context, agent agents.ArtistTopSongsRetriever, artist *auxArtist, count int) (model.MediaFiles, error) {
	songs, err := agent.GetArtistTopSongs(ctx, artist.ID, artist.Name, artist.MbzArtistID, count)
	if errors.Is(err, agents.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var mfs model.MediaFiles
	for _, t := range songs {
		mf, err := e.findMatchingTrack(ctx, t.MBID, artist.ID, t.Name)
		if err != nil {
			continue
		}
		mfs = append(mfs, *mf)
		if len(mfs) == count {
			break
		}
	}
	if len(mfs) == 0 {
		log.Debug(ctx, "No matching top songs found", "name", artist.Name)
	} else {
		log.Debug(ctx, "Found matching top songs", "name", artist.Name, "numSongs", len(mfs))
	}
	return mfs, nil
}

func (e *externalMetadata) findMatchingTrack(ctx context.Context, mbid string, artistID, title string) (*model.MediaFile, error) {
	if mbid != "" {
		mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.And{
				squirrel.Eq{"mbz_recording_id": mbid},
				squirrel.Eq{"missing": false},
			},
		})
		if err == nil && len(mfs) > 0 {
			return &mfs[0], nil
		}
		return e.findMatchingTrack(ctx, "", artistID, title)
	}
	mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Or{
				squirrel.Eq{"artist_id": artistID},
				squirrel.Eq{"album_artist_id": artistID},
			},
			squirrel.Like{"order_title": str.SanitizeFieldForSorting(title)},
			squirrel.Eq{"missing": false},
		},
		Sort: "starred desc, rating desc, year asc, compilation asc ",
		Max:  1,
	})
	if err != nil || len(mfs) == 0 {
		return nil, model.ErrNotFound
	}
	return &mfs[0], nil
}

func (e *externalMetadata) callGetURL(ctx context.Context, agent agents.ArtistURLRetriever, artist *auxArtist) {
	artisURL, err := agent.GetArtistURL(ctx, artist.ID, artist.Name, artist.MbzArtistID)
	if err != nil {
		return
	}
	artist.ExternalUrl = artisURL
}

func (e *externalMetadata) callGetBiography(ctx context.Context, agent agents.ArtistBiographyRetriever, artist *auxArtist) {
	bio, err := agent.GetArtistBiography(ctx, artist.ID, str.Clear(artist.Name), artist.MbzArtistID)
	if err != nil {
		return
	}
	bio = str.SanitizeText(bio)
	bio = strings.ReplaceAll(bio, "\n", " ")
	artist.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
}

func (e *externalMetadata) callGetImage(ctx context.Context, agent agents.ArtistImageRetriever, artist *auxArtist) {
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

func (e *externalMetadata) callGetSimilar(ctx context.Context, agent agents.ArtistSimilarRetriever, artist *auxArtist,
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

func (e *externalMetadata) mapSimilarArtists(ctx context.Context, similar []agents.Artist, includeNotPresent bool) (model.Artists, error) {
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

func (e *externalMetadata) findArtistByName(ctx context.Context, artistName string) (*auxArtist, error) {
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

func (e *externalMetadata) loadSimilar(ctx context.Context, artist *auxArtist, count int, includeNotPresent bool) error {
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
