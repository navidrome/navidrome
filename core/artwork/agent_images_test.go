package artwork

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeImageAgent is a built-in agent stub implementing both image retrievers; it
// records call counts so per-agent ordering and short-circuiting can be asserted.
type fakeImageAgent struct {
	name          string
	imgs          []agents.ExternalImage
	err           error
	artistCalls   int
	albumCalls    int
	gotArtistName string
	gotAlbumName  string
	// block, when set, holds every lookup until closed, standing in for a slow/rate-limited agent.
	block chan struct{}
	// mu guards the call counters: the worker resolves several items concurrently.
	mu sync.Mutex
}

func (f *fakeImageAgent) AgentName() string { return f.name }

func (f *fakeImageAgent) GetArtistImages(_ context.Context, _, name, _ string) ([]agents.ExternalImage, error) {
	if f.block != nil {
		<-f.block
	}
	f.mu.Lock()
	f.artistCalls++
	f.gotArtistName = name
	f.mu.Unlock()
	return f.imgs, f.err
}

func (f *fakeImageAgent) GetAlbumImages(_ context.Context, name, _, _ string) ([]agents.ExternalImage, error) {
	f.albumCalls++
	f.gotAlbumName = name
	return f.imgs, f.err
}

// imageAgents registers the fakes as built-in agents and enables them in order. The fakes
// ignore the DataStore, so reusing the process-wide GetAgents singleton across tests is safe.
func imageAgents(fakes ...*fakeImageAgent) *agents.Agents {
	names := make([]string, 0, len(fakes))
	for _, f := range fakes {
		fake := f
		agents.Register(fake.name, func(model.DataStore) agents.Interface { return fake })
		names = append(names, fake.name)
	}
	conf.Server.Agents = strings.Join(names, ",")
	return agents.GetAgents(&tests.MockDataStore{}, nil)
}

