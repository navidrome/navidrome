package cmd

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseArtworkKind", func() {
	It("accepts a supported kind", func() {
		k, err := parseArtworkKind("ar")
		Expect(err).ToNot(HaveOccurred())
		Expect(k).To(Equal(model.KindArtistArtwork))
	})

	It("rejects an unknown kind and lists the valid ones", func() {
		_, err := parseArtworkKind("zz")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ar"))
		Expect(err.Error()).To(ContainSubstring("al"))
	})

	It("rejects a known kind that has no artwork command", func() {
		_, err := parseArtworkKind("mf")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("explainResult", func() {
	It("reports the winning source", func() {
		steps := []artwork.TraceStep{{Candidate: "folder", Outcome: "hit", Detail: "/music/a.jpg"}}
		Expect(explainResult("folder", steps)).To(ContainSubstring("resolved from folder"))
	})

	It("reports not resolved when every candidate was tried and missed", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "miss"},
			{Candidate: "external:deezer", Outcome: "miss"},
		}
		Expect(explainResult("", steps)).To(Equal("not resolved"))
	})

	It("reports indeterminate when an external lookup failed transiently", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "miss"},
			{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
		}
		Expect(explainResult("", steps)).To(ContainSubstring("indeterminate"),
			"a failed network call is not evidence that the item has no artwork")
	})

	It("qualifies a win a skipped higher-priority external candidate could have taken", func() {
		steps := []artwork.TraceStep{
			{Candidate: "external:deezer", Outcome: "would-try"},
			{Candidate: "artist.*", Outcome: "hit", Detail: "/music/artist.jpg"},
		}
		res := explainResult("artist.*", steps)
		Expect(res).To(ContainSubstring("resolved from artist.*"))
		Expect(res).To(ContainSubstring("--live"),
			"offline, the winner is only the winner because the external tier was skipped")
	})

	It("does not qualify a win that no skipped candidate outranked", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "hit"},
			{Candidate: "external:deezer", Outcome: "would-try"},
		}
		Expect(explainResult("artist.*", steps)).To(Equal("resolved from artist.*"))
	})

	It("reports indeterminate when external agents were never called", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "miss"},
			{Candidate: "external:deezer", Outcome: "would-try"},
		}
		Expect(explainResult("", steps)).To(ContainSubstring("indeterminate"),
			"an offline run must not claim an item is unresolvable when external agents were skipped")
	})
})

var _ = Describe("walksPriorityChain", func() {
	It("is true for the kinds the resolver walks", func() {
		Expect(walksPriorityChain(model.KindArtistArtwork)).To(BeTrue())
		Expect(walksPriorityChain(model.KindAlbumArtwork)).To(BeTrue())
	})

	It("is false for the kinds resolved directly", func() {
		Expect(walksPriorityChain(model.KindPlaylistArtwork)).To(BeFalse())
		Expect(walksPriorityChain(model.KindRadioArtwork)).To(BeFalse())
	})
})

