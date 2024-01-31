package agents

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type Agents struct {
	ds     model.DataStore
	agents []Interface
}

func New(ds model.DataStore) *Agents {
	var order []string
	if conf.Server.Agents != "" {
		order = strings.Split(conf.Server.Agents, ",")
	}
	order = append(order, LocalAgentName)
	var res []Interface
	for _, name := range order {
		init, ok := Map[name]
		if !ok {
			log.Error("Agent not available. Check configuration", "name", name)
			continue
		}

		res = append(res, init(ds))
	}

	return &Agents{ds: ds, agents: res}
}

func (a *Agents) AgentName() string {
	return "agents"
}

func (a *Agents) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistMBIDRetriever)
		if !ok {
			continue
		}
		mbid, err := agent.GetArtistMBID(ctx, id, name)
		if mbid != "" && err == nil {
			log.Debug(ctx, "Got MBID", "agent", ag.AgentName(), "artist", name, "mbid", mbid, "elapsed", time.Since(start))
			return mbid, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistURLRetriever)
		if !ok {
			continue
		}
		url, err := agent.GetArtistURL(ctx, id, name, mbid)
		if url != "" && err == nil {
			log.Debug(ctx, "Got External Url", "agent", ag.AgentName(), "artist", name, "url", url, "elapsed", time.Since(start))
			return url, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistBiographyRetriever)
		if !ok {
			continue
		}
		bio, err := agent.GetArtistBiography(ctx, id, name, mbid)
		if err == nil {
			log.Debug(ctx, "Got Biography", "agent", ag.AgentName(), "artist", name, "len", len(bio), "elapsed", time.Since(start))
			return bio, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistSimilarRetriever)
		if !ok {
			continue
		}
		similar, err := agent.GetSimilarArtists(ctx, id, name, mbid, limit)
		if len(similar) > 0 && err == nil {
			if log.IsGreaterOrEqualTo(log.LevelTrace) {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similar", similar, "elapsed", time.Since(start))
			} else {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similarReceived", len(similar), "elapsed", time.Since(start))
			}
			return similar, err
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetArtistImages(ctx context.Context, id, name, sortName, mbid string) ([]ExternalImage, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistImageRetriever)
		if !ok {
			continue
		}
		images, err := agent.GetArtistImages(ctx, id, name, sortName, mbid)
		if len(images) > 0 && err == nil {
			log.Debug(ctx, "Got Images", "agent", ag.AgentName(), "artist", name, "images", images, "elapsed", time.Since(start))
			return images, nil
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistTopSongsRetriever)
		if !ok {
			continue
		}
		songs, err := agent.GetArtistTopSongs(ctx, id, artistName, mbid, count)
		if len(songs) > 0 && err == nil {
			log.Debug(ctx, "Got Top Songs", "agent", ag.AgentName(), "artist", artistName, "songs", songs, "elapsed", time.Since(start))
			return songs, nil
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error) {
	if name == consts.UnknownAlbum {
		return nil, ErrNotFound
	}
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(AlbumInfoRetriever)
		if !ok {
			continue
		}
		album, err := agent.GetAlbumInfo(ctx, name, artist, mbid)
		if err == nil {
			log.Debug(ctx, "Got Album Info", "agent", ag.AgentName(), "album", name, "artist", artist,
				"mbid", mbid, "elapsed", time.Since(start))
			return album, nil
		}
	}
	return nil, ErrNotFound
}

var _ Interface = (*Agents)(nil)
var _ ArtistMBIDRetriever = (*Agents)(nil)
var _ ArtistURLRetriever = (*Agents)(nil)
var _ ArtistBiographyRetriever = (*Agents)(nil)
var _ ArtistSimilarRetriever = (*Agents)(nil)
var _ ArtistImageRetriever = (*Agents)(nil)
var _ ArtistTopSongsRetriever = (*Agents)(nil)
var _ AlbumInfoRetriever = (*Agents)(nil)
