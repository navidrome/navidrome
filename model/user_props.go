package model

import "context"

type UserPropsRepository interface {
	Put(ctx context.Context, userId, key string, value string) error
	Get(ctx context.Context, userId, key string) (string, error)
	Delete(ctx context.Context, userId, key string) error
	DefaultGet(ctx context.Context, userId, key string, defaultValue string) (string, error)
}
