package model

type UserProp struct {
	UserID string `structs:"user_id" orm:"column(user_id)"`
	Key    string `structs:"key"`
	Value  string `structs:"value"`
}

type UserProps []UserProp

type UserPropsRepository interface {
	Put(userId, key string, value string) error
	Get(userId, key string) (string, error)
	GetAllWithPrefix(key string) (UserProps, error)
	Delete(userId, key string) error
	DefaultGet(userId, key string, defaultValue string) (string, error)
}
