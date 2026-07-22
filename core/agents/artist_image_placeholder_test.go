package agents

import (
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agents.IsArtistImagePlaceholder", func() {
	var ds model.DataStore

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ds = &tests.MockDataStore{}
		Register("phDetector", func(model.DataStore) Interface { return &placeholderDetectorAgent{marker: "PLACEHOLDER"} })
		Register("phPlain", func(model.DataStore) Interface { return &plainAgent{} })
	})

	It("returns true when any enabled agent's detector recognizes the URL", func() {
		conf.Server.Agents = "phDetector"
		ag := createAgents(ds, nil)

		Expect(ag.IsArtistImagePlaceholder("http://cdn/PLACEHOLDER.jpg")).To(BeTrue())
		Expect(ag.IsArtistImagePlaceholder("http://cdn/real.jpg")).To(BeFalse())
	})

	It("returns false when no enabled agent implements the detector", func() {
		conf.Server.Agents = "phPlain"
		ag := createAgents(ds, nil)

		Expect(ag.IsArtistImagePlaceholder("http://cdn/PLACEHOLDER.jpg")).To(BeFalse())
	})

	It("returns false for an empty URL", func() {
		conf.Server.Agents = "phDetector"
		ag := createAgents(ds, nil)

		Expect(ag.IsArtistImagePlaceholder("")).To(BeFalse())
	})
})

type placeholderDetectorAgent struct {
	Interface
	marker string
}

func (a *placeholderDetectorAgent) AgentName() string { return "phDetector" }
func (a *placeholderDetectorAgent) IsArtistImagePlaceholder(url string) bool {
	return strings.Contains(url, a.marker)
}

type plainAgent struct{ Interface }

func (a *plainAgent) AgentName() string { return "phPlain" }
