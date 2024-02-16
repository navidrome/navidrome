package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func initialSetup(ds model.DataStore) {
	_ = ds.WithTx(func(tx model.DataStore) error {
		properties := ds.Property(context.TODO())
		_, err := properties.Get(consts.InitialSetupFlagKey)
		if err == nil {
			return nil
		}
		log.Info("Running initial setup")
		if err = createJWTSecret(ds); err != nil {
			return err
		}

		if conf.Server.DevAutoCreateAdminPassword != "" {
			if err = createInitialAdminUser(ds, conf.Server.DevAutoCreateAdminPassword); err != nil {
				return err
			}
		}

		err = properties.Put(consts.InitialSetupFlagKey, time.Now().String())
		return err
	})
}

// If the Dev Admin user is not present, create it
func createInitialAdminUser(ds model.DataStore, initialPassword string) error {
	users := ds.User(context.TODO())
	c, err := users.CountAll(model.QueryOptions{Filters: squirrel.Eq{"user_name": consts.DevInitialUserName}})
	if err != nil {
		panic(fmt.Sprintf("Could not access User table: %s", err))
	}
	if c == 0 {
		id := uuid.NewString()
		log.Warn("Creating initial admin user. This should only be used for development purposes!!",
			"user", consts.DevInitialUserName, "password", initialPassword, "id", id)
		initialUser := model.User{
			ID:          id,
			UserName:    consts.DevInitialUserName,
			Name:        consts.DevInitialName,
			Email:       "",
			NewPassword: initialPassword,
			IsAdmin:     true,
		}
		err := users.Put(&initialUser)
		if err != nil {
			log.Error("Could not create initial admin user", "user", initialUser, err)
		}
	}
	return err
}

func createJWTSecret(ds model.DataStore) error {
	properties := ds.Property(context.TODO())
	_, err := properties.Get(consts.JWTSecretKey)
	if err == nil {
		return nil
	}
	log.Info("Creating new JWT secret, used for encrypting UI sessions")
	err = properties.Put(consts.JWTSecretKey, uuid.NewString())
	if err != nil {
		log.Error("Could not save JWT secret in DB", err)
	}
	return err
}

func checkFfmpegInstallation() {
	f := ffmpeg.New()
	_, err := f.CmdPath()
	if err == nil {
		return
	}
	log.Warn("Unable to find ffmpeg. Transcoding will fail if used", err)
	if conf.Server.Scanner.Extractor == "ffmpeg" {
		log.Warn("ffmpeg cannot be used for metadata extraction. Falling back to taglib")
		conf.Server.Scanner.Extractor = "taglib"
	}
}

func checkExternalCredentials() {
	if conf.Server.EnableExternalServices {
		if !conf.Server.LastFM.Enabled {
			log.Info("Last.fm integration is DISABLED")
		} else {
			log.Debug("Last.fm integration is ENABLED")
		}

		if !conf.Server.ListenBrainz.Enabled {
			log.Info("ListenBrainz integration is DISABLED")
		} else {
			log.Debug("ListenBrainz integration is ENABLED", "ListenBrainz.BaseURL", conf.Server.ListenBrainz.BaseURL)
		}

		if conf.Server.Spotify.ID == "" || conf.Server.Spotify.Secret == "" {
			log.Info("Spotify integration is not enabled: missing ID/Secret")
		} else {
			log.Debug("Spotify integration is ENABLED")
		}
	}
}
