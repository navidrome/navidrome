package gravatar_test

import (
	"testing"

	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gravatar"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGravatar(t *testing.T) {
	tests.Init(t, false)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gravatar Test Suite")
}

var _ = Describe("Gravatar", func() {
	It("returns a well formatted gravatar URL", func() {
		Expect(gravatar.Url("my@email.com", 100)).To(Equal("https://www.gravatar.com/avatar/cf3d8259741b19a2b09e17d4fa9a97c63adc44bf2a5fa075cdcb5491f525feaa?s=100"))
	})
	It("sets the default size", func() {
		Expect(gravatar.Url("my@email.com", 0)).To(Equal("https://www.gravatar.com/avatar/cf3d8259741b19a2b09e17d4fa9a97c63adc44bf2a5fa075cdcb5491f525feaa?s=80"))
	})
	It("caps maximum size", func() {
		Expect(gravatar.Url("my@email.com", 3000)).To(Equal("https://www.gravatar.com/avatar/cf3d8259741b19a2b09e17d4fa9a97c63adc44bf2a5fa075cdcb5491f525feaa?s=2048"))
	})
	It("ignores case", func() {
		Expect(gravatar.Url("MY@email.com", 0)).To(Equal(gravatar.Url("my@email.com", 0)))
	})
	It("ignores spaces", func() {
		Expect(gravatar.Url("  my@email.com  ", 0)).To(Equal(gravatar.Url("my@email.com", 0)))
	})
})
