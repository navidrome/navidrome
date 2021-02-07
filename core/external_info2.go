package core

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/lastfm"
	"github.com/navidrome/navidrome/core/spotify"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type externalInfo2 struct {
	ds model.DataStore
}

func NewExternalInfo2(ds model.DataStore, lfm *lastfm.Client, spf *spotify.Client) ExternalInfo {
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

func (e *externalInfo2) getArtist(ctx context.Context, id string) (*model.Artist, error) {
	var entity interface{}
	entity, err := GetEntityByID(ctx, e.ds, id)
	if err != nil {
		return nil, err
	}

	switch v := entity.(type) {
	case *model.Artist:
		return v, nil
	case *model.MediaFile:
		return e.ds.Artist(ctx).Get(v.ArtistID)
	case *model.Album:
		return e.ds.Artist(ctx).Get(v.AlbumArtistID)
	}
	return nil, model.ErrNotFound
}

func (e *externalInfo2) UpdateArtistInfo(ctx context.Context, id string, similarCount int, includeNotPresent bool) (*model.Artist, error) {
	allAgents := e.initAgents(ctx)
	artist, err := e.getArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	// TODO Uncomment
	// If we have updated info, just return it
	//if time.Since(artist.ExternalInfoUpdatedAt) < consts.ArtistInfoTimeToLive {
	//	log.Debug("Found cached ArtistInfo", "updatedAt", artist.ExternalInfoUpdatedAt, "name", artist.Name)
	//	err := e.loadSimilar(ctx, artist, includeNotPresent)
	//	return artist, err
	//}
	log.Debug("ArtistInfo not cached", "updatedAt", artist.ExternalInfoUpdatedAt, "id", id)

	wg := sync.WaitGroup{}
	e.callGetMBID(ctx, allAgents, artist, &wg)
	e.callGetBiography(ctx, allAgents, artist, &wg)
	e.callGetURL(ctx, allAgents, artist, &wg)
	e.callGetSimilar(ctx, allAgents, artist, similarCount, &wg)
	// TODO Images
	wg.Wait()

	artist.ExternalInfoUpdatedAt = time.Now()
	err = e.ds.Artist(ctx).Put(artist)
	if err != nil {
		log.Error(ctx, "Error trying to update artistImageUrl", "id", id, err)
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

	return artist, nil
}

func (e *externalInfo2) SimilarSongs(ctx context.Context, id string, count int) (model.MediaFiles, error) {
	return nil, nil
}

func (e *externalInfo2) TopSongs(ctx context.Context, artistName string, count int) (model.MediaFiles, error) {
	return nil, nil
}

func (e *externalInfo2) callGetMBID(ctx context.Context, allAgents []agents.Interface, artist *model.Artist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, a := range allAgents {
			agent, ok := a.(agents.ArtistMBIDRetriever)
			if !ok {
				continue
			}
			mbid, err := agent.GetMBID(artist.Name)
			if mbid != "" && err == nil {
				artist.MbzArtistID = mbid
			}
		}
	}()
}

func (e *externalInfo2) callGetURL(ctx context.Context, allAgents []agents.Interface, artist *model.Artist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, a := range allAgents {
			agent, ok := a.(agents.ArtistURLRetriever)
			if !ok {
				continue
			}
			url, err := agent.GetURL(artist.Name, artist.MbzArtistID)
			if url != "" && err == nil {
				artist.ExternalUrl = url
			}
		}
	}()
}

func (e *externalInfo2) callGetBiography(ctx context.Context, allAgents []agents.Interface, artist *model.Artist, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, a := range allAgents {
			agent, ok := a.(agents.ArtistBiographyRetriever)
			if !ok {
				continue
			}
			bio, err := agent.GetBiography(artist.Name, artist.MbzArtistID)
			if bio != "" && err == nil {
				artist.Biography = bio
			}
		}
	}()
}

func (e *externalInfo2) callGetSimilar(ctx context.Context, allAgents []agents.Interface, artist *model.Artist, limit int, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, a := range allAgents {
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
			artist.SimilarArtists = sa
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

func (e *externalInfo2) loadSimilar(ctx context.Context, artist *model.Artist, includeNotPresent bool) error {
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
