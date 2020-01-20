package persistence

import (
	"time"

	"github.com/cloudsonic/sonic-server/model"
)

type user struct {
	ID           string    `json:"id"             orm:"pk;column(id)"`
	Name         string    `json:"name"           orm:"index"`
	Password     string    `json:"-"`
	IsAdmin      bool      `json:"isAdmin"`
	LastLoginAt  time.Time `json:"lastLoginAt"`
	LastAccessAt time.Time `json:"lastAccessAt"`
	CreatedAt    time.Time `json:"createdAt"      orm:"auto_now_add;type(datetime)"`
	UpdatedAt    time.Time `json:"updatedAt"      orm:"auto_now;type(datetime)"`
}

var _ = model.User(user{})
