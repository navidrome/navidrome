package radio

import (
	"fmt"

	"github.com/navidrome/navidrome/core/radio/icy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata provider", func() {
	Describe("classifyICYError", func() {
		It("marks invalid metadata intervals as permanent", func() {
			err := classifyICYError(fmt.Errorf("%w: %q", icy.ErrInvalidMetaInt, "999999999"))

			Expect(isPermanent(err)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring(icy.ErrInvalidMetaInt.Error())))
		})
	})
})
