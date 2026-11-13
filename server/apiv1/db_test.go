package apiv1

import (
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
)

var realDS model.DataStore

func resetDB() {
	_, _ = db.Db().Exec("delete from api_grant")
	_, _ = db.Db().Exec("delete from user")
}
