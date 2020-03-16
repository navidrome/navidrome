package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
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
	userName := ctx.Value("username").(string)
	if id != "" {
		plr, err = p.ds.Player(ctx).Get(id)
	}
	if err != nil || id == "" {
		plr, err = p.ds.Player(ctx).FindByName(client, userName)
		if err == nil {
			log.Trace("Found player by name", "id", plr.ID, "client", client, "userName", userName)
		} else {
			r, _ := uuid.NewRandom()
			plr = &model.Player{
				ID:       r.String(),
				Name:     fmt.Sprintf("%s (%s)", client, userName),
				UserName: userName,
				Client:   client,
			}
			log.Trace("Create new player", "id", plr.ID, "client", client, "userName", userName)
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