var _ = Describe("formatExplain", func() {
	var rep explainReport

	BeforeEach(func() {
		rep = explainReport{
			kind:         model.KindArtistArtwork,
			id:           "ar-1",
			name:         "Radiohead",
			priorityName: "ArtistArtPriority",
			priority:     "external, artist.*",
			agents:       "lastfm,spotify",
			walksChain:   true,
			steps: []artwork.TraceStep{
				{Candidate: "upload", Outcome: "skipped", Detail: "no uploaded image"},
				{Candidate: "external:deezer", Outcome: "would-try"},
			},
			source: "",
		}
	})

	It("prints every block", func() {
		out := formatExplain(rep)
		for _, block := range []string{"Item", "Stored", "Queue", "Config", "Chain", "Result"} {
			Expect(out).To(ContainSubstring(block))
		}
		Expect(out).To(ContainSubstring("Radiohead"))
		Expect(out).To(ContainSubstring("ar-1"))
		Expect(out).To(ContainSubstring("ArtistArtPriority"))
		Expect(out).To(ContainSubstring("lastfm,spotify"))
		Expect(out).To(ContainSubstring("external:deezer"))
		Expect(out).To(ContainSubstring("would-try"))
		Expect(out).To(ContainSubstring("indeterminate"))
	})

	It("reports the absence of stored state and of a queue row", func() {
		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("no artwork state recorded"))
		Expect(out).To(ContainSubstring("not queued"))
	})

	It("prints the stored state and the queue row when they exist", func() {
		attempted := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
		rep.stored = &model.ItemArtwork{Source: "folder", Hash: "abc123",
			SourcePath: "/music/cover.jpg", AttemptedAt: attempted}
		rep.queued = &model.ArtworkQueueItem{Priority: model.ArtworkPriorityScan, Attempts: 2,
			RetryAt: attempted.Add(time.Hour)}
		rep.source = "folder"

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("abc123"))
		Expect(out).To(ContainSubstring("/music/cover.jpg"))
		Expect(out).To(ContainSubstring("2026-08-13T10:00:00Z"))
		Expect(out).To(ContainSubstring("50"))
		Expect(out).To(ContainSubstring("resolved from folder"))
	})

	It("marks a known-absent stored state instead of printing an empty hash", func() {
		rep.stored = &model.ItemArtwork{AttemptedAt: time.Now()}
		Expect(formatExplain(rep)).To(ContainSubstring("absent"))
	})

	It("reports a failed walk as failed, not as unresolved", func() {
		rep.resolveErr = errors.New("no such directory")

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("resolution failed: no such directory"))
		Expect(out).ToNot(ContainSubstring("indeterminate"))
		Expect(out).To(ContainSubstring("would-try"), "the steps taken before the failure still print")
	})

	It("says a kind that does not walk a chain has no chain, without an empty table", func() {
		rep.kind = model.KindPlaylistArtwork
		rep.walksChain = false
		rep.steps = nil
		rep.priorityName, rep.priority = "CoverArtPriority", "cover.*, embedded"

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("does not walk a priority chain"))
		Expect(out).ToNot(ContainSubstring("CANDIDATE"),
			"an empty chain table reads as 'nothing was tried', which is false")
		Expect(out).ToNot(ContainSubstring("not resolved"),
			"nothing was resolved because nothing was attempted")
		Expect(out).ToNot(ContainSubstring("CoverArtPriority"),
			"the priority chain config does not govern this kind")
	})
})

var _ = Describe("artwork refresh command", func() {
	It("requires at least a kind and one id", func() {
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{"ar"})).To(HaveOccurred())
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{"ar", "id1"})).ToNot(HaveOccurred())
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{"ar", "id1", "id2"})).ToNot(HaveOccurred())
	})
})

