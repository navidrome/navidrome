package repositories

type BaseRepository struct {
	key string
}


func (r *BaseRepository) saveOrUpdate(id string, rec interface{}) error {
	return saveStruct(r.key, id, rec)
}

func (r *BaseRepository) CountAll() (int, error) {
	return count(r.key)
}

func (r *BaseRepository) Dump() {
}


