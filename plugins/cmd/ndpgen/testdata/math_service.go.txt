package testpkg

import "context"

//nd:hostservice name=Math permission=math
type MathService interface {
	//nd:hostfunc
	Add(ctx context.Context, a int32, b int32) (result int32, err error)
}
