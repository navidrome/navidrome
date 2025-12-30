package testpkg

import "context"

//nd:hostservice name=Echo permission=echo
type EchoService interface {
	//nd:hostfunc
	Echo(ctx context.Context, message string) (reply string, err error)
}
