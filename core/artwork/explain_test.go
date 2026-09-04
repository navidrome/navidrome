package artwork_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
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
