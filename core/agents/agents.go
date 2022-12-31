package agents

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
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

func (a *Agents) GetMBID(ctx context.Context, id string, name string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistMBIDRetriever)
		if !ok {
			continue
		}
		mbid, err := agent.GetMBID(ctx, id, name)
		if mbid != "" && err == nil {
			log.Debug(ctx, "Got MBID", "agent", ag.AgentName(), "artist", name, "mbid", mbid, "elapsed", time.Since(start))
			return mbid, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetURL(ctx context.Context, id, name, mbid string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistURLRetriever)
		if !ok {
			continue
		}
		url, err := agent.GetURL(ctx, id, name, mbid)
		if url != "" && err == nil {
			log.Debug(ctx, "Got External Url", "agent", ag.AgentName(), "artist", name, "url", url, "elapsed", time.Since(start))
			return url, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetBiography(ctx context.Context, id, name, mbid string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistBiographyRetriever)
		if !ok {
			continue
		}
		bio, err := agent.GetBiography(ctx, id, name, mbid)
		if bio != "" && err == nil {
			log.Debug(ctx, "Got Biography", "agent", ag.AgentName(), "artist", name, "len", len(bio), "elapsed", time.Since(start))
			return bio, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetSimilar(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistSimilarRetriever)
		if !ok {
			continue
		}
		similar, err := agent.GetSimilar(ctx, id, name, mbid, limit)
		if len(similar) > 0 && err == nil {
			if log.CurrentLevel() >= log.LevelTrace {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similar", similar, "elapsed", time.Since(start))
			} else {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similarReceived", len(similar), "elapsed", time.Since(start))
			}
			return similar, err
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetImages(ctx context.Context, id, name, mbid string) ([]ArtistImage, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistImageRetriever)
		if !ok {
			continue
		}
		images, err := agent.GetImages(ctx, id, name, mbid)
		if len(images) > 0 && err == nil {
			log.Debug(ctx, "Got Images", "agent", ag.AgentName(), "artist", name, "images", images, "elapsed", time.Since(start))
			return images, nil
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(ctx) {
			break
		}
		agent, ok := ag.(ArtistTopSongsRetriever)
		if !ok {
			continue
		}
		songs, err := agent.GetTopSongs(ctx, id, artistName, mbid, count)
		if len(songs) > 0 && err == nil {
			log.Debug(ctx, "Got Top Songs", "agent", ag.AgentName(), "artist", artistName, "songs", songs, "elapsed", time.Since(start))
			return songs, nil
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