var _ = Describe("artwork reprocess selection", func() {
	It("errors when no selector is given", func() {
		_, err := selectedKinds(nil, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--all"))
	})

	It("returns every kind for --all", func() {
		ks, err := selectedKinds(nil, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(ConsistOf(artworkKinds))
	})

	It("returns only the named kinds", func() {
		ks, err := selectedKinds([]string{"ar"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(Equal([]model.Kind{model.KindArtistArtwork}))
	})

	It("rejects an unknown kind", func() {
		_, err := selectedKinds([]string{"zz"}, false)
		Expect(err).To(HaveOccurred())
	})

	It("counts a repeated kind once", func() {
		ks, err := selectedKinds([]string{"ar", "ar"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(Equal([]model.Kind{model.KindArtistArtwork}))
	})
})

var _ = Describe("reprocessSelectsAll", func() {
	It("keeps a named kind selection", func() {
		Expect(reprocessSelectsAll([]string{"ar"}, nil, false)).To(BeFalse())
		Expect(reprocessSelectsAll([]string{"ar"}, []string{"folder"}, false)).To(BeFalse())
	})

	It("targets every kind for a source filter given without a kind", func() {
		Expect(reprocessSelectsAll(nil, []string{"folder"}, false)).To(BeTrue())
	})

	It("targets every kind for --all", func() {
		Expect(reprocessSelectsAll(nil, nil, true)).To(BeTrue())
		Expect(reprocessSelectsAll([]string{"ar"}, nil, true)).To(BeTrue())
	})

	It("is false with no selector, leaving selectedKinds to reject it", func() {
		Expect(reprocessSelectsAll(nil, nil, false)).To(BeFalse())
	})
})

var _ = Describe("repositorySources", func() {
	It("maps the user-facing absent name onto the stored empty source", func() {
		Expect(repositorySources([]string{"absent", "folder"})).To(Equal([]string{"", "folder"}))
	})

	It("keeps an empty selection empty, meaning every source", func() {
		Expect(repositorySources(nil)).To(BeEmpty())
	})
})

var _ = Describe("promptConfirm", func() {
	var out strings.Builder

	BeforeEach(func() { out.Reset() })

	It("states the external cost and accepts an explicit yes", func() {
		Expect(promptConfirm(strings.NewReader("y\n"))(&out, 42, 7)).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("re-resolve 42 items"))
		Expect(out.String()).To(ContainSubstring("7 external lookups"))
	})

	It("defaults to no on anything else", func() {
		Expect(promptConfirm(strings.NewReader("\n"))(&out, 1, 1)).To(BeFalse())
		Expect(promptConfirm(strings.NewReader("nope\n"))(&out, 1, 1)).To(BeFalse())
		Expect(promptConfirm(strings.NewReader(""))(&out, 1, 1)).To(BeFalse())
	})

	It("drops the external clause when no lookup will be made", func() {
		Expect(promptConfirm(strings.NewReader("y\n"))(&out, 3, 0)).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("re-resolve 3 items."))
		Expect(out.String()).ToNot(ContainSubstring("external lookups"))
	})
})

var _ = Describe("reprocessConfirm", func() {
	var out strings.Builder

	BeforeEach(func() { out.Reset() })

	It("prompts when --yes was not given", func() {
		Expect(reprocessConfirm(false, strings.NewReader("n\n"))(&out, 5, 5)).To(BeFalse())
		Expect(out.String()).To(ContainSubstring("Continue?"))
	})

	It("bypasses the prompt only for --yes", func() {
		Expect(reprocessConfirm(true, strings.NewReader(""))(&out, 5, 5)).To(BeTrue())
		Expect(out.String()).To(BeEmpty(), "--yes must not print a prompt it never reads")
	})
})

var _ = Describe("reprocessArtwork", func() {
	var ds *tests.MockDataStore
	var art *tests.MockArtworkRepo
	var queue *tests.MockArtworkQueueRepo
	var out strings.Builder
	ctx := context.Background()
	kinds := []model.Kind{model.KindArtistArtwork, model.KindAlbumArtwork}
	accept := func(io.Writer, int64, int64) bool { return true }
	decline := func(io.Writer, int64, int64) bool { return false }

	put := func(kind model.Kind, id, source string) {
		Expect(art.PutItemArtwork(&model.ItemArtwork{ItemKind: kind.Prefix(), ItemID: id,
			ImageType: model.ImageTypePrimary, Hash: "h" + id, Source: source})).To(Succeed())
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		art = ds.Artwork(ctx).(*tests.MockArtworkRepo)
		queue = ds.ArtworkQueue(ctx).(*tests.MockArtworkQueueRepo)
		out.Reset()
		put(model.KindArtistArtwork, "ar-1", "external:deezer")
		put(model.KindArtistArtwork, "ar-2", "")
		put(model.KindAlbumArtwork, "al-1", "external:deezer")
		put(model.KindAlbumArtwork, "al-2", "folder")
	})

	It("previews the per-kind breakdown and queues nothing on a dry run", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("external:deezer"))
		Expect(out.String()).To(ContainSubstring("artist"))
		Expect(out.String()).To(ContainSubstring("album"))
		Expect(out.String()).To(ContainSubstring("TOTAL"))
		Expect(out.String()).To(ContainSubstring("Dry run"))
		Expect(queue.Count()).To(BeZero())
	})

	It("queues nothing when the operator declines", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, nil, false, decline, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Aborted"))
		Expect(queue.Count()).To(BeZero())
	})

	It("queues the matching items at recheck priority, leaving their artwork state alone", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, false, accept, &out)).To(Succeed())

		Expect(queue.Count()).To(Equal(int64(2)))
		queued, err := queue.Get(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(queued.Priority).To(Equal(model.ArtworkPriorityRecheck))
		_, err = queue.Get(model.KindAlbumArtwork, "al-2", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "a non-matching source must not be queued")

		stored, err := art.GetItemArtwork(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.Hash).To(Equal("hal-1"), "bulk reprocessing must not blank the current artwork")
	})

	It("targets the absent state", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, []string{""}, false, accept, &out)).To(Succeed())

		Expect(queue.Count()).To(Equal(int64(1)))
		_, err := queue.Get(model.KindArtistArtwork, "ar-2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
	})

	It("reports matched and queued separately when part of the set is already queued", func() {
		Expect(queue.Enqueue(model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar-1",
			ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBump})).To(Succeed())

		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, false, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Queued 1 of 2 matched items"))
		Expect(out.String()).To(ContainSubstring("Already queued, left unchanged: 1"))
		queued, err := queue.Get(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(queued.Priority).To(Equal(model.ArtworkPriorityBump),
			"an already-queued row keeps its priority and backoff")
	})

	It("stops at a selection that matches nothing instead of prompting", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindRadioArtwork}, nil, false,
			func(io.Writer, int64, int64) bool {
				Fail("must not prompt when there is nothing to queue")
				return true
			}, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing"))
		Expect(queue.Count()).To(BeZero())
	})

	It("reports an empty selection as a dry run when one was asked for", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindRadioArtwork}, nil, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing matches"))
		Expect(out.String()).To(ContainSubstring("Dry run"))
	})

	It("shows the external estimate on a dry run, which never reaches the prompt", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindAlbumArtwork}, nil, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("External lookups: up to 2"))
	})

	It("says so when the selection needs no external lookup", func() {
		put(model.KindPlaylistArtwork, "pl-1", "playlist")

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindPlaylistArtwork}, nil, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("External lookups: none"))
	})

	It("counts only the kinds that call an external agent as external cost", func() {
		put(model.KindPlaylistArtwork, "pl-1", "playlist")
		var total, external int64
		capture := func(_ io.Writer, t, e int64) bool { total, external = t, e; return false }

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindAlbumArtwork, model.KindPlaylistArtwork},
			nil, false, capture, &out)).To(Succeed())

		Expect(total).To(Equal(int64(3)))
		Expect(external).To(Equal(int64(2)), "playlist artwork never reaches an external agent")
	})

	It("rejects an unknown source and names the ones in use", func() {
		err := reprocessArtwork(ctx, ds, kinds, []string{"externa:deezer"}, true, accept, &out)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("externa:deezer"))
		Expect(err.Error()).To(ContainSubstring("external:deezer"))
		Expect(err.Error()).To(ContainSubstring("folder"))
		Expect(err.Error()).To(ContainSubstring("absent"), "the empty source prints under its user-facing name")
		Expect(queue.Count()).To(BeZero())
	})

	It("accepts a source another kind uses, letting the empty selection report itself", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindArtistArtwork}, []string{"folder"},
			false, decline, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing matches"),
			"a well-formed filter must not be reported as a typo because of the kinds selected")
		Expect(queue.Count()).To(BeZero())
	})
})

