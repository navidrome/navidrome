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

var _ = Describe("trace vocabulary", func() {
	// The CLI renders these verbatim and branches on them; a value change is a change to
	// what `artwork explain` tells an operator, so it must be made deliberately.
	It("pins the wire values the CLI reads", func() {
		Expect([]Outcome{
			OutcomeHit, OutcomeMiss, OutcomeUnreadable, OutcomeSkipped, OutcomeError,
		}).To(Equal([]Outcome{"hit", "miss", "unreadable", "skipped", "error"}))
		Expect(externalCandidate).To(Equal("external"))
		Expect(ExternalPrefix).To(Equal("external:"))
	})
})

var _ = Describe("encodeSteps/DecodeTrace", func() {
	It("round-trips a trace", func() {
		steps := []TraceStep{
			{Candidate: "cover.png", Outcome: OutcomeMiss},
			{Candidate: "cover.*", Outcome: OutcomeHit, Detail: "/music/a/cover.jpg"},
		}
		Expect(DecodeTrace(encodeSteps(steps, ""), "")).To(Equal(steps))
	})

	It("encodes an empty trace as an empty JSON array", func() {
		Expect(encodeSteps(nil, "")).To(Equal("[]"))
		Expect(DecodeTrace("[]", "")).To(BeEmpty())
	})

	It("tolerates a row written before the column existed", func() {
		Expect(DecodeTrace("", "")).To(BeEmpty())
	})

	// The hit detail repeats source_path byte for byte, and that column is on the same row.
	It("drops a hit detail that repeats sourcePath, and restores it on read", func() {
		path := "/music/artist/album/cover.jpg"
		steps := []TraceStep{{Candidate: "cover.*", Outcome: OutcomeHit, Detail: path}}
		encoded := encodeSteps(steps, path)
		Expect(encoded).NotTo(ContainSubstring(path))
		Expect(DecodeTrace(encoded, path)).To(Equal(steps))
	})

	It("keeps a detail that differs from sourcePath", func() {
		steps := []TraceStep{{Candidate: "external:deezer", Outcome: OutcomeHit, Detail: "https://cdn/x.jpg"}}
		Expect(DecodeTrace(encodeSteps(steps, "/music/a/cover.jpg"), "/music/a/cover.jpg")).To(Equal(steps))
	})

	// A row past ~1kB spills to an overflow page on these WITHOUT ROWID tables, which would
	// slow every scan; Detail is an error string on the failure paths, so it needs a bound.
	It("bounds a detail so one long error cannot inflate the row", func() {
		steps := []TraceStep{{Candidate: "decode", Outcome: OutcomeError, Detail: strings.Repeat("x", 5000)}}

		got := DecodeTrace(encodeSteps(steps, ""), "")

		Expect(len(got[0].Detail)).To(BeNumerically("<=", 210))
		Expect(got[0].Detail).To(HaveSuffix("..."))
		Expect(got[0].Candidate).To(Equal("decode"), "truncating the detail must not disturb the step")
	})

	It("only restores sourcePath onto a detail-less hit", func() {
		steps := []TraceStep{{Candidate: "cover.*", Outcome: OutcomeMiss}}
		Expect(DecodeTrace(encodeSteps(steps, "/music/a/cover.jpg"), "/music/a/cover.jpg")).To(Equal(steps))
	})
})

var _ = Describe("chainTrace", func() {
	It("returns nil when no trace is attached", func() {
		Expect(traceFrom(context.Background())).To(BeNil())
	})

	It("collects steps in order", func() {
		t := &ChainTrace{}
		ctx := withTrace(context.Background(), t)

		traceFrom(ctx).add(TraceStep{Candidate: "cover.*", Outcome: OutcomeMiss})
		traceFrom(ctx).add(TraceStep{Candidate: "embedded", Outcome: OutcomeHit, Detail: "/music/a.flac"})

		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "cover.*", Outcome: OutcomeMiss},
			{Candidate: "embedded", Outcome: OutcomeHit, Detail: "/music/a.flac"},
		}))

		s := t.Steps()
		s[0].Candidate = "mutated"
		Expect(t.Steps()[0].Candidate).To(Equal("cover.*"))
	})

	It("does not panic when the trace is nil", func() {
		var t *ChainTrace
		Expect(func() { t.add(TraceStep{Candidate: "cover.*", Outcome: OutcomeMiss}) }).ToNot(Panic())
		Expect(t.Steps()).To(BeEmpty(), "a nil trace collects nothing, so reading it must be as safe as writing it")
	})

	It("is safe to use concurrently", func() {
		t := &ChainTrace{}
		done := make(chan struct{})
		for range 10 {
			go func() {
				defer GinkgoRecover()
				t.add(TraceStep{Candidate: "x", Outcome: OutcomeMiss})
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
		t := &ChainTrace{}
		c := chainState{trace: t}

		_, ok := c.try("cover.*", resolution{}, false)

		Expect(ok).To(BeFalse())
		Expect(t.Steps()).To(Equal([]TraceStep{{Candidate: "cover.*", Outcome: OutcomeMiss}}))
	})

	It("records unreadable when the candidate existed but could not be read", func() {
		t := &ChainTrace{}
		c := chainState{trace: t}

		_, ok := c.try("cover.*", resolution{localError: true}, false)

		Expect(ok).To(BeFalse())
		Expect(t.Steps()).To(HaveLen(1))
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeUnreadable),
			"a candidate that existed and failed to decode must be distinguishable from one that was absent")
	})

	It("records a hit with the backing path", func() {
		t := &ChainTrace{}
		c := chainState{trace: t}

		res, ok := c.try("embedded", resolution{reader: nil, source: "embedded", sourcePath: "/music/a.flac"}, true)

		Expect(ok).To(BeTrue())
		Expect(res.source).To(Equal("embedded"))
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "embedded", Outcome: OutcomeHit, Detail: "/music/a.flac"},
		}))
	})
})

