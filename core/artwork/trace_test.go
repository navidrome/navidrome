package artwork

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

var _ = Describe("external gate tracing", func() {
	hit := func() (io.ReadCloser, string, error) {
		return io.NopCloser(strings.NewReader("x")), "http://img", nil
	}
	miss := func() (io.ReadCloser, string, error) { return nil, "", agents.ErrNotFound }
	boom := func() (io.ReadCloser, string, error) { return nil, "", errors.New("returned status 429") }

	It("records a hit with the image path", func() {
		t := &chainTrace{}
		g := tracingGate(t, passthroughGate)

		r, _, err := g("deezer", hit)

		Expect(err).ToNot(HaveOccurred())
		Expect(r).ToNot(BeNil())
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "external:deezer", Outcome: outcomeHit, Detail: "http://img"},
		}))
	})

	It("records a miss for a not-found", func() {
		t := &chainTrace{}
		_, _, _ = tracingGate(t, passthroughGate)("deezer", miss)
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeMiss))
	})

	It("records a miss for a model not-found", func() {
		t := &chainTrace{}
		notFound := func() (io.ReadCloser, string, error) { return nil, "", model.ErrNotFound }
		_, _, _ = tracingGate(t, passthroughGate)("deezer", notFound)
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeMiss),
			"both not-found flavours are definitive answers, not faults")
	})

	It("records an error with its reason", func() {
		t := &chainTrace{}
		_, _, _ = tracingGate(t, passthroughGate)("apple-music", boom)
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeError))
		Expect(t.Steps()[0].Detail).To(ContainSubstring("429"))
	})

	It("records skipped when the breaker is open", func() {
		t := &chainTrace{}
		open := func(string, func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
			return nil, "", errBreakerOpen
		}
		_, _, _ = tracingGate(t, open)("apple-music", hit)
		Expect(t.Steps()[0].Outcome).To(Equal(outcomeSkipped))
		Expect(t.Steps()[0].Detail).To(ContainSubstring("circuit breaker"))
	})

	It("never calls the agent in offline mode", func() {
		t := &chainTrace{}
		called := false
		counting := func() (io.ReadCloser, string, error) {
			called = true
			return hit()
		}

		_, _, err := offlineGate(t)("deezer", counting)

		Expect(called).To(BeFalse(), "offline mode must not perform external requests")
		Expect(err).To(MatchError(errOfflineSkipped))
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "external:deezer", Outcome: outcomeWouldTry},
		}))
	})
})

var _ = Describe("resolveAlbum tracing", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		albumRepo  *tests.MockAlbumRepo
		folderRepo *fakeFolderRepo
		ffm        *tests.MockFFmpeg
		ag         *agents.Agents
		t          *chainTrace
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CoverArtPriority = "cover.jpg, embedded"
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		albumRepo = tests.CreateMockAlbumRepo()
		folderRepo = &fakeFolderRepo{}
		ds = &tests.MockDataStore{
			MockedAlbum:   albumRepo,
			MockedFolder:  folderRepo,
			MockedLibrary: libRepo,
		}
		ffm = tests.NewMockFFmpeg("")
		ag = agents.GetAgents(&tests.MockDataStore{}, nil)
		t = &chainTrace{}
		ctx = withTrace(context.Background(), t)
	})

	It("records a pattern skipped because the album folder holds no images", func() {
		albumRepo.SetData(model.Albums{
			{ID: "al1", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})

		res, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(t.Steps()).To(HaveLen(2), "a configured pattern must appear even when the chain never evaluated it")
		Expect(t.Steps()[0]).To(Equal(traceStep{
			Candidate: "cover.jpg", Outcome: outcomeSkipped, Detail: "no images in album folder",
		}))
		Expect(t.Steps()[1].Candidate).To(Equal("embedded"))
		Expect(t.Steps()[1].Outcome).To(Equal(outcomeHit))
	})

	It("records an evaluated pattern that matched nothing as a miss, not a skip", func() {
		folderRepo.result = []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"artist.png"},
		}}
		albumRepo.SetData(model.Albums{
			{ID: "al3", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})

		res, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(t.Steps()[0]).To(Equal(traceStep{Candidate: "cover.jpg", Outcome: outcomeMiss}),
			"the folder was searched and held no cover.jpg, which is not the same as never looking")
	})

	It("ignores an empty priority token", func() {
		conf.Server.CoverArtPriority = "cover.jpg,"
		albumRepo.SetData(model.Albums{{ID: "al2", Name: "Album", FolderIDs: []string{"f1"}}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "cover.jpg", Outcome: outcomeSkipped, Detail: "no images in album folder"},
		}))
	})
})

