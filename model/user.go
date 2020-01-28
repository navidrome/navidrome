package model

import "time"

type User struct {
	ID           string     `json:"id" orm:"column(id)"`
	UserName     string     `json:"userName"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Password     string     `json:"password"`
	IsAdmin      bool       `json:"isAdmin"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	LastAccessAt *time.Time `json:"lastAccessAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	// TODO ChangePassword string     `json:"password"`
}

type Users []User

type UserRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Get(id string) (*User, error)
	Put(*User) error
	FindByUsername(username string) (*User, error)
	UpdateLastLoginAt(id string) error
	UpdateLastAccessAt(id string) error
}
