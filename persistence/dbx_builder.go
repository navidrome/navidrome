package persistence

import (
	"github.com/navidrome/navidrome/db"
	"github.com/pocketbase/dbx"
)

type dbxBuilder struct {
	dbx.Builder
	rdb dbx.Builder
}

func NewDBXBuilder(d db.DB) *dbxBuilder {
	b := &dbxBuilder{}
	b.Builder = dbx.NewFromDB(d.WriteDB(), db.Driver)
	b.rdb = dbx.NewFromDB(d.ReadDB(), db.Driver)
	return b
}

func (d *dbxBuilder) NewQuery(s string) *dbx.Query {
	return d.rdb.NewQuery(s)
}

func (d *dbxBuilder) Select(s ...string) *dbx.SelectQuery {
	return d.rdb.Select(s...)
}

func (d *dbxBuilder) GeneratePlaceholder(i int) string {
	return d.rdb.GeneratePlaceholder(i)
}

func (d *dbxBuilder) Quote(s string) string {
	return d.rdb.Quote(s)
}

func (d *dbxBuilder) QuoteSimpleTableName(s string) string {
	return d.rdb.QuoteSimpleTableName(s)
}

func (d *dbxBuilder) QuoteSimpleColumnName(s string) string {
	return d.rdb.QuoteSimpleColumnName(s)
}

func (d *dbxBuilder) QueryBuilder() dbx.QueryBuilder {
	return d.rdb.QueryBuilder()
}

func (d *dbxBuilder) Transactional(f func(*dbx.Tx) error) (err error) {
	return d.Builder.(*dbx.DB).Transactional(f)
}
