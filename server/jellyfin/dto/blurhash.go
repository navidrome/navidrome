package dto

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
)

// blurHash returns a valid 6-char blurhash for a solid color derived from seed. Finamp only needs a
// well-formed, per-tag-stable value (it uses this as a download de-dup key and blur placeholder), so
// a solid color unique to the tag satisfies both without decoding cover art.
func blurHash(seed string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	sum := h.Sum(nil)
	r, g, b := int(sum[0]), int(sum[1]), int(sum[2])
	dc := (r << 16) | (g << 8) | b
	return "00" + blurhash.Encode83(dc, 4)
}

// primaryBlurHash returns the stored blurhash when it was computed from the entity's current
// artwork version or later (the snapshot folds in image file mtimes, which can exceed row
// timestamps); otherwise a fake seeded by id+version, so the value still rotates on any artwork
// change (Finamp keys its cover caches by this value; tags never reach its image URLs).
func primaryBlurHash(stored string, storedAt *time.Time, id string, version time.Time) string {
	if stored != "" && storedAt != nil && !storedAt.Before(version) {
		return stored
	}
	return blurHash(fmt.Sprintf("%s-%x", id, version.UnixMilli()))
}
