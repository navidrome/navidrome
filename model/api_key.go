package model

import (
	"github.com/deluan/rest"
	"time"
)

type APIKey struct {
	ID        string    `structs:"id"         json:"id"`
	PlayerID  string    `structs:"player_id"  json:"playerId"`
	Name      string    `structs:"name"       json:"name"`
	Key       string    `structs:"key"        json:"key"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type APIKeys []APIKey

type APIKeyRepository interface {
	ResourceRepository
	rest.Persistable
	CountAll(...QueryOptions) (int64, error)
	Get(id string) (*APIKey, error)
	GetAll(options ...QueryOptions) (APIKeys, error)
	FindByKey(key string) (*APIKey, error)
	RefreshKey(id string) (string, error)
}
