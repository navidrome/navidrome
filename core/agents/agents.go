package agents

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
)

type Agents struct {
	ctx    context.Context
	agents []Interface
}

func NewAgents(ctx context.Context) *Agents {
	order := strings.Split(conf.Server.Agents, ",")
	order = append(order, PlaceholderAgentName)
	var res []Interface
	for _, name := range order {
		init, ok := Map[name]
		if !ok {
			log.Error(ctx, "Agent not available. Check configuration", "name", name)
			continue
		}

		res = append(res, init(ctx))
	}

	return &Agents{ctx: ctx, agents: res}
}

func (a *Agents) AgentName() string {
	return "agents"
}

func (a *Agents) GetMBID(id string, name string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistMBIDRetriever)
		if !ok {
			continue
		}
		mbid, err := agent.GetMBID(id, name)
		if mbid != "" && err == nil {
			log.Debug(a.ctx, "Got MBID", "agent", ag.AgentName(), "artist", name, "mbid", mbid, "elapsed", time.Since(start))
			return mbid, err
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetURL(id, name, mbid string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistURLRetriever)
		if !ok {
			continue
		}
		url, err := agent.GetURL(id, name, mbid)
		if url != "" && err == nil {
			log.Debug(a.ctx, "Got External Url", "agent", ag.AgentName(), "artist", name, "url", url, "elapsed", time.Since(start))
			return url, err
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetBiography(id, name, mbid string) (string, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistBiographyRetriever)
		if !ok {
			continue
		}
		bio, err := agent.GetBiography(id, name, mbid)
		if bio != "" && err == nil {
			log.Debug(a.ctx, "Got Biography", "agent", ag.AgentName(), "artist", name, "len", len(bio), "elapsed", time.Since(start))
			return bio, err
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetSimilar(id, name, mbid string, limit int) ([]Artist, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistSimilarRetriever)
		if !ok {
			continue
		}
		similar, err := agent.GetSimilar(id, name, mbid, limit)
		if len(similar) >= 0 && err == nil {
			log.Debug(a.ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similar", similar, "elapsed", time.Since(start))
			return similar, err
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetImages(id, name, mbid string) ([]ArtistImage, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistImageRetriever)
		if !ok {
			continue
		}
		images, err := agent.GetImages(id, name, mbid)
		if len(images) > 0 && err == nil {
			log.Debug(a.ctx, "Got Images", "agent", ag.AgentName(), "artist", name, "images", images, "elapsed", time.Since(start))
			return images, err
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetTopSongs(id, artistName, mbid string, count int) ([]Song, error) {
	start := time.Now()
	for _, ag := range a.agents {
		if utils.IsCtxDone(a.ctx) {
			break
		}
		agent, ok := ag.(ArtistTopSongsRetriever)
		if !ok {
			continue
		}
		songs, err := agent.GetTopSongs(id, artistName, mbid, count)
		if len(songs) > 0 && err == nil {
			log.Debug(a.ctx, "Got Top Songs", "agent", ag.AgentName(), "artist", artistName, "songs", songs, "elapsed", time.Since(start))
			return songs, err
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
