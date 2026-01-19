package testpkg

import "context"

//nd:hostservice name=Ping permission=ping
type PingService interface {
	//nd:hostfunc
	Ping(ctx context.Context) error
}
