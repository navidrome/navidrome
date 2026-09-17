package shuffle

import (
	"strings"

	"github.com/navidrome/navidrome/utils/random"
)

// Keys are the attributes used to keep same-artist (and same-album) tracks
// from sitting next to each other after a shuffle.
type Keys struct {
	Artist string
	Album  string
}

// TrackKeys maps media-file artist/album fields onto shuffle Keys.
// Artist falls back to albumArtist when the track artist is empty.
func TrackKeys(artist, albumArtist, album string) Keys {
	if artist == "" {
		artist = albumArtist
	}
	return Keys{Artist: artist, Album: album}
}

// IsRandomSort reports whether a query sort is a random ordering
// (Native API `_sort=random`, SQL `random()`, or the SEEDEDRAND rewrite).
func IsRandomSort(sort string) bool {
	s := strings.ToLower(strings.TrimSpace(sort))
	return s == "random" || s == "random()" || strings.HasPrefix(s, "seededrand(")
}

// Slice is a full permutation of items: Fisher-Yates, then a light
// adjacent-repair pass. It never drops or duplicates items.
//
// Important: do NOT rebuild the list by always draining the largest artist
// pile — that made libraries dominated by a few artists (e.g. Unknown /
// Arctic Monkeys / Muse) play those artists almost exclusively.
func Slice[T any](items []T, keys func(T) Keys) {
	n := len(items)
	if n < 2 {
		return
	}

	for i := n - 1; i > 0; i-- {
		j := int(random.Int64N(i + 1))
		items[i], items[j] = items[j], items[i]
	}

	repairAdjacent(items, keys)
}

// repairAdjacent keeps the Fisher-Yates order and only swaps forward when
// two neighbors share an artist (then album) and a later break is available.
func repairAdjacent[T any](items []T, keys func(T) Keys) {
	n := len(items)
	for i := 1; i < n; i++ {
		prev := keys(items[i-1])
		cur := keys(items[i])

		if prev.Artist != "" && cur.Artist == prev.Artist {
			for j := i + 1; j < n; j++ {
				if keys(items[j]).Artist != prev.Artist {
					items[i], items[j] = items[j], items[i]
					cur = keys(items[i])
					break
				}
			}
		}

		if prev.Artist != "" && cur.Artist == prev.Artist &&
			prev.Album != "" && cur.Album == prev.Album {
			for j := i + 1; j < n; j++ {
				k := keys(items[j])
				if k.Artist != prev.Artist || k.Album != prev.Album {
					items[i], items[j] = items[j], items[i]
					break
				}
			}
		}
	}
}
