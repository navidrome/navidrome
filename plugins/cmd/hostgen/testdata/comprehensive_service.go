package testpkg

import "context"

type User2 struct {
	ID   string
	Name string
}

type Filter2 struct {
	Active bool
}

//nd:hostservice name=Comprehensive permission=comprehensive
type ComprehensiveService interface {
	//nd:hostfunc
	SimpleParams(ctx context.Context, name string, count int32) (string, error)
	//nd:hostfunc
	StructParam(ctx context.Context, user User2) error
	//nd:hostfunc
	MixedParams(ctx context.Context, id string, filter Filter2) (int32, error)
	//nd:hostfunc
	NoError(ctx context.Context, name string) string
	//nd:hostfunc
	NoParams(ctx context.Context) error
	//nd:hostfunc
	NoParamsNoReturns(ctx context.Context)
	//nd:hostfunc
	PointerParams(ctx context.Context, id *string, user *User2) (*User2, error)
	//nd:hostfunc
	MapParams(ctx context.Context, data map[string]any) (interface{}, error)
	//nd:hostfunc
	MultipleReturns(ctx context.Context, query string) (results []User2, total int32, err error)
	//nd:hostfunc
	ByteSlice(ctx context.Context, data []byte) ([]byte, error)
}
