//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	ledis2 "github.com/cloudsonic/sonic-server/persistence/ledis"
	"github.com/cloudsonic/sonic-server/persistence/storm"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	ledis2.Set,
	storm.Set,
	engine.Set,
	scanner.Set,
	newDB,
)

func initImporter(musicFolder string) *scanner.Importer {
	panic(wire.Build(allProviders))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(ledis2.Db())
}
