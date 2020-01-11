//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence/db_ledis"
	"github.com/cloudsonic/sonic-server/persistence/db_storm"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	//db_ledis.Set,
	db_storm.Set,
	engine.Set,
	scanner.Set,
	newDB,
	api.NewRouter,
)

func initRouter() *api.Router {
	panic(wire.Build(allProviders))
}

func initImporter(musicFolder string) *scanner.Importer {
	panic(wire.Build(allProviders))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(db_ledis.Db())
}