var _ = Describe("resolveArtist tracing", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		artistRepo *tests.MockArtistRepo
		albumRepo  *tests.MockAlbumRepo
		folderRepo *fakeFolderRepo
		ffm        *tests.MockFFmpeg
		ag         *agents.Agents
		t          *chainTrace
		repoRoot   string
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
		conf.Server.ArtistArtPriority = "album/artist.*"
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		artistRepo = tests.CreateMockArtistRepo()
		albumRepo = tests.CreateMockAlbumRepo()
		folderRepo = &fakeFolderRepo{}
		ds = &tests.MockDataStore{
			MockedArtist:  artistRepo,
			MockedAlbum:   albumRepo,
			MockedFolder:  folderRepo,
			MockedLibrary: libRepo,
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
		Expect(t.Steps()[0]).To(Equal(traceStep{Candidate: "upload", Outcome: outcomeMiss}))
	})

	It("labels each step with the configured priority token", func() {
		folderRepo.result = []model.Folder{{
			LibraryPath: testFileLibPath(repoRoot),
			Path:        "tests/fixtures/artist/an-album",
			ImageFiles:  []string{"artist.png"},
		}}
		artistRepo.SetData(model.Artists{{ID: "ar4", Name: "Artist"}})
		albumRepo.All = model.Albums{{ID: "al9", Name: "Album", LibraryID: 0, FolderIDs: []string{"f1"}}}

		res, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar4"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(t.Steps()).To(HaveLen(2))
		Expect(t.Steps()[1].Candidate).To(Equal("album/artist.*"),
			"the step must be labelled with the priority token, not the pattern it was rewritten into")
		Expect(t.Steps()[1].Outcome).To(Equal(outcomeHit))
		Expect(filepath.ToSlash(t.Steps()[1].Detail)).To(HaveSuffix("tests/fixtures/artist/an-album/artist.png"))
	})

	It("records a configured pattern that could not be evaluated", func() {
		artistRepo.SetData(model.Artists{{ID: "ar5", Name: "Artist"}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar5"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "upload", Outcome: outcomeMiss},
			{Candidate: "album/artist.*", Outcome: outcomeSkipped, Detail: "artist has no albums"},
		}), "a configured pattern that was never evaluated must still appear, and say why")
	})

	It("records why an artist folder pattern was skipped", func() {
		conf.Server.ArtistArtPriority = "artist.*"
		artistRepo.SetData(model.Artists{{ID: "ar6", Name: "Artist"}})
		albumRepo.All = model.Albums{{ID: "al10", Name: "Album", LibraryID: 0, FolderIDs: []string{"f1"}}}

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar6"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "upload", Outcome: outcomeMiss},
			{Candidate: "artist.*", Outcome: outcomeSkipped, Detail: "no artist folder"},
		}))
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

var _ = Describe("NewTracingResolver", func() {
	var (
		ds          *tests.MockDataStore
		albumRepo   *tests.MockAlbumRepo
		artistRepo  *tests.MockArtistRepo
		artworkRepo *tests.MockArtworkRepo
		queueRepo   *tests.MockArtworkQueueRepo
		ffm         *tests.MockFFmpeg
		t           *ChainTrace
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.CoverArtPriority = "external, embedded"
		conf.Server.ArtistArtPriority = "external"
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		albumRepo = tests.CreateMockAlbumRepo()
		artistRepo = tests.CreateMockArtistRepo()
		artworkRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds = &tests.MockDataStore{
			MockedAlbum:        albumRepo,
			MockedArtist:       artistRepo,
			MockedFolder:       &fakeFolderRepo{},
			MockedLibrary:      libRepo,
			MockedArtwork:      artworkRepo,
			MockedArtworkQueue: queueRepo,
		}
		ffm = tests.NewMockFFmpeg("")
		t = &ChainTrace{}
	})

	It("uses the offline gate when live is false", func() {
		r := NewTracingResolver(nil, nil, nil, t, false)
		Expect(r).ToNot(BeNil())
	})

	Context("offline", func() {
		var fake *fakeImageAgent

		BeforeEach(func() {
			fake = &fakeImageAgent{name: "offline-probe"}
			albumRepo.SetData(model.Albums{{
				ID: "al1", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"},
			}})
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
		})

		It("reports the external tier without asking any agent", func() {
			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveAlbum(context.Background(), "al1")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(Equal("embedded"))
			Expect(fake.albumCalls).To(BeZero(), "offline mode must not add load to an external provider")
			Expect(t.Steps()).To(ContainElement(TraceStep{Candidate: "external:offline-probe", Outcome: outcomeWouldTry}))
		})

		It("records the local chain steps too", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveAlbum(context.Background(), "al1")

			Expect(err).ToNot(HaveOccurred())
			last := t.Steps()[len(t.Steps())-1]
			Expect(last.Candidate).To(Equal("embedded"), "the local chain must be traced, not just the external gate")
			Expect(last.Outcome).To(Equal(outcomeHit))
		})

		It("never persists artwork state", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveAlbum(context.Background(), "al1")

			Expect(err).ToNot(HaveOccurred())
			Expect(artworkRepo.ItemData).To(BeEmpty(),
				"an offline resolution carries extError, which must never be recorded as a real provider failure")
			Expect(queueRepo.Data).To(BeEmpty())
		})

		It("resolves an artist without persisting anything", func() {
			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveArtist(context.Background(), "ar1")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(BeEmpty())
			Expect(fake.artistCalls).To(BeZero())
			Expect(t.Steps()).To(ContainElement(TraceStep{Candidate: "external:offline-probe", Outcome: outcomeWouldTry}))
			Expect(artworkRepo.ItemData).To(BeEmpty())
			Expect(queueRepo.Data).To(BeEmpty())
		})

		It("closes the reader it does not hand back", func() {
			conf.Server.CoverArtPriority = "embedded"
			ffm = tests.NewMockFFmpeg("fake image bytes")
			albumRepo.SetData(model.Albums{{
				ID: "al2", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/no-such-file.mp3", FolderIDs: []string{"f1"},
			}})

			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveAlbum(context.Background(), "al2")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(Equal("embedded"))
			Expect(ffm.IsClosed()).To(BeTrue(), "nothing downstream closes it, so a leak is one file handle per invocation")
		})

		It("propagates a lookup error", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, false).ResolveAlbum(context.Background(), "nope")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	It("asks the agents when live is true", func() {
		fake := &fakeImageAgent{name: "live-probe", err: agents.ErrNotFound}
		albumRepo.SetData(model.Albums{{
			ID: "al1", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"},
		}})

		_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).ResolveAlbum(context.Background(), "al1")

		Expect(err).ToNot(HaveOccurred())
		Expect(fake.albumCalls).To(Equal(1))
		Expect(t.Steps()).To(ContainElement(TraceStep{Candidate: "external:live-probe", Outcome: outcomeMiss}))
	})
})
