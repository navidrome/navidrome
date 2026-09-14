package shuffle_test

import (
	"strconv"
	"testing"

	"github.com/navidrome/navidrome/utils/shuffle"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestShuffle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shuffle Suite")
}

type track struct {
	id, artist, album string
}

func keys(t track) shuffle.Keys {
	return shuffle.Keys{Artist: t.artist, Album: t.album}
}

func ids(tracks []track) []string {
	out := make([]string, len(tracks))
	for i, t := range tracks {
		out[i] = t.id
	}
	return out
}

func clone(tracks []track) []track {
	out := make([]track, len(tracks))
	copy(out, tracks)
	return out
}

func adjacentSame(tracks []track, field func(track) string) int {
	n := 0
	for i := 1; i < len(tracks); i++ {
		if field(tracks[i]) != "" && field(tracks[i]) == field(tracks[i-1]) {
			n++
		}
	}
	return n
}

func minAdjacent(n, maxFreq int) int {
	if n == 0 {
		return 0
	}
	return max(0, 2*maxFreq-n-1)
}

var _ = Describe("IsRandomSort", func() {
	It("matches Native API, SQL random(), and SEEDEDRAND rewrite", func() {
		Expect(shuffle.IsRandomSort("random")).To(BeTrue())
		Expect(shuffle.IsRandomSort("RANDOM")).To(BeTrue())
		Expect(shuffle.IsRandomSort("random()")).To(BeTrue())
		Expect(shuffle.IsRandomSort("SEEDEDRAND('media_file|abc', media_file.id)")).To(BeTrue())
		Expect(shuffle.IsRandomSort("title")).To(BeFalse())
		Expect(shuffle.IsRandomSort("")).To(BeFalse())
		Expect(shuffle.IsRandomSort("order_title")).To(BeFalse())
	})
})

var _ = Describe("TrackKeys", func() {
	It("falls back to album artist when track artist is empty", func() {
		Expect(shuffle.TrackKeys("", "AA", "X")).To(Equal(shuffle.Keys{Artist: "AA", Album: "X"}))
		Expect(shuffle.TrackKeys("A", "AA", "X")).To(Equal(shuffle.Keys{Artist: "A", Album: "X"}))
	})
})

var _ = Describe("Slice", func() {
	It("leaves empty and single-item slices unchanged", func() {
		empty := []track{}
		shuffle.Slice(empty, keys)
		Expect(empty).To(BeEmpty())

		one := []track{{id: "1", artist: "A", album: "X"}}
		shuffle.Slice(one, keys)
		Expect(ids(one)).To(Equal([]string{"1"}))
	})

	It("permutes two items without dropping or duplicating", func() {
		tracks := []track{
			{id: "1", artist: "A", album: "X"},
			{id: "2", artist: "B", album: "Y"},
		}
		original := ids(tracks)
		shuffle.Slice(tracks, keys)
		Expect(ids(tracks)).To(ConsistOf(original))
	})

	It("is a full permutation of a larger list", func() {
		tracks := make([]track, 20)
		for i := range tracks {
			tracks[i] = track{
				id:     strconv.Itoa(i),
				artist: "Artist" + strconv.Itoa(i%5),
				album:  "Album" + strconv.Itoa(i%7),
			}
		}
		original := ids(tracks)
		shuffle.Slice(tracks, keys)
		Expect(ids(tracks)).To(ConsistOf(original))
	})

	It("still shuffles a single-artist playlist", func() {
		tracks := []track{
			{id: "1", artist: "A", album: "X"},
			{id: "2", artist: "A", album: "X"},
			{id: "3", artist: "A", album: "X"},
			{id: "4", artist: "A", album: "X"},
		}
		shuffle.Slice(tracks, keys)
		Expect(ids(tracks)).To(ConsistOf("1", "2", "3", "4"))
		Expect(adjacentSame(tracks, func(t track) string { return t.artist })).To(Equal(3))
	})

	It("produces no adjacent same-artist pairs when artists are balanced", func() {
		original := make([]track, 0, 24)
		for i := range 24 {
			original = append(original, track{
				id:     strconv.Itoa(i),
				artist: "Artist" + strconv.Itoa(i%6),
				album:  "Album" + strconv.Itoa(i),
			})
		}
		for range 25 {
			tracks := clone(original)
			shuffle.Slice(tracks, keys)
			Expect(ids(tracks)).To(ConsistOf(ids(original)))
			Expect(adjacentSame(tracks, func(t track) string { return t.artist })).To(Equal(0))
		}
	})

	It("meets the minimum adjacent-artist count when one artist dominates", func() {
		// 11 of A, 3 of B, 2 of C → n=16, maxFreq=11, min adjacent = 2*11-16-1 = 5
		original := make([]track, 0, 16)
		for i := range 11 {
			original = append(original, track{id: "A" + strconv.Itoa(i), artist: "A", album: "X"})
		}
		for i := range 3 {
			original = append(original, track{id: "B" + strconv.Itoa(i), artist: "B", album: "Y"})
		}
		for i := range 2 {
			original = append(original, track{id: "C" + strconv.Itoa(i), artist: "C", album: "Z"})
		}
		for range 20 {
			tracks := clone(original)
			shuffle.Slice(tracks, keys)
			Expect(ids(tracks)).To(ConsistOf(ids(original)))
			Expect(adjacentSame(tracks, func(t track) string { return t.artist })).To(Equal(minAdjacent(16, 11)))
		}
	})

	It("spaces albums when every track is the same artist", func() {
		original := make([]track, 0, 12)
		for i := range 12 {
			original = append(original, track{
				id:     strconv.Itoa(i),
				artist: "A",
				album:  "Album" + strconv.Itoa(i%3),
			})
		}
		for range 20 {
			tracks := clone(original)
			shuffle.Slice(tracks, keys)
			Expect(ids(tracks)).To(ConsistOf(ids(original)))
			Expect(adjacentSame(tracks, func(t track) string { return t.album })).To(Equal(0))
		}
	})
})
