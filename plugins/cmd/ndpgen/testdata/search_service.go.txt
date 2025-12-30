package testpkg

import "context"

type Result struct {
	ID string
}

//nd:hostservice name=Search permission=search
type SearchService interface {
	//nd:hostfunc
	Find(ctx context.Context, query string) (results []Result, total int32, err error)
}
