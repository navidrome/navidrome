package blurhash_test

import (
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/zeebo/xxh3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Synthetic", func() {
	It("returns a well-formed 3x3 blurhash", func() {
		h := blurhash.Synthetic("alb-1", "")
		// 3x3 encodes as (3-1)+(3-1)*9 = 20, which is base83 'K'; length is 1+1+4+8*2.
		Expect(h).To(HaveLen(22))
		Expect(h).To(HavePrefix("K"))
		for _, c := range h {
			Expect(strings.ContainsRune(alphabet, c)).To(BeTrue(), "unexpected char %q", c)
		}
	})

	It("keeps its prefix exclusive to real encodes, across aspect ratios", func() {
		Expect(blurhash.Synthetic("alb-1", "")).To(HavePrefix("K"))

		ratios := []struct{ w, h int }{
			{10, 200}, {200, 10}, {1, 50}, {50, 1}, {64, 64}, {100, 300}, {300, 100},
		}
		for _, r := range ratios {
			encoded, err := blurhash.Encode(gradientImage(r.w, r.h))
			Expect(err).ToNot(HaveOccurred())
			Expect(encoded).ToNot(HavePrefix("K"), "real encode at %dx%d produced the synthetic prefix", r.w, r.h)
		}
	})

	It("is deterministic for one seed", func() {
		Expect(blurhash.Synthetic("alb-1", "")).To(Equal(blurhash.Synthetic("alb-1", "")))
	})

	It("differs across seeds", func() {
		Expect(blurhash.Synthetic("alb-1", "")).ToNot(Equal(blurhash.Synthetic("alb-2", "")))
	})

	It("decodes to a muted DC colour", func() {
		dc := decode83(blurhash.Synthetic("alb-1", "")[2:6])
		r, g, b := dc>>16&0xFF, dc>>8&0xFF, dc&0xFF
		spread := max(r, g, b) - min(r, g, b)
		Expect(spread).To(BeNumerically("<", 120), "saturation is capped, so no channel should run away")
		Expect(max(r, g, b)).To(BeNumerically("<", 240), "lightness is capped below white")
		Expect(min(r, g, b)).To(BeNumerically(">", 10), "lightness is capped above black")
	})

	// The Finamp maintainers' condition for accepting a fabricated value is that it is
	// unique; this is the assertion that holds us to it.
	It("keeps 1,000,000 seeds effectively distinct", func() {
		const count = 1_000_000
		seen := make(map[uint64]struct{}, count)
		for i := range count {
			seen[xxh3.HashString(blurhash.Synthetic(fmt.Sprintf("item-%d", i), ""))] = struct{}{}
		}
		Expect(len(seen)).To(BeNumerically(">=", 999_900))
	})
})
