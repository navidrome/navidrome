package model

import "context"

type PropertyRepository interface {
	Put(ctx context.Context, id string, value string) error
	Get(ctx context.Context, id string) (string, error)
	Delete(ctx context.Context, id string) error
	DefaultGet(ctx context.Context, id string, defaultValue string) (string, error)
}
