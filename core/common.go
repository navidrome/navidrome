package core

import (
	"context"

	"github.com/navidrome/navidrome/model/request"
)

func userName(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); !ok {
		return "UNKNOWN"
	} else {
		return user.UserName
	}
}
