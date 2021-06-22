package core

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const (
	unavailableArtistID = "-1"
	maxSimilarArtists   = 100
)

type ExternalMetadata interface {
	UpdateArtistInfo(ctx context.Context, id string, count int, includeNotPresent bool) (*model.Artist, error)
	SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error)
	TopSongs(ctx context.Context, artist string, count int) (model.MediaFiles, error)
}

type externalMetadata struct {
	ds model.DataStore
	ag *agents.Agents
}

type auxArtist struct {
	model.Artist
	Name string
}

func NewExternalMetadata(ds model.DataStore, agents *agents.Agents) ExternalMetadata {
	return &externalMetadata{ds: ds, ag: agents}
}

func (e *externalMetadata) getArtist(ctx context.Context, id string) (*auxArtist, error) {
	var entity interface{}
	entity, err := GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return nil, err
	}

	var artist auxArtist
	switch v := entity.(type) {
	case *model.Artist:
		artist.Artist = *v
		artist.Name = clearName(v.Name)
	case *model.MediaFile:
		return e.getArtist(ctx, v.ArtistID)
	case *model.Album:
		return e.getArtist(ctx, v.AlbumArtistID)
	default:
		return nil, model.ErrNotFound
	}
	return &artist, nil
}

// Replace some Unicode chars with their equivalent ASCII
func clearName(name string) string {
	name = strings.ReplaceAll(name, "–", "-")
	name = strings.ReplaceAll(name, "‐", "-")
	name = strings.ReplaceAll(name, "“", `"`)
	name = strings.ReplaceAll(name, "”", `"`)
	name = strings.ReplaceAll(name, "‘", `'`)
	name = strings.ReplaceAll(name, "’", `'`)
	return name
}

func (e *externalMetadata) UpdateArtistInfo(ctx context.Context, id string, similarCount int, includeNotPresent bool) (*model.Artist, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	// If we have fresh info, just return it and trigger a refresh in the background
	if time.Since(artist.ExternalInfoUpdatedAt) < consts.ArtistInfoTimeToLive {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			err := e.refreshArtistInfo(ctx, artist)
			if err != nil {
				log.Error("Error refreshing ArtistInfo", "id", id, "name", artist.Name, err)
			}
		}()
		log.Debug("Found cached ArtistInfo, refreshing in the background", "updatedAt", artist.ExternalInfoUpdatedAt, "name", artist.Name)
		err := e.loadSimilar(ctx, artist, similarCount, includeNotPresent)
		return &artist.Artist, err
	}

	log.Debug(ctx, "ArtistInfo not cached or expired", "updatedAt", artist.ExternalInfoUpdatedAt, "id", id, "name", artist.Name)
	err = e.refreshArtistInfo(ctx, artist)
	if err != nil {
		return nil, err
	}
	err = e.loadSimilar(ctx, artist, similarCount, includeNotPresent)
	return &artist.Artist, err
}

func (e *externalMetadata) refreshArtistInfo(ctx context.Context, artist *auxArtist) error {
	// Get MBID first, if it is not yet available
	if artist.MbzArtistID == "" {
		mbid, err := e.ag.GetMBID(ctx, artist.ID, artist.Name)
		if mbid != "" && err == nil {
			artist.MbzArtistID = mbid
		}
	}

	// Call all registered agents and collect information
	callParallel([]func(){
		func() { e.callGetBiography(ctx, e.ag, artist) },
		func() { e.callGetURL(ctx, e.ag, artist) },
		func() { e.callGetImage(ctx, e.ag, artist) },
		func() { e.callGetSimilar(ctx, e.ag, artist, maxSimilarArtists, true) },
	})

	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "ArtistInfo update canceled", ctx.Err())
		return ctx.Err()
	}

	artist.ExternalInfoUpdatedAt = time.Now()
	err := e.ds.Artist(ctx).Put(&artist.Artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artist external information", "id", artist.ID, "name", artist.Name, err)
	}

	log.Trace(ctx, "ArtistInfo collected", "artist", artist)
	return nil
}

func callParallel(fs []func()) {
	wg := &sync.WaitGroup{}
	wg.Add(len(fs))
	for _, f := range fs {
		go func(f func()) {
			f()
			wg.Done()
		}(f)
	}
	wg.Wait()
}

