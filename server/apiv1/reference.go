package apiv1

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
)

type referenceIDKey struct{}

func withReferenceID(ctx context.Context, ref string) context.Context {
	return context.WithValue(ctx, referenceIDKey{}, ref)
}

func referenceIDFrom(ctx context.Context) string {
	ref, _ := ctx.Value(referenceIDKey{}).(string)
	return ref
}

// referenceIDMiddleware tags every log line of the request with an id that 500 problems also carry.
func referenceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref := id.NewRandom()
		ctx := log.NewContext(withReferenceID(r.Context(), ref), "referenceId", ref)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
