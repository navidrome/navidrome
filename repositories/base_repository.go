package repositories

type BaseRepository struct {
	key string
}


func (r *BaseRepository) saveOrUpdate(id string, rec interface{}) error {
	return hmset(r.key + "_id_" + id, rec)
}

func (r *BaseRepository) Dump() {
}


