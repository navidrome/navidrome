package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type Players interface {
	Get(ctx context.Context, playerId string) (*model.Player, error)
	Register(ctx context.Context, id, client, typ, ip string) (*model.Player, *model.Transcoding, error)
}

func NewPlayers(ds model.DataStore) Players {
	return &players{ds}
}

type players struct {
	ds model.DataStore
}

func (p *players) Register(ctx context.Context, id, client, userAgent, ip string) (*model.Player, *model.Transcoding, error) {
	plr := p.GetPlayerFromIdOrNewPlayerForUser(ctx, id, client, userAgent)

	plr.UserAgent = userAgent
	plr.IPAddress = ip
	plr.LastSeen = time.Now()

	if err := p.ds.Player(ctx).Put(plr); err != nil {
		return nil, nil, err
	}

	trc, err := p.GetTranscodingOptionsForPlayer(ctx, plr.TranscodingId)

	return plr, trc, err
}

func (p *players) GetPlayerFromIdOrNewPlayerForUser(ctx context.Context, id string, client string, userAgent string) *model.Player {
	if id != "" {
		plr, err := p.ds.Player(ctx).Get(id)
		if err == nil && plr.Client == client {
			return plr
		}
	}

	userId, _ := request.UserIdFrom(ctx)

	if plr, err := p.ds.Player(ctx).FindMatch(userId, client, userAgent); err == nil {
		log.Debug("Found matching player", "id", plr.ID, "client", client, "userId", userId, "type", userAgent)
		return plr
	}

	plr := &model.Player{
		ID:              uuid.NewString(),
		Name:            fmt.Sprintf("%s [%s]", client, userAgent),
		UserId:          userId,
		Client:          client,
		ScrobbleEnabled: true,
	}
	log.Info("Registering new player", "id", plr.ID, "client", client, "userId", userId, "type", userAgent)
	return plr
}

func (p *players) GetTranscodingOptionsForPlayer(ctx context.Context, transcodingId string) (*model.Transcoding, error) {
	if transcodingId == "" {
		return nil, nil
	}

	return p.ds.Transcoding(ctx).Get(transcodingId)
}

func (p *players) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return p.ds.Player(ctx).Get(playerId)
}
