package server

import (
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
		_, err := ds.Property().Get(consts.InitialSetupFlagKey)
		if err == nil {
			return nil
		}
		log.Warn("Running initial setup")
		if err = createDefaultUser(ds); err != nil {
			return err
		}
		if err = createJWTSecret(ds); err != nil {
			return err
		}

		err = ds.Property().Put(consts.InitialSetupFlagKey, time.Now().String())
		return err
	})
}

func createJWTSecret(ds model.DataStore) error {
	_, err := ds.Property().Get(consts.JWTSecretKey)
	if err == nil {
		return nil
	}
	jwtSecret, _ := uuid.NewRandom()
	log.Warn("Creating JWT secret, used for encrypting UI sessions")
	err = ds.Property().Put(consts.JWTSecretKey, jwtSecret.String())
	if err != nil {
		log.Error("Could not save JWT secret in DB", err)
	}
	return err
}

func createDefaultUser(ds model.DataStore) error {
	c, err := ds.User().CountAll()
	if err != nil {
		panic(fmt.Sprintf("Could not access User table: %s", err))
	}
	if c == 0 {
		id, _ := uuid.NewRandom()
		random, _ := uuid.NewRandom()
		initialPassword := random.String()
		if conf.Sonic.DevInitialPassword != "" {
			initialPassword = conf.Sonic.DevInitialPassword
		}
		log.Warn("Creating initial user. Please change the password!", "user", consts.InitialUserName, "password", initialPassword)
		initialUser := model.User{
			ID:       id.String(),
			UserName: consts.InitialUserName,
			Name:     consts.InitialName,
			Email:    "",
			Password: initialPassword,
			IsAdmin:  true,
		}
		err := ds.User().Put(&initialUser)
		if err != nil {
			log.Error("Could not create initial user", "user", initialUser, err)
		}
	}
	return err
}
