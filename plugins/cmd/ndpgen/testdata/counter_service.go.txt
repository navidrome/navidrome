package testpkg

import "context"

//nd:hostservice name=Counter permission=counter
type CounterService interface {
	//nd:hostfunc
	Count(ctx context.Context, name string) (value int32)
}
