package cmd

import (
	"context"
	"errors"
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
