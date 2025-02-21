package core

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
)

type Players interface {
	Get(ctx context.Context, playerId string) (*model.Player, error)
	Register(ctx context.Context, id, client, userAgent, ip string) (*model.Player, *model.Transcoding, error)
}

func NewPlayers(ds model.DataStore) Players {
	return &players{
		ds:      ds,
		limiter: utils.Limiter{Interval: consts.UpdatePlayerFrequency},
	}
}

type players struct {
	ds      model.DataStore
	limiter utils.Limiter
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
	username := userName(ctx)
	if err != nil || playerID == "" {
		plr, err = p.ds.Player(ctx).FindMatch(user.ID, client, userAgent)
		if err == nil {
			log.Debug(ctx, "Found matching player", "id", plr.ID, "client", client, "username", username, "type", userAgent)
		} else {
			plr = &model.Player{
				ID:              id.NewRandom(),
				UserId:          user.ID,
				Client:          client,
				ScrobbleEnabled: true,
				ReportRealPath:  conf.Server.DefaultReportRealPath,
			}
			log.Info(ctx, "Registering new player", "id", plr.ID, "client", client, "username", username, "type", userAgent)
		}
	}
	plr.Name = fmt.Sprintf("%s [%s]", client, userAgent)
	plr.UserAgent = userAgent
	plr.IP = ip
	plr.LastSeen = time.Now()
	p.limiter.Do(plr.ID, func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		err = p.ds.Player(ctx).Put(plr)
		if err != nil {
			log.Warn(ctx, "Could not save player", "id", plr.ID, "client", client, "username", username, "type", userAgent, err)
		}
	})
	if plr.TranscodingId != "" {
		trc, err = p.ds.Transcoding(ctx).Get(plr.TranscodingId)
	}
	return plr, trc, err
}

func (p *players) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return p.ds.Player(ctx).Get(playerId)
}
