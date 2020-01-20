package model

import "time"

type User struct {
	ID           string
	Name         string
	Password     string
	IsAdmin      bool
	LastLoginAt  *time.Time
	LastAccessAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Get(id string) (*User, error)
	Put(*User) error
}
