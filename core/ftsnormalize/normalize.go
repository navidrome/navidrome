package ftsnormalize

import (
	"context"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/core/metadataworker"
	"github.com/navidrome/navidrome/core/rustworker"
	"github.com/navidrome/navidrome/log"
	"golang.org/x/sync/singleflight"
)

const normalizeCacheSep = "\x00"

// NormalizeForFTS returns normalized FTS secondary tokens via the Rust
// navidrome-metadata worker. The normalization rules live in rust/fts-normalize.
// Results are cached and singleflighted so scanner Put storms reuse one RPC.
func NormalizeForFTS(ctx context.Context, values ...string) string {
	if len(values) == 0 {
		return ""
	}
	key := cacheKey(values)
	if cached, ok := normalizeCache.Load(key); ok {
		return cached.(string)
	}
	v, err, _ := normalizeSF.Do(key, func() (any, error) {
		if cached, ok := normalizeCache.Load(key); ok {
			return cached.(string), nil
		}
		normalized, err := metadataworker.PersistentNormalizeWorkers().Normalize(ctx, values...)
		if err != nil {
			return "", err
		}
		if normalized != "" {
			normalizeCache.Store(key, normalized)
		}
		return normalized, nil
	})
	if err != nil {
		log.Warn(ctx, "Rust FTS normalize worker failed", err)
		return ""
	}
	return v.(string)
}

// NormalizeMany normalizes many value-groups in one metadata gRPC round-trip
// when possible, filling gaps via NormalizeForFTS (cached) on failure.
func NormalizeMany(ctx context.Context, groups [][]string) []string {
	out := make([]string, len(groups))
	if len(groups) == 0 {
		return out
	}

	missingIdx := make([]int, 0, len(groups))
	missing := make([][]string, 0, len(groups))
	for i, values := range groups {
		if len(values) == 0 {
			continue
		}
		key := cacheKey(values)
		if cached, ok := normalizeCache.Load(key); ok {
			out[i] = cached.(string)
			continue
		}
		missingIdx = append(missingIdx, i)
		missing = append(missing, values)
	}
	if len(missing) == 0 {
		return out
	}

	batched, err := metadataworker.NormalizeFtsBatch(ctx, missing)
	if err == nil && len(batched) == len(missing) {
		for j, idx := range missingIdx {
			out[idx] = batched[j]
			if batched[j] != "" {
				normalizeCache.Store(cacheKey(missing[j]), batched[j])
			}
		}
		return out
	}
	if err != nil {
		if rustworker.PreferGRPC(err, metadataworker.ErrNoGRPC) {
			log.Warn(ctx, "Rust FTS normalize batch failed; falling back per item", err)
		} else {
			log.Debug(ctx, "Rust FTS normalize batch unavailable; falling back per item", err)
		}
	}

	for j, idx := range missingIdx {
		out[idx] = NormalizeForFTS(ctx, missing[j]...)
	}
	return out
}

func cacheKey(values []string) string {
	return strings.Join(values, normalizeCacheSep)
}

var (
	normalizeCache sync.Map
	normalizeSF    singleflight.Group
)
