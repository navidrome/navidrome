package testpkg

import "context"

//nd:hostservice name=Meta permission=meta
type MetaService interface {
	//nd:hostfunc
	Get(ctx context.Context, key string) (value interface{}, err error)
	//nd:hostfunc
	Set(ctx context.Context, data map[string]any) error
}
