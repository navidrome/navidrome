//go:build go1.21

package scanner

import (
	"context"
)

func contextWithoutCancel(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
