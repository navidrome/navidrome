package plugins

// UserAccess encapsulates user authorization for a plugin,
// determining which users are allowed to interact with it.
type UserAccess struct {
	allUsers  bool
	userIDMap map[string]struct{}
}

// NewUserAccess creates a UserAccess from the plugin's configuration.
// If allUsers is true, all users are allowed regardless of the list.
func NewUserAccess(allUsers bool, userIDs []string) UserAccess {
	userIDMap := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		userIDMap[id] = struct{}{}
	}
	return UserAccess{
		allUsers:  allUsers,
		userIDMap: userIDMap,
	}
}

// IsAllowed checks if the given user ID is permitted.
func (ua UserAccess) IsAllowed(userID string) bool {
	if ua.allUsers {
		return true
	}
	_, ok := ua.userIDMap[userID]
	return ok
}

// HasConfiguredUsers reports whether any specific user IDs have been configured.
func (ua UserAccess) HasConfiguredUsers() bool {
	return ua.allUsers || len(ua.userIDMap) > 0
}