var _ = Describe("external agent tracing", func() {
	var (
		t    *ChainTrace
		ctx  context.Context
		body io.ReadCloser
	)
	BeforeEach(func() {
		t = &ChainTrace{}
		ctx = withTrace(context.Background(), t)
		body = io.NopCloser(strings.NewReader("x"))
	})

	It("records a hit with the image path", func() {
		recordAgent(ctx, "deezer", body, "http://img", nil)
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "external:deezer", Outcome: OutcomeHit, Detail: "http://img"},
		}))
	})

	It("records a miss for a not-found", func() {
		recordAgent(ctx, "deezer", nil, "", agents.ErrNotFound)
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeMiss))
	})

	It("records a miss for a model not-found", func() {
		recordAgent(ctx, "deezer", nil, "", model.ErrNotFound)
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeMiss),
			"both not-found flavours are definitive answers, not faults")
	})

	It("records an error with its reason", func() {
		recordAgent(ctx, "apple-music", nil, "", errors.New("returned status 429"))
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeError))
		Expect(t.Steps()[0].Detail).To(ContainSubstring("429"))
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
		t          *ChainTrace
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
		t = &ChainTrace{}
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
		Expect(t.Steps()[0]).To(Equal(TraceStep{
			Candidate: "cover.jpg", Outcome: OutcomeSkipped, Detail: "no images in album folder",
		}))
		Expect(t.Steps()[1].Candidate).To(Equal("embedded"))
		Expect(t.Steps()[1].Outcome).To(Equal(OutcomeHit))
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
		Expect(t.Steps()[0]).To(Equal(TraceStep{Candidate: "cover.jpg", Outcome: OutcomeMiss}),
			"the folder was searched and held no cover.jpg, which is not the same as never looking")
	})

	It("ignores an empty priority token", func() {
		conf.Server.CoverArtPriority = "cover.jpg,"
		albumRepo.SetData(model.Albums{{ID: "al2", Name: "Album", FolderIDs: []string{"f1"}}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "cover.jpg", Outcome: OutcomeSkipped, Detail: "no images in album folder"},
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
		t          *ChainTrace
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
		t = &ChainTrace{}
		ctx = withTrace(context.Background(), t)
	})

	It("records the upload short-circuit as a hit", func() {
		path := uploadPath("ar1_test.jpg")
		artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist", UploadedImage: "ar1_test.jpg"}})

		res, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar1"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(t.Steps()).To(Equal([]TraceStep{{Candidate: "upload", Outcome: OutcomeHit, Detail: path}}))
	})

	It("records an upload miss before walking the chain", func() {
		artistRepo.SetData(model.Artists{{ID: "ar2", Name: "Artist"}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar2"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()[0]).To(Equal(TraceStep{Candidate: "upload", Outcome: OutcomeMiss}))
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
		Expect(t.Steps()[1].Outcome).To(Equal(OutcomeHit))
		Expect(filepath.ToSlash(t.Steps()[1].Detail)).To(HaveSuffix("tests/fixtures/artist/an-album/artist.png"))
	})

	It("records a configured pattern that could not be evaluated", func() {
		artistRepo.SetData(model.Artists{{ID: "ar5", Name: "Artist"}})

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar5"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "upload", Outcome: OutcomeMiss},
			{Candidate: "album/artist.*", Outcome: OutcomeSkipped, Detail: "artist has no albums"},
		}), "a configured pattern that was never evaluated must still appear, and say why")
	})

	It("records why an artist folder pattern was skipped", func() {
		conf.Server.ArtistArtPriority = "artist.*"
		artistRepo.SetData(model.Artists{{ID: "ar6", Name: "Artist"}})
		albumRepo.All = model.Albums{{ID: "al10", Name: "Album", LibraryID: 0, FolderIDs: []string{"f1"}}}

		_, err := newResolver(ds, ag, ffm, nil).resolve(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar6"})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "upload", Outcome: OutcomeMiss},
			{Candidate: "artist.*", Outcome: OutcomeSkipped, Detail: "no artist folder"},
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
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeUnreadable),
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

	Context("resolving", func() {
		var fake *fakeImageAgent

		BeforeEach(func() {
			// Misses, so the chain falls through to the local tier and both are traced.
			fake = &fakeImageAgent{name: "probe", err: agents.ErrNotFound}
			albumRepo.SetData(model.Albums{{
				ID: "al1", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"},
			}})
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
		})

		It("asks the agents and records what each answered", func() {
			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindAlbumArtwork, "al1")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(Equal("embedded"))
			Expect(fake.albumCalls).To(Equal(1))
			Expect(t.Steps()).To(ContainElement(TraceStep{Candidate: "external:probe", Outcome: OutcomeMiss}))
		})

		It("records the local chain steps too", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindAlbumArtwork, "al1")

			Expect(err).ToNot(HaveOccurred())
			last := t.Steps()[len(t.Steps())-1]
			Expect(last.Candidate).To(Equal("embedded"), "the local chain must be traced, not just the external tier")
			Expect(last.Outcome).To(Equal(OutcomeHit))
		})

		It("never persists artwork state", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindAlbumArtwork, "al1")

			Expect(err).ToNot(HaveOccurred())
			Expect(artworkRepo.ItemData).To(BeEmpty(),
				"explain is read-only; a diagnostic walk must never become the stored answer")
			Expect(queueRepo.Data).To(BeEmpty())
		})

		It("resolves an artist without persisting anything", func() {
			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindArtistArtwork, "ar1")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(BeEmpty())
			Expect(fake.artistCalls).To(Equal(1))
			Expect(artworkRepo.ItemData).To(BeEmpty())
			Expect(queueRepo.Data).To(BeEmpty())
		})

		It("closes the reader it does not hand back", func() {
			conf.Server.CoverArtPriority = "embedded"
			ffm = tests.NewMockFFmpeg("fake image bytes")
			albumRepo.SetData(model.Albums{{
				ID: "al2", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/no-such-file.mp3", FolderIDs: []string{"f1"},
			}})

			source, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindAlbumArtwork, "al2")

			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(Equal("embedded"))
			Expect(ffm.IsClosed()).To(BeTrue(), "nothing downstream closes it, so a leak is one file handle per invocation")
		})

		// Serving falls back disc -> album and track -> disc -> album. The resolver does not, but
		// if it ever did, an explain without --live would start calling providers uninvited.
		It("cannot reach a provider without live, whatever the chain does", func() {
			conf.Server.DiscArtPriority = "external, cover.*"
			conf.Server.CoverArtPriority = "external, cover.*"
			conf.Server.EnableMediaFileCoverArt = true
			mfRepo := tests.CreateMockMediaFileRepo()
			mfRepo.SetData(model.MediaFiles{{ID: "mf1", LibraryID: 0, HasCoverArt: true,
				Path: "tests/fixtures/artist/an-album/test.mp3"}})
			ds.MockedMediaFile = mfRepo
			offline := NewTracingResolver(ds, imageAgents(fake), ffm, t, false)

			_, err := offline.Resolve(context.Background(), model.KindDiscArtwork, "al1:1")
			Expect(err).ToNot(HaveOccurred())
			_, err = offline.Resolve(context.Background(), model.KindMediaFileArtwork, "mf1")
			Expect(err).ToNot(HaveOccurred())

			Expect(fake.albumCalls).To(BeZero())
			Expect(fake.artistCalls).To(BeZero())
		})

		It("propagates a lookup error", func() {
			_, err := NewTracingResolver(ds, imageAgents(fake), ffm, t, true).Resolve(context.Background(), model.KindAlbumArtwork, "nope")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})
})

