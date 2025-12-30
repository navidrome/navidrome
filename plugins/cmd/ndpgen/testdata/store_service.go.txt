package testpkg

import "context"

type Item struct {
	ID   string
	Name string
}

//nd:hostservice name=Store permission=store
type StoreService interface {
	//nd:hostfunc
	Save(ctx context.Context, item Item) (id string, err error)
}
