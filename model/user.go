package model

import (
	"time"
)

type User struct {
	ID           string     `structs:"id" json:"id"`
	UserName     string     `structs:"user_name" json:"userName"`
	Name         string     `structs:"name" json:"name"`
	Email        string     `structs:"email" json:"email"`
	IsAdmin      bool       `structs:"is_admin" json:"isAdmin"`
	LastLoginAt  *time.Time `structs:"last_login_at" json:"lastLoginAt"`
	LastAccessAt *time.Time `structs:"last_access_at" json:"lastAccessAt"`
	CreatedAt    time.Time  `structs:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `structs:"updated_at" json:"updatedAt"`

	// Library associations (many-to-many relationship)
	Libraries Libraries `structs:"-" json:"libraries,omitempty"`

	// This is only available on the backend, and it is never sent over the wire
	Password string `structs:"-" json:"-"`
	// This is used to set or change a password when calling Put. If it is empty, the password is not changed.
	// It is received from the UI with the name "password"
	NewPassword string `structs:"password,omitempty" json:"password,omitempty"`
	// If changing the password, this is also required
	CurrentPassword string `structs:"current_password,omitempty" json:"currentPassword,omitempty"`
}

func (u User) HasLibraryAccess(libraryID int) bool {
	if u.IsAdmin {
		return true // Admin users have access to all libraries
	}
	for _, lib := range u.Libraries {
		if lib.ID == libraryID {
			return true
		}
	}
	return false
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

	// Library association methods
	GetUserLibraries(userID string) (Libraries, error)
	SetUserLibraries(userID string, libraryIDs []int) error
}
