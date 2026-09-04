package artwork_test

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Result", func() {
	It("reports the source when one was found", func() {
		steps := []artwork.TraceStep{{Candidate: "cover.*", Outcome: artwork.OutcomeHit}}
		Expect(artwork.Result("folder", steps)).To(Equal("resolved from folder"))
	})

	It("flags a local win that a failed external lookup could have outranked", func() {
		steps := []artwork.TraceStep{
			{Candidate: artwork.ExternalPrefix + "deezer", Outcome: artwork.OutcomeError, Detail: "timeout"},
			{Candidate: "cover.*", Outcome: artwork.OutcomeHit},
		}
		Expect(artwork.Result("folder", steps)).To(ContainSubstring("indeterminate"))
	})

	It("does not flag an external winner", func() {
		steps := []artwork.TraceStep{
			{Candidate: artwork.ExternalPrefix + "lastfm", Outcome: artwork.OutcomeError},
			{Candidate: artwork.ExternalPrefix + "deezer", Outcome: artwork.OutcomeHit},
		}
		Expect(artwork.Result(artwork.ExternalPrefix+"deezer", steps)).
			To(Equal("resolved from external:deezer"))
	})

	It("is indeterminate when an external lookup failed and nothing resolved", func() {
		steps := []artwork.TraceStep{{Candidate: artwork.ExternalPrefix + "deezer", Outcome: artwork.OutcomeError}}
		Expect(artwork.Result("", steps)).To(ContainSubstring("an external lookup failed"))
	})

	It("is indeterminate when a candidate could not be processed", func() {
		steps := []artwork.TraceStep{{Candidate: "cover.*", Outcome: artwork.OutcomeUnreadable}}
		Expect(artwork.Result("", steps)).To(ContainSubstring("could not be processed"))
	})

	It("reports a clean miss", func() {
		steps := []artwork.TraceStep{{Candidate: "cover.*", Outcome: artwork.OutcomeMiss}}
		Expect(artwork.Result("", steps)).To(Equal("not resolved"))
	})
})

var _ = Describe("ConfigFor", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.ArtistArtPriority = "external"
	})

	It("names the setting for a kind that has one", func() {
		setting, value := artwork.ConfigFor(model.KindArtistArtwork)
		Expect(setting).To(Equal("ArtistArtPriority"))
		Expect(value).To(Equal("external"))
	})

	It("names the setting for the album kind", func() {
		conf.Server.CoverArtPriority = "cover.*, embedded"
		setting, value := artwork.ConfigFor(model.KindAlbumArtwork)
		Expect(setting).To(Equal("CoverArtPriority"))
		Expect(value).To(Equal("cover.*, embedded"))
	})

	It("returns nothing for a kind with no source configuration", func() {
		setting, _ := artwork.ConfigFor(model.KindPlaylistArtwork)
		Expect(setting).To(BeEmpty())
	})
})

var _ = Describe("FormatAgents", func() {
	It("reports none when nothing is configured", func() {
		Expect(artwork.FormatAgents("  ", nil, "")).To(Equal("(none)"))
	})

	It("keeps the configured order and marks what is unavailable", func() {
		got := artwork.FormatAgents("spotify, lastfm", []string{"lastfm"}, "  (* missing)")
		Expect(got).To(Equal("spotify*, lastfm  (* missing)"))
	})

	It("omits the note when every configured agent is available", func() {
		Expect(artwork.FormatAgents("lastfm", []string{"lastfm"}, "  (* missing)")).To(Equal("lastfm"))
	})
})

var _ = Describe("PriorityName", func() {
	It("names a known priority", func() {
		Expect(artwork.PriorityName(model.ArtworkPriorityScan)).To(Equal("scan"))
	})

	It("falls back to the number for an unknown priority", func() {
		Expect(artwork.PriorityName(999)).To(Equal("999"))
	})
})

