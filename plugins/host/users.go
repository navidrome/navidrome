package host

import "context"

// User represents a Navidrome user with minimal information exposed to plugins.
// Sensitive fields like password, email, and internal IDs are intentionally excluded.
type User struct {
	UserName string `json:"userName"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isAdmin"`
}

// UsersService provides access to user information for plugins.
//
// This service allows plugins to query information about users that the plugin
// has been granted access to. Access is controlled by the administrator who
// configures which users each plugin can see.
//
//nd:hostservice name=Users permission=users
type UsersService interface {
	// GetUsers returns all users the plugin has been granted access to.
	// Only minimal user information (userName, name, isAdmin) is returned.
	// Sensitive fields like password and email are never exposed.
	//
	// Returns a slice of users the plugin can access, or an empty slice if none configured.
	//nd:hostfunc
	GetUsers(ctx context.Context) ([]User, error)
}
