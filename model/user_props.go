package model

// UserPropsRepository is meant to be scoped for the user, that can be obtained from request.UserFrom(r.Context())
type UserPropsRepository interface {
	Put(key string, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	DefaultGet(key string, defaultValue string) (string, error)
}
