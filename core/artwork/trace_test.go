package artwork

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("chainTrace", func() {
	It("returns nil when no trace is attached", func() {
		Expect(traceFrom(context.Background())).To(BeNil())
	})

	It("collects steps in order", func() {
		t := &chainTrace{}
		ctx := withTrace(context.Background(), t)

		traceFrom(ctx).add(traceStep{Candidate: "cover.*", Outcome: outcomeMiss})
		traceFrom(ctx).add(traceStep{Candidate: "embedded", Outcome: outcomeHit, Detail: "/music/a.flac"})

		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "cover.*", Outcome: outcomeMiss},
			{Candidate: "embedded", Outcome: outcomeHit, Detail: "/music/a.flac"},
		}))

		s := t.Steps()
		s[0].Candidate = "mutated"
		Expect(t.Steps()[0].Candidate).To(Equal("cover.*"))
	})

	It("does not panic when the trace is nil", func() {
		var t *chainTrace
		Expect(func() { t.add(traceStep{Candidate: "cover.*", Outcome: outcomeMiss}) }).ToNot(Panic())
	})

	It("is safe to use concurrently", func() {
		t := &chainTrace{}
		done := make(chan struct{})
		for range 10 {
			go func() {
				defer GinkgoRecover()
				t.add(traceStep{Candidate: "x", Outcome: outcomeMiss})
				done <- struct{}{}
			}()
		}
		for range 10 {
			<-done
		}
		Expect(t.Steps()).To(HaveLen(10))
	})
})

var _ = Describe("chainState tracing", func() {
	It("records a miss when the candidate was absent", func() {
		t := &chainTrace{}
		c := chainState{trace: t}

		_, ok := c.try("cover.*", resolution{}, false)

		Expect(ok).To(BeFalse())
		Expect(t.Steps()).To(Equal([]traceStep{{Candidate: "cover.*", Outcome: outcomeMiss}}))
	})

	It("records unreadable when the candidate existed but could not be read", func() {
		t := &chainTrace{}
		c := chainState{trace: t}

		_, ok := c.try("cover.*", resolution{localError: true}, false)

		Expect(ok).To(BeFalse())
		Expect(t.Steps()).To(HaveLen(1))
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeUnreadable),
			"a candidate that existed and failed to decode must be distinguishable from one that was absent")
	})

	It("records a hit with the backing path", func() {
		t := &chainTrace{}
		c := chainState{trace: t}

		res, ok := c.try("embedded", resolution{reader: nil, source: "embedded", sourcePath: "/music/a.flac"}, true)

		Expect(ok).To(BeTrue())
		Expect(res.source).To(Equal("embedded"))
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "embedded", Outcome: outcomeHit, Detail: "/music/a.flac"},
		}))
	})

	It("does not panic when no trace is attached", func() {
		c := chainState{}
		Expect(func() { _, _ = c.try("cover.*", resolution{}, false) }).ToNot(Panic())
	})
})

var _ = Describe("resolveArtist tracing", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		artistRepo *tests.MockArtistRepo
		ffm        *tests.MockFFmpeg
		ag         *agents.Agents
		t          *chainTrace
	)

	uploadPath := func(file string) string {
		path := model.UploadedImagePath(consts.EntityArtist, file)
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, []byte("uploaded artist image"), 0o600)).To(Succeed())
		return path
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.ArtistArtPriority = ""
		artistRepo = tests.CreateMockArtistRepo()
		ds = &tests.MockDataStore{
			MockedArtist:  artistRepo,
			MockedFolder:  &fakeFolderRepo{},
			MockedLibrary: &tests.MockLibraryRepo{},
		}
		ffm = tests.NewMockFFmpeg("")
		ag = agents.GetAgents(&tests.MockDataStore{}, nil)
		t = &chainTrace{}
		ctx = withTrace(context.Background(), t)
	})

	It("records the upload short-circuit as a hit", func() {
		path := uploadPath("ar1_test.jpg")
		artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist", UploadedImage: "ar1_test.jpg"}})

		res, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar1"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(t.Steps()).To(Equal([]traceStep{{Candidate: "upload", Outcome: outcomeHit, Detail: path}}))
	})

	It("records an upload miss before walking the chain", func() {
		artistRepo.SetData(model.Artists{{ID: "ar2", Name: "Artist"}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar2"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]traceStep{{Candidate: "upload", Outcome: outcomeMiss}}))
	})

	It("records an upload that exists but cannot be read as unreadable", func() {
		if runtime.GOOS == "windows" {
			Skip("chmod does not restrict read access on Windows")
		}
		path := uploadPath("ar3_test.jpg")
		Expect(os.Chmod(path, 0o000)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(path, 0o600) })
		artistRepo.SetData(model.Artists{{ID: "ar3", Name: "Artist", UploadedImage: "ar3_test.jpg"}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar3"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(HaveLen(1))
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeUnreadable),
			"an upload that exists and will not open must not look like an absent upload")
	})
})
