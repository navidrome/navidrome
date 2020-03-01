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
	Register(ctx context.Context, id, client, typ, ip string) (*model.Player, error)
}

func NewPlayers(ds model.DataStore) Players {
	return &players{ds}
}

type players struct {
	ds model.DataStore
}

func (p *players) Register(ctx context.Context, id, client, typ, ip string) (*model.Player, error) {
	var plr *model.Player
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
	return plr, err
}

func (p *players) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return p.ds.Player(ctx).Get(playerId)
}