var _ = Describe("refreshItems", func() {
	var ds *tests.MockDataStore
	var queue *tests.MockArtworkQueueRepo
	var art *tests.MockArtworkRepo
	var out strings.Builder
	ctx := context.Background()

	BeforeEach(func() {
		albums := tests.CreateMockAlbumRepo()
		albums.SetData(model.Albums{{ID: "al-1"}, {ID: "al-3"}})
		ds = &tests.MockDataStore{MockedAlbum: albums}
		art = ds.Artwork(ctx).(*tests.MockArtworkRepo)
		queue = ds.ArtworkQueue(ctx).(*tests.MockArtworkQueueRepo)
		out.Reset()
	})

	It("clears the stored state and queues each id at Bump priority", func() {
		Expect(art.PutItemArtwork(&model.ItemArtwork{ItemKind: model.KindAlbumArtwork.Prefix(),
			ItemID: "al-1", ImageType: model.ImageTypePrimary, Hash: "abc123"})).To(Succeed())

		Expect(refreshItems(ctx, ds, model.KindAlbumArtwork, []string{"al-1", "al-3"}, &out)).To(BeZero())

		_, err := art.GetItemArtwork(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		queued, err := queue.Get(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(queued.Priority).To(Equal(model.ArtworkPriorityBump))
		Expect(out.String()).To(Equal("al/al-1: queued\nal/al-3: queued\n"))
	})

	It("skips an id that does not exist instead of queuing it", func() {
		Expect(refreshItems(ctx, ds, model.KindAlbumArtwork, []string{"al-2"}, &out)).To(Equal(1))

		_, err := queue.Get(model.KindAlbumArtwork, "al-2", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "a typo must not leave an orphan queue row")
		Expect(out.String()).To(BeEmpty())
	})

	It("continues past a failing id and counts the failures", func() {
		Expect(refreshItems(ctx, ds, model.KindAlbumArtwork,
			[]string{"al-1", "al-2", "al-3"}, &out)).To(Equal(1))

		Expect(out.String()).To(Equal("al/al-1: queued\nal/al-3: queued\n"),
			"the ids after a failure are still refreshed")
	})
})
