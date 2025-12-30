package testpkg

import "context"

type User struct {
	ID   string
	Name string
}

//nd:hostservice name=Users permission=users
type UsersService interface {
	//nd:hostfunc
	Get(ctx context.Context, id *string, filter *User) (*User, error)
}
