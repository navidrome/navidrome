package apikeyworker

import (
	"context"
)

type generateResult struct {
	Token        string
	LookupPrefix string
	Hash         string
}

func Generate(ctx context.Context, pepper string) (generateResult, error) {
	return generateGRPC(ctx, pepper)
}

func Hash(ctx context.Context, token, pepper string) (string, string, error) {
	return hashGRPC(ctx, token, pepper)
}

func Verify(ctx context.Context, token, hash, pepper string) (bool, error) {
	return verifyGRPC(ctx, token, hash, pepper)
}