func (e *externalMetadata) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	e.callGetSimilar(ctx, e.ag, artist, 15, false)
	if utils.IsCtxDone(ctx) {
		log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
		return nil, ctx.Err()
	}

	artists := model.Artists{artist.Artist}
	artists = append(artists, artist.SimilarArtists...)

	weightedSongs := utils.NewWeightedRandomChooser()
	for _, a := range artists {
		if utils.IsCtxDone(ctx) {
			log.Warn(ctx, "SimilarSongs call canceled", ctx.Err())
			return nil, ctx.Err()
		}

		topCount := utils.MaxInt(count, 20)
		topSongs, err := e.getMatchingTopSongs(ctx, e.ag, &auxArtist{Name: a.Name, Artist: a}, topCount)
		if err != nil {
			log.Warn(ctx, "Error getting artist's top songs", "artist", a.Name, err)
			continue
		}

		weight := topCount * 4
		for _, mf := range topSongs {
			weightedSongs.Put(mf, weight)
			weight -= 4
		}
	}

	var similarSongs model.MediaFiles
	for len(similarSongs) < count && weightedSongs.Size() > 0 {
		s, err := weightedSongs.GetAndRemove()
		if err != nil {
			log.Warn(ctx, "Error getting weighted song", err)
			continue
		}
		similarSongs = append(similarSongs, s.(model.MediaFile))
	}

	return similarSongs, nil
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
	songs, err := agent.GetTopSongs(ctx, artist.ID, artist.Name, artist.MbzArtistID, count)
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
	return mfs, nil
}

func (e *externalMetadata) findMatchingTrack(ctx context.Context, mbid string, artistID, title string) (*model.MediaFile, error) {
	if mbid != "" {
		mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"mbz_track_id": mbid},
		})
		if err == nil && len(mfs) > 0 {
			return &mfs[0], nil
		}
	}
	mfs, err := e.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Or{
				squirrel.Eq{"artist_id": artistID},
				squirrel.Eq{"album_artist_id": artistID},
			},
			squirrel.Like{"title": title},
		},
		Sort: "starred desc, rating desc, year asc",
	})
	if err != nil || len(mfs) == 0 {
		return nil, model.ErrNotFound
	}
	return &mfs[0], nil
}

func (e *externalMetadata) callGetURL(ctx context.Context, agent agents.ArtistURLRetriever, artist *auxArtist) {
	url, err := agent.GetURL(ctx, artist.ID, artist.Name, artist.MbzArtistID)
	if url == "" || err != nil {
		return
	}
	artist.ExternalUrl = url
}

func (e *externalMetadata) callGetBiography(ctx context.Context, agent agents.ArtistBiographyRetriever, artist *auxArtist) {
	bio, err := agent.GetBiography(ctx, artist.ID, clearName(artist.Name), artist.MbzArtistID)
	if bio == "" || err != nil {
		return
	}
	policy := bluemonday.UGCPolicy()
	bio = policy.Sanitize(bio)
	bio = strings.ReplaceAll(bio, "\n", " ")
	artist.Biography = strings.ReplaceAll(bio, "<a ", "<a target='_blank' ")
}

func (e *externalMetadata) callGetImage(ctx context.Context, agent agents.ArtistImageRetriever, artist *auxArtist) {
	images, err := agent.GetImages(ctx, artist.ID, artist.Name, artist.MbzArtistID)
	if len(images) == 0 || err != nil {
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
	similar, err := agent.GetSimilar(ctx, artist.ID, artist.Name, artist.MbzArtistID, limit)
	if len(similar) == 0 || err != nil {
		return
	}
	sa, err := e.mapSimilarArtists(ctx, similar, includeNotPresent)
	if err != nil {
		return
	}
	artist.SimilarArtists = sa
}

func (e *externalMetadata) mapSimilarArtists(ctx context.Context, similar []agents.Artist, includeNotPresent bool) (model.Artists, error) {
	var result model.Artists
	var notPresent []string

	// First select artists that are present.
	for _, s := range similar {
		sa, err := e.findArtistByName(ctx, s.Name)
		if err != nil {
			notPresent = append(notPresent, s.Name)
			continue
		}
		result = append(result, sa.Artist)
	}

	// Then fill up with non-present artists
	if includeNotPresent {
		for _, s := range notPresent {
			sa := model.Artist{ID: unavailableArtistID, Name: s}
			result = append(result, sa)
		}
	}

	return result, nil
}

func (e *externalMetadata) findArtistByName(ctx context.Context, artistName string) (*auxArtist, error) {
	artists, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Like{"name": artistName},
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
		Name:   clearName(artists[0].Name),
	}
	return artist, nil
}

func (e *externalMetadata) loadSimilar(ctx context.Context, artist *auxArtist, count int, includeNotPresent bool) error {
	var ids []string
	for _, sa := range artist.SimilarArtists {
		if sa.ID == unavailableArtistID {
			continue
		}
		ids = append(ids, sa.ID)
	}

	similar, err := e.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"id": ids},
	})
	if err != nil {
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
			la.ID = unavailableArtistID
		}
		loaded = append(loaded, la)
	}
	artist.SimilarArtists = loaded
	return nil
}