var _ = Describe("Explain", func() {
	var ds *tests.MockDataStore
	var artRepo *tests.MockArtworkRepo
	var queueRepo *tests.MockArtworkQueueRepo
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds = &tests.MockDataStore{MockedArtwork: artRepo, MockedArtworkQueue: queueRepo}
		Expect(ds.Artist(ctx).Put(&model.Artist{ID: "ar-1", Name: "Radiohead"})).To(Succeed())
	})

	It("returns an error when the item does not exist", func() {
		_, err := artwork.Explain(ctx, ds, nil, model.KindArtistArtwork, "nope", artwork.ExplainOptions{})
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("reads the recorded trace when no walker is supplied", func() {
		// Storage shape mirrors storedStep in trace.go: c=candidate, o=outcome, d=detail.
		trace := `[{"c":"external:deezer","o":"hit","d":"https://cdn/x.jpg"}]`
		Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
			ItemKind: model.KindArtistArtwork.Prefix(), ItemID: "ar-1", ImageType: model.ImageTypePrimary,
			Hash: "abc", Source: "external:deezer",
			Trace:       trace,
			AttemptedAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		})).To(Succeed())

		rep, err := artwork.Explain(ctx, ds, nil, model.KindArtistArtwork, "ar-1", artwork.ExplainOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Name).To(Equal("Radiohead"))
		Expect(rep.Walked).To(BeFalse())
		Expect(rep.Steps).To(Equal([]artwork.TraceStep{
			{Candidate: "external:deezer", Outcome: artwork.OutcomeHit, Detail: "https://cdn/x.jpg"},
		}))
		Expect(rep.Source).To(Equal("external:deezer"))
		Expect(rep.Result()).To(Equal("resolved from external:deezer"))
		Expect(rep.ChainOrigin()).To(ContainSubstring("recorded"))
	})

	It("renders a zero attempted-at the same way the CLI's formatTime does", func() {
		rep := artwork.ExplainReport{Stored: &model.ItemArtwork{Source: "folder"}}
		Expect(rep.ChainOrigin()).To(Equal("recorded -"))
	})

	It("reports nothing recorded when there is no stored state", func() {
		rep, err := artwork.Explain(ctx, ds, nil, model.KindArtistArtwork, "ar-1", artwork.ExplainOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Stored).To(BeNil())
		Expect(rep.Steps).To(BeEmpty())
		Expect(rep.ChainOrigin()).To(Equal("not recorded"))
	})

	It("includes the queue row and its failure trace", func() {
		failure := `[{"c":"external:lastfm","o":"error","d":"429"}]`
		Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
			ItemKind: model.KindArtistArtwork.Prefix(), ItemID: "ar-1", ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityScan,
		})).To(Succeed())
		queueRepo.SetTrace(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary, failure)

		rep, err := artwork.Explain(ctx, ds, nil, model.KindArtistArtwork, "ar-1", artwork.ExplainOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Queued).ToNot(BeNil())
		Expect(rep.Queued.Priority).To(Equal(model.ArtworkPriorityScan))
		Expect(rep.LastAttemptFailed()).To(Equal([]artwork.TraceStep{
			{Candidate: "external:lastfm", Outcome: artwork.OutcomeError, Detail: "429"},
		}))
	})

	It("records nothing for a kind that keeps no state and has no walker", func() {
		Expect(ds.Album(ctx).Put(&model.Album{ID: "al-1", Name: "OK Computer"})).To(Succeed())

		rep, err := artwork.Explain(ctx, ds, nil, model.KindDiscArtwork, "al-1:2", artwork.ExplainOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Stored).To(BeNil())
		Expect(rep.Steps).To(BeEmpty())
	})

	It("performs a fresh walk and reports its outcome, including a failed one, when a walker is supplied", func() {
		// GetAll erroring mid-walk is a deterministic way to force ResolveErr without a real library.
		ds.Album(ctx).(*tests.MockAlbumRepo).SetError(true)
		opts := artwork.ExplainOptions{Walk: func(t *artwork.ChainTrace) *artwork.TracingResolver {
			return artwork.NewTracingResolver(ds, nil, nil, t, false)
		}}

		rep, err := artwork.Explain(ctx, ds, nil, model.KindArtistArtwork, "ar-1", opts)
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Walked).To(BeTrue())
		Expect(rep.ResolveErr).To(HaveOccurred())
	})

	It("still reports Walked for a kind with no chain to walk, when a walker is supplied", func() {
		Expect(ds.Playlist(ctx).Put(&model.Playlist{ID: "pl-1", Name: "Favorites"})).To(Succeed())
		opts := artwork.ExplainOptions{Walk: func(*artwork.ChainTrace) *artwork.TracingResolver {
			panic("a kind that cannot walk must never be handed to the walker")
		}}

		rep, err := artwork.Explain(ctx, ds, nil, model.KindPlaylistArtwork, "pl-1", opts)
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.Walked).To(BeTrue())
	})
})
