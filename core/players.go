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

func (p *players) Register(ctx context.Context, id, client, typ, ip string) (*model.Player, *model.Transcoding, error) {
	var plr *model.Player
	var trc *model.Transcoding
	var err error
	userName, _ := request.UsernameFrom(ctx)
	if id != "" {
		plr, err = p.ds.Player(ctx).Get(id)
		if err == nil && plr.Client != client {
			id = ""
		}
	}
	if err != nil || id == "" {
		plr, err = p.ds.Player(ctx).FindByName(client, userName)
		if err == nil {
			log.Debug("Found player by name", "id", plr.ID, "client", client, "username", userName)
		} else {
			plr = &model.Player{
				ID:       uuid.NewString(),
				Name:     fmt.Sprintf("%s (%s)", client, userName),
				UserName: userName,
				Client:   client,
			}
			log.Info("Registering new player", "id", plr.ID, "client", client, "username", userName)
		}
	}
	plr.LastSeen = time.Now()
	plr.Type = typ
	plr.IPAddress = ip
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
