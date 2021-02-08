package core

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type externalInfo2 struct {
	ds model.DataStore
}

type auxArtist struct {
	model.Artist
	Name string
}

func NewExternalInfo2(ds model.DataStore) ExternalInfo {
	return &externalInfo2{ds: ds}
}

func (e *externalInfo2) initAgents(ctx context.Context) []agents.Interface {
	order := strings.Split(conf.Server.Agents, ",")
	var res []agents.Interface
	for _, name := range order {
		init, ok := agents.Map[name]
		if !ok {
			log.Error(ctx, "Agent not available. Check configuration", "name", name)
			continue
		}

		res = append(res, init(ctx))
	}

	return res
}

func (e *externalInfo2) getArtist(ctx context.Context, id string) (*auxArtist, error) {
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

func (e *externalInfo2) UpdateArtistInfo(ctx context.Context, id string, similarCount int, includeNotPresent bool) (*model.Artist, error) {
	allAgents := e.initAgents(ctx)
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	// If we have fresh info, just return it
	if time.Since(artist.ExternalInfoUpdatedAt) < time.Second { // TODO: consts.ArtistInfoTimeToLive {
		log.Debug("Found cached ArtistInfo", "updatedAt", artist.ExternalInfoUpdatedAt, "name", artist.Name)
		err := e.loadSimilar(ctx, artist, includeNotPresent)
		return &artist.Artist, err
	}
	log.Debug(ctx, "ArtistInfo not cached or expired", "updatedAt", artist.ExternalInfoUpdatedAt, "id", id, "name", artist.Name)

	// Get MBID first, if it is not yet available
	if artist.MbzArtistID == "" {
		e.callGetMBID(ctx, allAgents, artist)
	}

	// Call all registered agents and collect information
	wg := &sync.WaitGroup{}
	e.callGetBiography(ctx, allAgents, artist, wg)
	e.callGetURL(ctx, allAgents, artist, wg)
	e.callGetImage(ctx, allAgents, artist, wg)
	e.callGetSimilar(ctx, allAgents, artist, similarCount, wg)
	wg.Wait()

	if isDone(ctx) {
		log.Warn(ctx, "ArtistInfo update canceled", ctx.Err())
		return nil, ctx.Err()
	}

	artist.ExternalInfoUpdatedAt = time.Now()
	err = e.ds.Artist(ctx).Put(&artist.Artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artist external information", "id", id, "name", artist.Name, err)
	}

	if !includeNotPresent {
		similar := artist.SimilarArtists
		artist.SimilarArtists = nil
		for _, s := range similar {
			if s.ID == unavailableArtistID {
				continue
			}
			artist.SimilarArtists = append(artist.SimilarArtists, s)
		}
	}

	log.Trace(ctx, "ArtistInfo collected", "artist", artist)
	return &artist.Artist, nil
}

func (e *externalInfo2) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	// TODO
	return nil, nil
}

func (e *externalInfo2) TopSongs(ctx context.Context, artistName string, count int) (model.MediaFiles, error) {
	// TODO
	return nil, nil
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (e *externalInfo2) callGetMBID(ctx context.Context, allAgents []agents.Interface, artist *auxArtist) {
	start := time.Now()
	for _, a := range allAgents {
		if isDone(ctx) {
			break
		}
		agent, ok := a.(agents.ArtistMBIDRetriever)
		if !ok {
			continue
		}
		mbid, err := agent.GetMBID(artist.Name)
		if mbid != "" && err == nil {
			artist.MbzArtistID = mbid
			log.Debug(ctx, "Got MBID", "agent", a.AgentName(), "artist", artist.Name, "mbid", mbid, "elapsed", time.Since(start))
			break
		}
	}
}

func (e *externalInfo2) callGetURL(ctx context.Context, allAgents []agents.Interface, artist *auxArtist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for _, a := range allAgents {
			if isDone(ctx) {
				break
			}
			agent, ok := a.(agents.ArtistURLRetriever)
			if !ok {
				continue
			}
			url, err := agent.GetURL(artist.Name, artist.MbzArtistID)
			if url != "" && err == nil {
				artist.ExternalUrl = url
				log.Debug(ctx, "Got External Url", "agent", a.AgentName(), "artist", artist.Name, "url", url, "elapsed", time.Since(start))
				break
			}
		}
	}()
}

func (e *externalInfo2) callGetBiography(ctx context.Context, allAgents []agents.Interface, artist *auxArtist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for _, a := range allAgents {
			if isDone(ctx) {
				break
			}
			agent, ok := a.(agents.ArtistBiographyRetriever)
			if !ok {
				continue
			}
			bio, err := agent.GetBiography(clearName(artist.Name), artist.MbzArtistID)
			if bio != "" && err == nil {
				artist.Biography = bio
				log.Debug(ctx, "Got Biography", "agent", a.AgentName(), "artist", artist.Name, "len", len(bio), "elapsed", time.Since(start))
				break
			}
		}
	}()
}

func (e *externalInfo2) callGetImage(ctx context.Context, allAgents []agents.Interface, artist *auxArtist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for _, a := range allAgents {
			if isDone(ctx) {
				break
			}
			agent, ok := a.(agents.ArtistImageRetriever)
			if !ok {
				continue
			}
			images, err := agent.GetImages(artist.Name, artist.MbzArtistID)
			if len(images) == 0 || err != nil {
				continue
			}
			log.Debug(ctx, "Got Images", "agent", a.AgentName(), "artist", artist.Name, "images", images, "elapsed", time.Since(start))
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
			break
		}
	}()
}

func (e *externalInfo2) callGetSimilar(ctx context.Context, allAgents []agents.Interface, artist *auxArtist, limit int, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for _, a := range allAgents {
			if isDone(ctx) {
				break
			}
			agent, ok := a.(agents.ArtistSimilarRetriever)
			if !ok {
				continue
			}
			similar, err := agent.GetSimilar(artist.Name, artist.MbzArtistID, limit)
			if len(similar) == 0 || err != nil {
				continue
			}
			sa, err := e.mapSimilarArtists(ctx, similar, true)
			if err != nil {
				continue
			}
			log.Debug(ctx, "Got Similar Artists", "agent", a.AgentName(), "artist", artist.Name, "similar", similar, "elapsed", time.Since(start))
			artist.SimilarArtists = sa
			break
		}
	}()
}

func (e *externalInfo2) mapSimilarArtists(ctx context.Context, similar []agents.Artist, includeNotPresent bool) (model.Artists, error) {
	var result model.Artists
	var notPresent []string

	// First select artists that are present.
	for _, s := range similar {
		sa, err := e.findArtistByName(ctx, s.Name)
		if err != nil {
			notPresent = append(notPresent, s.Name)
			continue
		}
		result = append(result, *sa)
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

func (e *externalInfo2) findArtistByName(ctx context.Context, artistName string) (*model.Artist, error) {
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
	return &artists[0], nil
}

func (e *externalInfo2) loadSimilar(ctx context.Context, artist *auxArtist, includeNotPresent bool) error {
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
