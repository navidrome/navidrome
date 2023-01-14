package gravatar_test

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gravatar"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGravatar(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gravatar Test Suite")
}

var _ = Describe("Gravatar", func() {
	It("returns a well formatted gravatar URL", func() {
		Expect(gravatar.Url("my@email.com", 100)).To(Equal("https://www.gravatar.com/avatar/4f384e9f3e8e625aae72b52658323d70?s=100"))
	})
	It("sets the default size", func() {
		Expect(gravatar.Url("my@email.com", 0)).To(Equal("https://www.gravatar.com/avatar/4f384e9f3e8e625aae72b52658323d70?s=80"))
	})
	It("caps maximum size", func() {
		Expect(gravatar.Url("my@email.com", 3000)).To(Equal("https://www.gravatar.com/avatar/4f384e9f3e8e625aae72b52658323d70?s=2048"))
	})
	It("ignores case", func() {
		Expect(gravatar.Url("MY@email.com", 0)).To(Equal(gravatar.Url("my@email.com", 0)))
	})
	It("ignores spaces", func() {
		Expect(gravatar.Url("  my@email.com  ", 0)).To(Equal(gravatar.Url("my@email.com", 0)))
	})
})
