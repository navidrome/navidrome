package testpkg

import "context"

//nd:hostservice name=Codec permission=codec
type CodecService interface {
	//nd:hostfunc
	Encode(ctx context.Context, data []byte) ([]byte, error)
}
