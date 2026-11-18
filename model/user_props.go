package model

import "context"

type UserPropsRepository interface {
	Put(ctx context.Context, userId, key string, value string) error
	Get(ctx context.Context, userId, key string) (string, error)
	Delete(ctx context.Context, userId, key string) error
	DefaultGet(ctx context.Context, userId, key string, defaultValue string) (string, error)
	// CompareAndSwap stores value only when the current value equals expected.
	// expected == nil requires the row to be absent. It reports whether the
	// swap happened; false means another writer changed the row and the caller
	// must re-read and re-apply its update.
	CompareAndSwap(ctx context.Context, userId, key, value string, expected *string) (bool, error)
}
