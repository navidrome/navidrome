package persistence

import (
	"github.com/navidrome/navidrome/db"
	"github.com/pocketbase/dbx"
)

type dbxBuilder struct {
	dbx.Builder
	wdb dbx.Builder
}

func NewDBXBuilder(d db.DB) *dbxBuilder {
	b := &dbxBuilder{}
	b.Builder = dbx.NewFromDB(d.ReadDB(), db.Driver)
	b.wdb = dbx.NewFromDB(d.WriteDB(), db.Driver)
	return b
}

func (d *dbxBuilder) Transactional(f func(*dbx.Tx) error) (err error) {
	return d.wdb.(*dbx.DB).Transactional(f)
}
