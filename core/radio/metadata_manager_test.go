package radio

import (
	"context"
	"errors"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetadataManager", func() {
	var (
		reader    *fakeStreamReader
		publisher *fakePublisher
		manager   *MetadataManager
	)

	BeforeEach(func() {
		reader = newFakeStreamReader()
		publisher = newFakePublisher()
		manager = NewMetadataManager(
			reader.Read,
			publisher.Publish,
			WithRetryBackoff(func(int) time.Duration { return 0 }),
			WithNow(func() time.Time { return time.Unix(123, 0).UTC() }),
		)
	})

	It("starts a reader for an active radio session", func() {
		err := manager.Start(context.Background(), "session-1", Station{
			ID:        "rd-1",
			StreamURL: "https://stream.example.test/radio",
		})

		Expect(err).ToNot(HaveOccurred())
		Eventually(reader.StartCount).Should(Equal(1))
		Expect(reader.URLs()).To(Equal([]string{"https://stream.example.test/radio"}))
	})

	It("shares one reader for sessions using the same stream URL", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Expect(manager.Start(context.Background(), "session-2", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())

		Eventually(reader.StartCount).Should(Equal(1))
		Consistently(reader.StartCount).Should(Equal(1))
	})

	It("cancels the reader when the last session stops", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Eventually(reader.StartCount).Should(Equal(1))

		manager.Stop("session-1")

		Eventually(reader.CancelCount).Should(Equal(1))
	})

	It("keeps the reader alive while another session still references the stream URL", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Expect(manager.Start(context.Background(), "session-2", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Eventually(reader.StartCount).Should(Equal(1))

		manager.Stop("session-1")

		Consistently(reader.CancelCount).Should(Equal(0))
		manager.Stop("session-2")
		Eventually(reader.CancelCount).Should(Equal(1))
	})

	It("stops an existing session before starting that session on another station", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/one"})).To(Succeed())
		Eventually(reader.StartCount).Should(Equal(1))

		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-2", StreamURL: "https://stream.example.test/two"})).To(Succeed())

		Eventually(reader.CancelCount).Should(Equal(1))
		Eventually(reader.StartCount).Should(Equal(2))
		Expect(reader.URLs()).To(Equal([]string{
			"https://stream.example.test/one",
			"https://stream.example.test/two",
		}))
	})

	It("publishes title updates for active radio IDs", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Eventually(reader.StartCount).Should(Equal(1))

		reader.Emit("Live Artist - Live Track")

		Eventually(publisher.Updates).Should(Equal([]TitleUpdate{{
			RadioID:   "rd-1",
			Title:     "Live Artist - Live Track",
			UpdatedAt: time.Unix(123, 0).UTC(),
		}}))
	})

	It("does not publish duplicate consecutive titles from the same reader", func() {
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())
		Eventually(reader.StartCount).Should(Equal(1))

		reader.Emit("Same Title")
		reader.Emit("Same Title")

		Consistently(publisher.Updates).Should(Equal([]TitleUpdate{{
			RadioID:   "rd-1",
			Title:     "Same Title",
			UpdatedAt: time.Unix(123, 0).UTC(),
		}}))
	})

	It("retries reader failures until stopped", func() {
		reader.err = errors.New("stream failed")

		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(Succeed())

		Eventually(reader.StartCount).Should(BeNumerically(">", 1))
		manager.Stop("session-1")
		cancelCount := reader.CancelCount()
		Consistently(reader.StartCount).Should(BeNumerically(">=", 2))
		Expect(reader.CancelCount()).To(BeNumerically(">=", cancelCount))
	})

	It("ignores unknown stops", func() {
		manager.Stop("missing-session")
		Expect(reader.StartCount()).To(Equal(0))
	})

	It("rejects invalid start requests", func() {
		Expect(manager.Start(context.Background(), "", Station{ID: "rd-1", StreamURL: "https://stream.example.test/radio"})).To(MatchError(ErrInvalidSession))
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "", StreamURL: "https://stream.example.test/radio"})).To(MatchError(ErrInvalidStation))
		Expect(manager.Start(context.Background(), "session-1", Station{ID: "rd-1", StreamURL: ""})).To(MatchError(ErrInvalidStation))
	})
})

type fakeStreamReader struct {
	mu        sync.Mutex
	started   chan struct{}
	cancelled chan struct{}
	handler   func(string)
	urls      []string
	starts    int
	cancels   int
	err       error
}

func newFakeStreamReader() *fakeStreamReader {
	return &fakeStreamReader{
		started:   make(chan struct{}),
		cancelled: make(chan struct{}),
	}
}

func (r *fakeStreamReader) Read(ctx context.Context, streamURL string, handleTitle func(string)) error {
	r.mu.Lock()
	r.starts++
	r.urls = append(r.urls, streamURL)
	r.handler = handleTitle
	if r.starts == 1 {
		close(r.started)
	}
	err := r.err
	r.mu.Unlock()

	if err != nil {
		return err
	}

	<-ctx.Done()

	r.mu.Lock()
	r.cancels++
	if r.cancels == 1 {
		close(r.cancelled)
	}
	r.mu.Unlock()

	return ctx.Err()
}

func (r *fakeStreamReader) Emit(title string) {
	Eventually(func() func(string) {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.handler
	}).ShouldNot(BeNil())

	r.mu.Lock()
	handler := r.handler
	r.mu.Unlock()
	handler(title)
}

func (r *fakeStreamReader) StartCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.starts
}

func (r *fakeStreamReader) CancelCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancels
}

func (r *fakeStreamReader) URLs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.urls...)
}

type fakePublisher struct {
	mu      sync.Mutex
	updates []TitleUpdate
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{}
}

func (p *fakePublisher) Publish(_ context.Context, update TitleUpdate) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updates = append(p.updates, update)
}

func (p *fakePublisher) Updates() []TitleUpdate {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]TitleUpdate(nil), p.updates...)
}