var _ = Describe("resolveDisc tracing", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		albumRepo  *tests.MockAlbumRepo
		folderRepo *fakeFolderRepo
		ffm        *tests.MockFFmpeg
		t          *ChainTrace
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		albumRepo = tests.CreateMockAlbumRepo()
		albumRepo.SetData(model.Albums{{ID: "al1", Name: "Album", FolderIDs: []string{"f1"}}})
		folderRepo = &fakeFolderRepo{}
		mfRepo := tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{{ID: "mf1", AlbumID: "al1", DiscNumber: 2, Path: "tests/fixtures/artist/an-album/test.mp3"}})
		ds = &tests.MockDataStore{
			MockedAlbum:     albumRepo,
			MockedMediaFile: mfRepo,
			MockedFolder:    folderRepo,
			MockedLibrary:   libRepo,
		}
		ffm = tests.NewMockFFmpeg("")
		t = &ChainTrace{}
		ctx = withTrace(context.Background(), t)
	})

	It("accounts for every configured entry, including the ones that map to no source", func() {
		conf.Server.DiscArtPriority = "external, discsubtitle, cover.jpg"

		res, err := newResolver(ds, nil, ffm, nil).resolveDisc(ctx, model.DiscArtworkID("al1", 2))

		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).To(BeNil())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "external", Outcome: OutcomeSkipped, Detail: "external sources are not supported for disc artwork"},
			{Candidate: "discsubtitle", Outcome: OutcomeSkipped, Detail: "disc has no subtitle"},
			{Candidate: "cover.jpg", Outcome: OutcomeSkipped, Detail: "no images in album folder"},
		}))
	})

	It("records the entry that won and stops there", func() {
		conf.Server.DiscArtPriority = "disc*.*, cover.jpg, embedded"
		folderRepo.result = []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"cover.jpg"},
		}}

		res, err := newResolver(ds, nil, ffm, nil).resolveDisc(ctx, model.DiscArtworkID("al1", 2))

		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(res.source).To(Equal("folder"))
		Expect(t.Steps()).To(HaveLen(2), "the walk must stop at the winner, and record nothing below it")
		Expect(t.Steps()[0]).To(Equal(TraceStep{Candidate: "disc*.*", Outcome: OutcomeMiss}))
		Expect(t.Steps()[1].Candidate).To(Equal("cover.jpg"))
		Expect(t.Steps()[1].Outcome).To(Equal(OutcomeHit))
		Expect(t.Steps()[1].Detail).To(HaveSuffix(filepath.FromSlash("tests/fixtures/artist/an-album/cover.jpg")))
		Expect(t.Steps()[1].Detail).ToNot(Equal("tests/fixtures/artist/an-album/cover.jpg"),
			"a library-relative path sends the operator looking in the wrong place")
	})

	It("reports an unparseable disc id rather than explaining another disc", func() {
		conf.Server.DiscArtPriority = "cover.jpg"
		_, err := newResolver(ds, nil, ffm, nil).resolveDisc(ctx, "al1")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("resolveMediaFile tracing", func() {
	var (
		ctx    context.Context
		ds     *tests.MockDataStore
		mfRepo *tests.MockMediaFileRepo
		ffm    *tests.MockFFmpeg
		t      *ChainTrace
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableMediaFileCoverArt = true
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		mfRepo = tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{{
			ID: "mf1", Title: "Song", HasCoverArt: true, Path: "tests/fixtures/artist/an-album/test.mp3",
		}})
		ds = &tests.MockDataStore{MockedMediaFile: mfRepo, MockedLibrary: libRepo}
		ffm = tests.NewMockFFmpeg("")
		t = &ChainTrace{}
		ctx = withTrace(context.Background(), t)
	})

	It("separates a disabled setting from a track with nothing embedded", func() {
		conf.Server.EnableMediaFileCoverArt = false

		_, err := newResolver(ds, nil, ffm, nil).resolveMediaFile(ctx, "mf1")

		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "embedded", Outcome: OutcomeSkipped, Detail: "EnableMediaFileCoverArt is off"},
		}))
	})

	It("records a track with no embedded art as a miss", func() {
		mfRepo.SetData(model.MediaFiles{{ID: "mf2", Title: "Song", HasCoverArt: false}})

		_, err := newResolver(ds, nil, ffm, nil).resolveMediaFile(ctx, "mf2")

		Expect(err).ToNot(HaveOccurred())
		Expect(t.Steps()).To(Equal([]TraceStep{
			{Candidate: "embedded", Outcome: OutcomeMiss, Detail: "the track has no embedded cover art"},
		}))
	})

	It("records the embedded hit", func() {
		res, err := newResolver(ds, nil, ffm, nil).resolveMediaFile(ctx, "mf1")

		Expect(err).ToNot(HaveOccurred())
		Expect(res.reader).ToNot(BeNil())
		defer res.reader.Close()
		Expect(res.source).To(Equal("embedded"))
		Expect(t.Steps()).To(HaveLen(1))
		Expect(t.Steps()[0].Outcome).To(Equal(OutcomeHit))
	})
})
