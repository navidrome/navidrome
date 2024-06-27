package core

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
)

type Players interface {
	Get(ctx context.Context, playerId string) (*model.Player, error)
	Register(ctx context.Context, id, client, userAgent, ip string) (*model.Player, *model.Transcoding, error)
}

func NewPlayers(ds model.DataStore) Players {
	return &players{ds}
}

type players struct {
	ds model.DataStore
}

func (p *players) Register(ctx context.Context, playerID, client, userAgent, ip string) (*model.Player, *model.Transcoding, error) {
	var plr *model.Player
	var trc *model.Transcoding
	var err error
	user, _ := request.UserFrom(ctx)
	if playerID != "" {
		plr, err = p.ds.Player(ctx).Get(playerID)
		if err == nil && plr.Client != client {
			playerID = ""
		}
	}
	if err != nil || playerID == "" {
		plr, err = p.ds.Player(ctx).FindMatch(user.ID, client, userAgent)
		if err == nil {
			log.Debug(ctx, "Found matching player", "id", plr.ID, "client", client, "username", userName, "type", userAgent)
		} else {
			plr = &model.Player{
				ID:              id.NewRandom(),
				UserId:          user.ID,
				Client:          client,
				ScrobbleEnabled: true,
			}
			log.Info(ctx, "Registering new player", "id", plr.ID, "client", client, "username", userName, "type", userAgent)
		}
	}
	plr.Name = fmt.Sprintf("%s [%s]", client, userAgent)
	plr.UserAgent = userAgent
	plr.IP = ip
	plr.LastSeen = time.Now()
	err = p.ds.Player(ctx).Put(plr)
	if err != nil {
		return nil, nil, err
	}
	if plr.TranscodingId != "" {
		trc, err = p.ds.Transcoding(ctx).Get(plr.TranscodingId)
	}
	return plr, trc, err
}

func (p *players) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return p.ds.Player(ctx).Get(playerId)
}
