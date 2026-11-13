package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

// Keeps reads out of the transaction: SQLite fails a write that follows a read at once, without waiting for the lock
func initialSetup(ctx context.Context, ds model.DataStore) error {
	if err := ds.Library().StoreMusicFolder(ctx); err != nil {
		return err
	}
	if _, err := ds.Property().Get(ctx, consts.InitialSetupFlagKey); err == nil {
		return nil
	}
	return ds.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
		log.Info("Running initial setup")
		if conf.Server().DevAutoCreateAdminPassword != "" {
			if err := createInitialAdminUser(ctx, tx, conf.Server().DevAutoCreateAdminPassword); err != nil {
				return err
			}
		}
		return tx.Property().Put(ctx, consts.InitialSetupFlagKey, time.Now().String())
	}, "initial setup")
}

// If the Dev Admin user is not present, create it
func createInitialAdminUser(ctx context.Context, ds model.DataStore, initialPassword string) error {
	users := ds.User()
	c, err := users.CountAll(ctx, model.QueryOptions{Filters: squirrel.Eq{"user_name": consts.DevInitialUserName}})
	if err != nil {
		return fmt.Errorf("could not access User table: %w", err)
	}
	if c == 0 {
		newID := id.NewRandom()
		log.Warn("Creating initial admin user. This should only be used for development purposes!!",
			"user", consts.DevInitialUserName, "password", initialPassword, "id", newID)
		initialUser := model.User{
			ID:          newID,
			UserName:    consts.DevInitialUserName,
			Name:        consts.DevInitialName,
			Email:       "",
			NewPassword: initialPassword,
			IsAdmin:     true,
		}
		if err := users.Put(ctx, &initialUser); err != nil {
			return fmt.Errorf("could not create initial admin user: %w", err)
		}
	}
	return nil
}

func checkFFmpegInstallation() {
	f := ffmpeg.New()
	_, err := f.CmdPath()
	if err != nil {
		log.Warn("Unable to find ffmpeg. Transcoding will fail if used", err)
		return
	}
	if !f.IsProbeAvailable() {
		log.Warn("Unable to find ffprobe. Transcoding decisions will be limited")
	}
}

func checkExternalCredentials() {
	if conf.Server().EnableExternalServices {
		if !conf.Server().LastFM.Enabled {
			log.Info("Last.fm integration is DISABLED")
		} else {
			log.Debug("Last.fm integration is ENABLED")
		}

		if !conf.Server().ListenBrainz.Enabled {
			log.Info("ListenBrainz integration is DISABLED")
		} else {
			log.Debug("ListenBrainz integration is ENABLED", "ListenBrainz.BaseURL", conf.Server().ListenBrainz.BaseURL)
		}
	}
}
