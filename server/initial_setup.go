package server

import (
	"context"
	"fmt"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
)

func initialSetup(ds model.DataStore) {
	_ = ds.WithTx(func(tx model.DataStore) error {
		_, err := ds.Property(nil).Get(consts.InitialSetupFlagKey)
		if err == nil {
			return nil
		}
		log.Warn("Running initial setup")
		if err = createJWTSecret(ds); err != nil {
			return err
		}

		if conf.Server.DevAutoCreateAdminPassword != "" {
			if err = createInitialAdminUser(ds); err != nil {
				return err
			}
		}

		err = ds.Property(nil).Put(consts.InitialSetupFlagKey, time.Now().String())
		return err
	})
}

func createInitialAdminUser(ds model.DataStore) error {
	ctx := context.Background()
	c, err := ds.User(ctx).CountAll()
	if err != nil {
		panic(fmt.Sprintf("Could not access User table: %s", err))
	}
	if c == 0 {
		id, _ := uuid.NewRandom()
		random, _ := uuid.NewRandom()
		initialPassword := random.String()
		if conf.Server.DevAutoCreateAdminPassword != "" {
			initialPassword = conf.Server.DevAutoCreateAdminPassword
		}
		log.Warn("Creating initial admin user. This should only be used for development purposes!!", "user", consts.DevInitialUserName, "password", initialPassword)
		initialUser := model.User{
			ID:       id.String(),
			UserName: consts.DevInitialUserName,
			Name:     consts.DevInitialName,
			Email:    "",
			Password: initialPassword,
			IsAdmin:  true,
		}
		err := ds.User(ctx).Put(&initialUser)
		if err != nil {
			log.Error("Could not create initial admin user", "user", initialUser, err)
		}
	}
	return err
}

func createJWTSecret(ds model.DataStore) error {
	_, err := ds.Property(nil).Get(consts.JWTSecretKey)
	if err == nil {
		return nil
	}
	jwtSecret, _ := uuid.NewRandom()
	log.Warn("Creating JWT secret, used for encrypting UI sessions")
	err = ds.Property(nil).Put(consts.JWTSecretKey, jwtSecret.String())
	if err != nil {
		log.Error("Could not save JWT secret in DB", err)
	}
	return err
}
