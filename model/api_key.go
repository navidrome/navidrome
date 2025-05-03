package model

import (
	"github.com/deluan/rest"
	"time"
)

type APIKey struct {
	ID        string    `structs:"id" json:"id"`
	UserID    string    `structs:"user_id" json:"userId"`
	Name      string    `structs:"name" json:"name"`
	Key       string    `structs:"key" json:"key"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type APIKeys []APIKey

type APIKeyRepository interface {
	ResourceRepository
	rest.Persistable
	CountAll(...QueryOptions) (int64, error)
	Get(id string) (*APIKey, error)
	GetAll(options ...QueryOptions) (APIKeys, error)
	Put(*APIKey) error
	FindByKey(key string) (*APIKey, error)
}
