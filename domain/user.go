package domain

import "time"

type User struct {
	ID        string
	Name      string
	Password  string
	IsAdmin   bool
	CreatedAt time.Time
}
