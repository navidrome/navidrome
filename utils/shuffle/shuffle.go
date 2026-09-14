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

// Slice is a full permutation of items: Fisher-Yates, then a greedy spacing
// pass that avoids adjacent same-artist tracks when possible, and same-album
// tracks when the artist cannot change (single-artist lists, or one artist
// owning more than half the queue).
//
// It never drops or duplicates items. If a spacing constraint is impossible
// (e.g. every track is the same artist and album), the Fisher-Yates order is
// left as-is for that constraint.
func Slice[T any](items []T, keys func(T) Keys) {
	n := len(items)
	if n < 2 {
		return
	}

	for i := n - 1; i > 0; i-- {
		j := int(random.Int64N(i + 1))
		items[i], items[j] = items[j], items[i]
	}

	space(items, keys)
}

func space[T any](items []T, keys func(T) Keys) {
	remaining := append([]T(nil), items...)
	result := make([]T, 0, len(items))
	hasLast := false
	var lastArtist, lastAlbum string

	for len(remaining) > 0 {
		artist := pickKey(remaining, lastArtist, hasLast, func(item T) string { return keys(item).Artist })
		var sameArtist []T
		for _, item := range remaining {
			if keys(item).Artist == artist {
				sameArtist = append(sameArtist, item)
			}
		}
		album := pickKey(sameArtist, lastAlbum, hasLast, func(item T) string { return keys(item).Album })

		pick := 0
		for i, item := range remaining {
			k := keys(item)
			if k.Artist == artist && k.Album == album {
				pick = i
				break
			}
		}
		chosen := remaining[pick]
		remaining = append(remaining[:pick], remaining[pick+1:]...)
		result = append(result, chosen)
		k := keys(chosen)
		lastArtist, lastAlbum = k.Artist, k.Album
		hasLast = true
	}

	copy(items, result)
}

// pickKey chooses the value with the most remaining items, skipping `last`
// when any other value is still available. Ties keep Fisher-Yates order.
func pickKey[T any](items []T, last string, hasLast bool, key func(T) string) string {
	counts := make(map[string]int, len(items))
	order := make([]string, 0)
	for _, item := range items {
		k := key(item)
		if _, ok := counts[k]; !ok {
			order = append(order, k)
		}
		counts[k]++
	}

	others := 0
	if hasLast {
		for k := range counts {
			if k != last {
				others++
			}
		}
	}

	best := order[0]
	bestN := -1
	for _, k := range order {
		if hasLast && others > 0 && k == last {
			continue
		}
		if counts[k] > bestN {
			bestN = counts[k]
			best = k
		}
	}
	return best
}