var _ = Describe("agent images", func() {
	var (
		ctx context.Context
		srv *httptest.Server
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("image-bytes"))
		}))
		DeferCleanup(srv.Close)
	})

	img := func(path string, size int) agents.ExternalImage {
		return agents.ExternalImage{URL: srv.URL + path, Size: size}
	}

	Describe("bestImageURL", func() {
		It("picks the largest-Size URL and skips empty ones", func() {
			u := bestImageURL([]agents.ExternalImage{
				{URL: "http://x/small", Size: 10},
				{URL: "", Size: 9999},
				{URL: "http://x/big", Size: 100},
			})
			Expect(u).ToNot(BeNil())
			Expect(u.String()).To(Equal("http://x/big"))
		})

		It("skips a malformed largest URL and falls back to a valid smaller one", func() {
			u := bestImageURL([]agents.ExternalImage{
				{URL: "http://x/valid", Size: 10},
				{URL: "http://x/%zz", Size: 100}, // invalid percent-escape, largest
			})
			Expect(u).ToNot(BeNil())
			Expect(u.String()).To(Equal("http://x/valid"))
		})

		It("returns nil when there is no non-empty URL", func() {
			Expect(bestImageURL(nil)).To(BeNil())
			Expect(bestImageURL([]agents.ExternalImage{{URL: "", Size: 5}})).To(BeNil())
		})

		// Plugins hand these over as free-form strings, and url.Parse accepts them all. An
		// unfetchable candidate that wins here ends the agent's turn before its valid images run.
		DescribeTable("skips a candidate that cannot be fetched",
			func(badURL string) {
				u := bestImageURL([]agents.ExternalImage{
					{URL: badURL, Size: 100}, // largest, and first
					{URL: "https://cdn.example.com/ok.jpg", Size: 10},
				})
				Expect(u).ToNot(BeNil())
				Expect(u.String()).To(Equal("https://cdn.example.com/ok.jpg"))
			},
			Entry("a relative path", "images/big.jpg"),
			Entry("a root-relative path", "/images/big.jpg"),
			Entry("a scheme we cannot fetch", "ftp://host/big.jpg"),
			Entry("a scheme-relative URL", "//host/big.jpg"),
			Entry("a URL with no host", "http:///big.jpg"),
		)

		// Size is often 0 for every candidate, and only a strictly larger one replaces the first,
		// so an unfetchable entry in first position would otherwise stick.
		It("skips an unfetchable first candidate when every Size is zero", func() {
			u := bestImageURL([]agents.ExternalImage{
				{URL: "images/rel.jpg"},
				{URL: "https://cdn.example.com/ok.jpg"},
			})
			Expect(u).ToNot(BeNil())
			Expect(u.String()).To(Equal("https://cdn.example.com/ok.jpg"))
		})

		It("returns nil when no candidate is fetchable", func() {
			Expect(bestImageURL([]agents.ExternalImage{
				{URL: "images/a.jpg", Size: 10},
				{URL: "ftp://host/b.jpg", Size: 20},
			})).To(BeNil())
		})
	})

	Describe("fetchArtistImage", func() {
		It("returns the first agent's image and its name", func() {
			a := &fakeImageAgent{name: "agentA", imgs: []agents.ExternalImage{img("/a", 100)}}
			ag := imageAgents(a)

			r, name, extErr := fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: "ar1", Name: "Artist"})
			Expect(r).ToNot(BeNil())
			defer r.Close()
			Expect(name).To(Equal("agentA"))
			Expect(extErr).To(BeFalse())
		})

		It("skips the external lookup for synthetic artists", func() {
			a := &fakeImageAgent{name: "agentA", imgs: []agents.ExternalImage{img("/a", 100)}}
			ag := imageAgents(a)

			for _, id := range []string{consts.UnknownArtistID, consts.VariousArtistsID} {
				r, name, extErr := fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: id, Name: "Various Artists"})
				Expect(r).To(BeNil())
				Expect(name).To(BeEmpty())
				Expect(extErr).To(BeFalse())
			}
			Expect(a.artistCalls).To(Equal(0), "synthetic artists never reach the agents")
		})

		It("records a skipped external candidate when no agent provides artist images", func() {
			ag := imageAgents()
			t := &chainTrace{}

			r, _, extErr := fetchArtistImage(withTrace(ctx, t), ag, passthroughGate, model.Artist{ID: "ar1"})
			Expect(r).To(BeNil())
			Expect(extErr).To(BeFalse())
			Expect(t.Steps()).To(Equal([]traceStep{{Candidate: "external", Outcome: OutcomeSkipped,
				Detail: "no enabled agent provides artist images"}}),
				"a configured external token must never be silently absent from the chain")
		})

		It("records a skipped external candidate for synthetic artists", func() {
			a := &fakeImageAgent{name: "agentA", imgs: []agents.ExternalImage{img("/a", 100)}}
			ag := imageAgents(a)
			t := &chainTrace{}

			_, _, _ = fetchArtistImage(withTrace(ctx, t), ag, passthroughGate,
				model.Artist{ID: consts.VariousArtistsID, Name: "Various Artists"})
			Expect(t.Steps()).To(HaveLen(1))
			Expect(t.Steps()[0].Outcome).To(Equal(OutcomeSkipped))
			Expect(t.Steps()[0].Detail).To(ContainSubstring("synthetic"))
		})

		It("clears typographic characters from the query name unless preserving unicode", func() {
			conf.Server.DevPreserveUnicodeInExternalCalls = false
			a := &fakeImageAgent{name: "agentA"}
			ag := imageAgents(a)

			_, _, _ = fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: "ar1", Name: "AC’DC"})
			Expect(a.gotArtistName).To(Equal(str.Clear("AC’DC")))
		})

		It("falls through to a later agent, and its success beats the earlier error", func() {
			a := &fakeImageAgent{name: "agentA", err: errBreakerOpen}
			b := &fakeImageAgent{name: "agentB", imgs: []agents.ExternalImage{img("/b", 50)}}
			ag := imageAgents(a, b)

			r, name, extErr := fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: "ar1"})
			Expect(r).ToNot(BeNil())
			defer r.Close()
			Expect(name).To(Equal("agentB"))
			Expect(extErr).To(BeFalse(), "a later hit clears an earlier agent's error")
			Expect(a.artistCalls).To(Equal(1))
			Expect(b.artistCalls).To(Equal(1))
		})

		It("reports a clean miss when every agent finds nothing", func() {
			a := &fakeImageAgent{name: "agentA"} // no images, no error -> not found
			b := &fakeImageAgent{name: "agentB", err: agents.ErrNotFound}
			ag := imageAgents(a, b)

			r, name, extErr := fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: "ar1"})
			Expect(r).To(BeNil())
			Expect(name).To(BeEmpty())
			Expect(extErr).To(BeFalse(), "not-found is definitive, never a transient failure")
		})

		It("reports extErr when one agent fails transiently and the rest find nothing", func() {
			a := &fakeImageAgent{name: "agentA", err: agents.ErrNotFound}
			b := &fakeImageAgent{name: "agentB", err: context.DeadlineExceeded}
			ag := imageAgents(a, b)

			r, _, extErr := fetchArtistImage(ctx, ag, passthroughGate, model.Artist{ID: "ar1"})
			Expect(r).To(BeNil())
			Expect(extErr).To(BeTrue())
		})
	})

	Describe("fetchAlbumImage", func() {
		It("returns the winning agent's image and name", func() {
			a := &fakeImageAgent{name: "agentA", imgs: []agents.ExternalImage{img("/a", 100)}}
			ag := imageAgents(a)

			r, name, extErr := fetchAlbumImage(ctx, ag, passthroughGate, model.Album{Name: "Album", AlbumArtist: "Artist"})
			Expect(r).ToNot(BeNil())
			defer r.Close()
			Expect(name).To(Equal("agentA"))
			Expect(extErr).To(BeFalse())
			Expect(a.albumCalls).To(Equal(1))
		})

		It("records a skipped external candidate when no agent provides album images", func() {
			ag := imageAgents()
			t := &chainTrace{}

			r, _, extErr := fetchAlbumImage(withTrace(ctx, t), ag, passthroughGate, model.Album{Name: "Album"})
			Expect(r).To(BeNil())
			Expect(extErr).To(BeFalse())
			Expect(t.Steps()).To(Equal([]traceStep{{Candidate: "external", Outcome: OutcomeSkipped,
				Detail: "no enabled agent provides album images"}}),
				"a configured external token must never be silently absent from the chain")
		})

		It("reports extErr when the only agent fails transiently", func() {
			a := &fakeImageAgent{name: "agentA", err: context.DeadlineExceeded}
			ag := imageAgents(a)

			r, _, extErr := fetchAlbumImage(ctx, ag, passthroughGate, model.Album{Name: "Album"})
			Expect(r).To(BeNil())
			Expect(extErr).To(BeTrue())
		})
	})

	Describe("gate naming", func() {
		It("invokes the gate once per agent, keyed by agent name", func() {
			a := &fakeImageAgent{name: "agentA", err: context.DeadlineExceeded}
			b := &fakeImageAgent{name: "agentB", imgs: []agents.ExternalImage{img("/b", 1)}}
			ag := imageAgents(a, b)

			var gatedNames []string
			gate := func(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
				gatedNames = append(gatedNames, name)
				return f()
			}

			r, _, _ := fetchArtistImage(ctx, ag, gate, model.Artist{ID: "ar1"})
			Expect(r).ToNot(BeNil())
			defer r.Close()
			Expect(gatedNames).To(Equal([]string{"agentA", "agentB"}))
		})
	})
})
