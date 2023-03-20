package external_playlists

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaylistRetriever interface {
	GetAvailableAgents(ctx context.Context, userId string) []AgentType
	GetPlaylists(ctx context.Context, offset, count int, userId, agent, playlistType string) (*ExternalPlaylists, error)
	ImportPlaylists(ctx context.Context, update bool, userId, agent string, mapping map[string]string) error
	SyncPlaylist(ctx context.Context, playlistId string) error
}

type playlistRetriever struct {
	ds             model.DataStore
	retrievers     map[string]PlaylistAgent
	supportedTypes map[string]map[string]bool
}

var (
	ErrorMissingAgent    = errors.New("agent not found")
	ErrorUnsupportedType = errors.New("unsupported playlist type")
)

func GetPlaylistRetriever(ds model.DataStore) PlaylistRetriever {
	return singleton.GetInstance(func() *playlistRetriever {
		return newPlaylistRetriever(ds)
	})
}

func newPlaylistRetriever(ds model.DataStore) *playlistRetriever {
	p := &playlistRetriever{
		ds:             ds,
		retrievers:     make(map[string]PlaylistAgent),
		supportedTypes: make(map[string]map[string]bool),
	}
	for name, constructor := range constructors {
		s := constructor(ds)
		p.retrievers[name] = s

		mapping := map[string]bool{}

		for _, plsType := range s.GetPlaylistTypes() {
			mapping[plsType] = true
		}
		p.supportedTypes[name] = mapping
	}
	return p
}

func (p *playlistRetriever) GetAvailableAgents(ctx context.Context, userId string) []AgentType {
	user, _ := request.UserFrom(ctx)

	agents := []AgentType{}

	for name, agent := range p.retrievers {
		if agent.IsAuthorized(ctx, user.ID) {
			agents = append(agents, AgentType{
				Name:  name,
				Types: agent.GetPlaylistTypes(),
			})
		}
	}

	return agents
}

func (p *playlistRetriever) validateTypeAgent(ctx context.Context, agent, playlistType string) (PlaylistAgent, error) {
	ag, ok := p.retrievers[agent]

	if !ok {
		log.Error(ctx, "Agent not found", "agent", agent)
		return nil, ErrorMissingAgent
	}

	_, ok = p.supportedTypes[agent][playlistType]

	if !ok {
		log.Error(ctx, "Unsupported playlist type", "agent", agent, "playlist type", playlistType)
		return nil, ErrorUnsupportedType
	}

	return ag, nil
}

func (p *playlistRetriever) GetPlaylists(ctx context.Context, offset, count int, userId, agent, playlistType string) (*ExternalPlaylists, error) {
	ag, err := p.validateTypeAgent(ctx, agent, playlistType)

	if err != nil {
		return nil, err
	}

	pls, err := ag.GetPlaylists(ctx, offset, count, userId, playlistType)

	if err != nil {
		log.Error(ctx, "Error retrieving playlist", "agent", agent, "user", userId, err)
		return nil, err
	}

	ids := make([]string, len(pls.Lists))

	for i, list := range pls.Lists {
		ids[i] = list.ID
	}

	existingIDs, err := p.ds.Playlist(ctx).CheckExternalIds(agent, ids)

	if err != nil {
		log.Error(ctx, "Error checking for existing ids", "agent", agent, "user", userId, err)
		return nil, err
	}

	existingMap := map[string]bool{}

	for _, id := range existingIDs {
		existingMap[id] = true
	}

	for i, list := range pls.Lists {
		_, ok := existingMap[list.ID]
		pls.Lists[i].Existing = ok
	}

	return pls, nil
}

func (p *playlistRetriever) ImportPlaylists(ctx context.Context, update bool, userId, agent string, mapping map[string]string) error {
	ag, ok := p.retrievers[agent]

	if !ok {
		return ErrorMissingAgent
	}

	fail := 0
	var err error

	for id, name := range mapping {
		err = ag.ImportPlaylist(ctx, update, userId, id, name)

		if err != nil {
			fail++
			log.Error(ctx, "Could not import playlist", "agent", agent, "id", id, "error", err)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to sync %d playlist(s): %w", fail, err)
	}

	return nil
}

func (p *playlistRetriever) SyncPlaylist(ctx context.Context, playlistId string) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		pls, err := tx.Playlist(ctx).Get(playlistId)

		if err != nil {
			return err
		}

		if pls.ExternalId == "" {
			return model.ErrNotAvailable
		}

		user, _ := request.UserFrom(ctx)

		if user.ID != pls.OwnerID {
			return model.ErrNotAuthorized
		}

		ag, ok := p.retrievers[pls.ExternalAgent]

		if !ok {
			log.Error(ctx, "No retriever for playlist", "type", ag, "id", playlistId)
			return ErrorMissingAgent
		}

		return ag.SyncPlaylist(ctx, tx, pls)
	})
}

var constructors map[string]Constructor

func Register(name string, init Constructor) {
	if constructors == nil {
		constructors = make(map[string]Constructor)
	}
	constructors[name] = init
}
