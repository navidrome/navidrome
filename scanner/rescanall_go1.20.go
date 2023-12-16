//go:build !go1.21

package scanner

import (
	"context"
)

// TODO Remove this file when we drop support for go 1.20
func contextWithoutCancel(ctx context.Context) context.Context {
	return context.TODO()
}
