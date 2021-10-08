package model

type UserPropsRepository interface {
	Put(userId, key string, value string) error
	Get(userId, key string) (string, error)
	Delete(userId, key string) error
	DefaultGet(userId, key string, defaultValue string) (string, error)
}
