package db

import (
	"database/sql"
	"os"

	"github.com/deluan/navidrome/conf"
	_ "github.com/deluan/navidrome/db/migrations"
	"github.com/deluan/navidrome/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose"
)

const driver = "sqlite3"

func EnsureDB() {
	db, err := sql.Open(driver, conf.Server.DbPath)
	defer db.Close()
	if err != nil {
		log.Error("Failed to open DB", err)
		os.Exit(1)
	}

	err = goose.SetDialect(driver)
	if err != nil {
		log.Error("Invalid DB driver", "driver", driver, err)
		os.Exit(1)
	}
	err = goose.Run("up", db, "./")
	if err != nil {
		log.Error("Failed to apply new migrations", err)
		os.Exit(1)
	}
}
