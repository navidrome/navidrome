package model

import "time"

type User struct {
	ID           string     `json:"id" orm:"column(id)"`
	UserName     string     `json:"userName"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	IsAdmin      bool       `json:"isAdmin"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	LastAccessAt *time.Time `json:"lastAccessAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`

	// This is only available on the backend, and it is never sent over the wire
	Password string `json:"-"`
	// This is used to set or change a password when calling Put. If it is empty, the password is not changed.
	// It is received from the UI with the name "password"
	NewPassword string `json:"password,omitempty"`
	// If changing the password, this is also required
	CurrentPassword string `json:"currentPassword,omitempty"`
}

type Users []User

type UserRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Get(id string) (*User, error)
	Put(*User) error
	UpdateLastLoginAt(id string) error
	UpdateLastAccessAt(id string) error
	FindFirstAdmin() (*User, error)
	// FindByUsername must be case-insensitive
	FindByUsername(username string) (*User, error)
	// FindByUsernameWithPassword is the same as above, but also returns the decrypted password
	FindByUsernameWithPassword(username string) (*User, error)
}
