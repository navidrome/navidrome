//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	persistence.Set,
	engine.Set,
	scanner.Set,
	newDB,
)

func initImporter(musicFolder string) *scanner.Importer {
	panic(wire.Build(allProviders))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(persistence.Db())
}
