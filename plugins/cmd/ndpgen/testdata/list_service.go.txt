package testpkg

import "context"

type Filter struct {
	Active bool
}

//nd:hostservice name=List permission=list
type ListService interface {
	//nd:hostfunc
	Items(ctx context.Context, name string, filter Filter) (count int32, err error)
}
