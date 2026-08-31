package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseArtworkKind", func() {
	It("accepts a supported kind", func() {
		k, err := parseArtworkKind("ar", artwork.RecheckKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(k).To(Equal(model.KindArtistArtwork))
	})

	It("rejects an unknown kind and lists the valid ones", func() {
		_, err := parseArtworkKind("zz", artwork.RecheckKinds)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ar"))
		Expect(err.Error()).To(ContainSubstring("al"))
	})

	It("rejects a known kind the command does not accept", func() {
		_, err := parseArtworkKind("mf", artwork.RecheckKinds)
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("accepts the kinds each command supports",
		func(prefix string, valid []model.Kind) {
			_, err := parseArtworkKind(prefix, valid)
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("explain reads disc artwork", "dc", explainKinds),
		Entry("explain reads media file artwork", "mf", explainKinds),
		// Disc artwork has no state to clear and the worker cannot resolve it, so refresh must not
		// accept it: the queue row would be rejected on every drain.
		Entry("refresh re-queues media files", "mf", artwork.RefreshableKinds),
	)

	It("rejects disc artwork for refresh", func() {
		_, err := parseArtworkKind("dc", artwork.RefreshableKinds)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("resolveArtworkTargets", func() {
	var ds *tests.MockDataStore
	ctx := context.Background()

	BeforeEach(func() {
		artists := tests.CreateMockArtistRepo()
		artists.SetData(model.Artists{{ID: "artist1"}})
		ds = &tests.MockDataStore{MockedArtist: artists}
	})

	It("accepts the explicit <kind> <id> leader shared by every id", func() {
		targets, failures, err := resolveArtworkTargets(ctx, ds, []string{"al", "x", "y"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(failures).To(BeEmpty())
		Expect(targets).To(Equal([]model.ArtworkID{
			{Kind: model.KindAlbumArtwork, ID: "x"}, {Kind: model.KindAlbumArtwork, ID: "y"}}))
	})

	It("rejects an explicit kind the command does not accept as a usage error", func() {
		_, _, err := resolveArtworkTargets(ctx, ds, []string{"dc", "x"}, artwork.RefreshableKinds)
		Expect(err).To(MatchError(ContainSubstring("invalid kind")))
	})

	It("resolves a bare id by looking it up across tables", func() {
		targets, failures, err := resolveArtworkTargets(ctx, ds, []string{"artist1"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(failures).To(BeEmpty())
		Expect(targets).To(Equal([]model.ArtworkID{{Kind: model.KindArtistArtwork, ID: "artist1"}}))
	})

	It("reads the kind from a full artwork id prefix without a database lookup", func() {
		targets, _, err := resolveArtworkTargets(ctx, ds, []string{"al-realalbum"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(Equal([]model.ArtworkID{{Kind: model.KindAlbumArtwork, ID: "realalbum"}}))
	})

	It("strips the hash suffix from a full artwork id", func() {
		targets, _, err := resolveArtworkTargets(ctx, ds, []string{"al-realalbum_0123456789abcdef"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(Equal([]model.ArtworkID{{Kind: model.KindAlbumArtwork, ID: "realalbum"}}))
	})

	It("collects a self-describing arg whose kind the command does not accept", func() {
		targets, failures, err := resolveArtworkTargets(ctx, ds, []string{"dc-realalbum:2"}, artwork.RefreshableKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(BeEmpty())
		Expect(failures).To(HaveLen(1))
		Expect(failures[0]).To(MatchError(ContainSubstring("invalid kind")))
	})

	It("collects an id that matches nothing and has no kind prefix", func() {
		targets, failures, err := resolveArtworkTargets(ctx, ds, []string{"nope"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(BeEmpty())
		Expect(failures).To(HaveLen(1))
		Expect(failures[0]).To(MatchError(ContainSubstring("could not determine kind")))
	})

	It("resolves the valid ids and collects the unresolvable ones", func() {
		targets, failures, err := resolveArtworkTargets(ctx, ds, []string{"artist1", "nope", "al-realalbum"}, explainKinds)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(Equal([]model.ArtworkID{
			{Kind: model.KindArtistArtwork, ID: "artist1"}, {Kind: model.KindAlbumArtwork, ID: "realalbum"}}))
		Expect(failures).To(HaveLen(1))
		Expect(failures[0]).To(MatchError(ContainSubstring("could not determine kind")))
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

	It("reports indeterminate when a local candidate exists but could not be read", func() {
		steps := []artwork.TraceStep{
			{Candidate: "cover.*", Outcome: "miss"},
			{Candidate: "embedded", Outcome: "unreadable"},
		}
		Expect(explainResult("", steps)).To(ContainSubstring("indeterminate"),
			"the worker retries an unreadable candidate instead of settling absent, so this is not a clean miss")
	})

	It("reports indeterminate when a processing stage errored after a candidate was found", func() {
		steps := []artwork.TraceStep{
			{Candidate: "cover.*", Outcome: "hit", Detail: "/music/cover.jpg"},
			{Candidate: "store", Outcome: "error", Detail: "disk full"},
		}
		Expect(explainResult("", steps)).To(ContainSubstring("indeterminate"),
			"a stage error is a processing failure the worker retries, not a definitive miss")
	})

	It("does not qualify a hit that an earlier unreadable candidate preceded", func() {
		// chainState.try stamps only the external error onto a hit and drops the local one, so the
		// worker settles this as found; warning about it would be a false alarm.
		steps := []artwork.TraceStep{
			{Candidate: "embedded", Outcome: "unreadable"},
			{Candidate: "cover.*", Outcome: "hit", Detail: "/music/cover.jpg"},
		}
		Expect(explainResult("folder", steps)).To(Equal("resolved from folder"))
	})

	It("reports indeterminate when an external lookup failed transiently", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "miss"},
			{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
		}
		Expect(explainResult("", steps)).To(ContainSubstring("indeterminate"),
			"a failed network call is not evidence that the item has no artwork")
	})

	It("qualifies a win a failed higher-priority external lookup could have taken", func() {
		steps := []artwork.TraceStep{
			{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
			{Candidate: "artist.*", Outcome: "hit", Detail: "/music/artist.jpg"},
		}
		res := explainResult("artist.*", steps)
		Expect(res).To(ContainSubstring("resolved from artist.*"))
		Expect(res).To(ContainSubstring("indeterminate"),
			"the resolver serves this hit but retries later, so the winner is provisional")
	})

	It("does not qualify an external win that followed a failed external lookup", func() {
		steps := []artwork.TraceStep{
			{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
			{Candidate: "external:lastfm", Outcome: "hit", Detail: "http://img"},
		}
		Expect(explainResult("external:lastfm", steps)).To(Equal("resolved from external:lastfm"),
			"a later agent supplying the image discards the earlier error, so there is no retry to warn about")
	})

	It("does not qualify a win that outranked the failed external lookup", func() {
		steps := []artwork.TraceStep{
			{Candidate: "artist.*", Outcome: "hit"},
			{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
		}
		Expect(explainResult("artist.*", steps)).To(Equal("resolved from artist.*"))
	})
})

var _ = Describe("explainAgents", func() {
	It("accounts for every configured agent, marking the ones the CLI could not use", func() {
		out := explainAgents("artist-nfo-metadata,apple-music,deezer,lastfm", []string{"deezer"})
		for _, name := range []string{"artist-nfo-metadata", "apple-music", "deezer", "lastfm"} {
			Expect(out).To(ContainSubstring(name),
				"a configured agent missing from this line reads as if it had never been configured")
		}
		Expect(out).To(ContainSubstring("not available to the CLI"))
	})

	It("does not mark anything when every configured agent is available", func() {
		out := explainAgents("deezer, lastfm", []string{"lastfm", "deezer"})
		Expect(out).To(Equal("deezer, lastfm"))
	})

	It("reports an empty configuration as none, not as an unavailable agent", func() {
		Expect(explainAgents("", nil)).To(Equal("(none)"))
	})
})

var _ = Describe("formatExplain", func() {
	var rep explainReport

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.ArtistArtPriority = "external, artist.*"
		rep = explainReport{
			kind:   model.KindArtistArtwork,
			id:     "ar-1",
			name:   "Radiohead",
			agents: "lastfm,spotify",
			walked: true,
			steps: []artwork.TraceStep{
				{Candidate: "upload", Outcome: "skipped", Detail: "no uploaded image"},
				{Candidate: "external:deezer", Outcome: "error", Detail: "context deadline exceeded"},
			},
			source: "",
		}
	})

	It("reports the item, its config and the chain it walked", func() {
		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("Radiohead"))
		Expect(out).To(ContainSubstring("ar-1"))
		Expect(out).To(ContainSubstring("ArtistArtPriority"))
		Expect(out).To(ContainSubstring("lastfm,spotify"))
		Expect(out).To(ContainSubstring("external:deezer"))
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
		Expect(out).To(ContainSubstring("scan (50)"), "a bare 50 makes the operator look the priority up")
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
		Expect(out).To(ContainSubstring("external:deezer"), "the steps taken before the failure still print")
	})

	It("says a kind that does not walk a chain has no chain, without an empty table", func() {
		conf.Server.CoverArtPriority = "cover.*, embedded"
		rep.kind = model.KindPlaylistArtwork
		rep.steps = nil
		rep.agents = ""

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("does not walk a priority chain"))
		Expect(out).ToNot(ContainSubstring("CANDIDATE"),
			"an empty chain table reads as 'nothing was tried', which is false")
		Expect(out).ToNot(ContainSubstring("not resolved"),
			"nothing was resolved because nothing was attempted")
		Expect(out).ToNot(ContainSubstring("CoverArtPriority"),
			"the priority chain config does not govern this kind")
	})

	It("says disc artwork keeps no state instead of reporting it as unresolved state", func() {
		conf.Server.DiscArtPriority = "cover.jpg, embedded"
		rep = explainReport{
			kind: model.KindDiscArtwork, id: "al-1:2", name: "OK Computer (disc 2)",
			steps:  []artwork.TraceStep{{Candidate: "cover.jpg", Outcome: "hit", Detail: "/music/cover.jpg"}},
			source: "folder",
			walked: true,
		}

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("never recorded"))
		Expect(out).To(ContainSubstring("never queued"))
		Expect(out).ToNot(ContainSubstring("no artwork state recorded"),
			"a missing row would read as a lookup that failed, when disc artwork has no row by design")
		Expect(out).To(ContainSubstring("DiscArtPriority"))
		Expect(out).ToNot(ContainSubstring("Agents:"), "disc artwork never asks an agent")
		Expect(out).To(ContainSubstring("resolved from folder"))
	})

	Context("stored traces", func() {
		BeforeEach(func() {
			rep.walked = false
			rep.steps = nil
		})

		It("labels a recorded chain with when it was recorded, not as a walk done now", func() {
			attempted := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
			rep.stored = &model.ItemArtwork{Source: "folder", Hash: "abc", AttemptedAt: attempted}
			rep.steps = []artwork.TraceStep{{Candidate: "artist.*", Outcome: "hit", Detail: "/music/artist.jpg"}}
			rep.source = "folder"

			out := formatExplain(rep)
			Expect(out).To(ContainSubstring("Chain (recorded 2026-08-13T10:00:00Z)"))
			Expect(out).To(ContainSubstring("/music/artist.jpg"))
			Expect(out).To(ContainSubstring("resolved from folder"))
		})

		It("says so when the item has never been resolved", func() {
			out := formatExplain(rep)
			Expect(out).To(ContainSubstring("no resolution recorded yet"))
			Expect(out).To(ContainSubstring("--live"))
			Expect(out).ToNot(ContainSubstring("not resolved"),
				"nothing was recorded, which is not the same as resolving to nothing")
		})

		It("distinguishes a row written before traces existed from one with an empty chain", func() {
			rep.stored = &model.ItemArtwork{Source: "folder", Hash: "abc", AttemptedAt: time.Now()}

			Expect(formatExplain(rep)).To(ContainSubstring("resolved before traces were recorded"))
		})

		It("does not call an absent row with an empty recorded chain a pre-tracing row", func() {
			// An empty priority list records a real but empty chain and resolves absent; that is not a
			// legacy row, so it must not be reported as resolved before tracing existed.
			rep.stored = &model.ItemArtwork{Source: "", Hash: "", AttemptedAt: time.Now()}

			out := formatExplain(rep)
			Expect(out).ToNot(ContainSubstring("resolved before traces were recorded"))
			Expect(out).To(ContainSubstring("no candidates were recorded"))
			Expect(out).To(ContainSubstring("not resolved"), "the Result still reports the absence plainly")
		})

		It("prints why the last attempt failed and why it gave up", func() {
			rep.queued = &model.ArtworkQueueItem{Priority: model.ArtworkPriorityScan, Attempts: 3,
				Trace: `[{"c":"decode","o":"error","d":"bad header"}]`}
			rep.stored = &model.ItemArtwork{Source: "folder", Hash: "abc", AttemptedAt: time.Now(),
				LastFailure: `[{"c":"read","o":"error","d":"i/o timeout"}]`}

			out := formatExplain(rep)
			Expect(out).To(ContainSubstring("Last attempt failed"))
			Expect(out).To(ContainSubstring("bad header"))
			Expect(out).To(ContainSubstring("Gave up after"))
			Expect(out).To(ContainSubstring("i/o timeout"))
		})

		It("omits the failure tables when there is no failure to report", func() {
			out := formatExplain(rep)
			Expect(out).ToNot(ContainSubstring("Last attempt failed"))
			Expect(out).ToNot(ContainSubstring("Gave up after"))
		})
	})

	It("reports the setting that governs media file artwork", func() {
		conf.Server.EnableMediaFileCoverArt = false
		rep = explainReport{
			kind: model.KindMediaFileArtwork, id: "mf-1", name: "Airbag",
			walked: true,
			steps: []artwork.TraceStep{
				{Candidate: "embedded", Outcome: "skipped", Detail: "EnableMediaFileCoverArt is off"},
			},
		}

		out := formatExplain(rep)
		Expect(out).To(ContainSubstring("EnableMediaFileCoverArt"))
		Expect(out).To(ContainSubstring("false"))
		Expect(out).To(ContainSubstring("not resolved"))
		Expect(out).To(ContainSubstring("no artwork state recorded"), "media files do keep state")
	})
})

var _ = Describe("explainConfig", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DiscArtPriority = "cover.jpg"
		conf.Server.EnableMediaFileCoverArt = true
	})

	DescribeTable("names the setting that decides where a kind's artwork comes from",
		func(kind model.Kind, setting, value string) {
			gotSetting, gotValue := explainConfig(kind)
			Expect(gotSetting).To(Equal(setting))
			Expect(gotValue).To(Equal(value))
		},
		Entry("disc", model.KindDiscArtwork, "DiscArtPriority", "cover.jpg"),
		Entry("media file", model.KindMediaFileArtwork, "EnableMediaFileCoverArt", "true"),
		Entry("playlist has none", model.KindPlaylistArtwork, "", ""),
	)
})

var _ = Describe("artwork refresh command", func() {
	It("requires at least one argument", func() {
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{})).To(HaveOccurred())
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{"id1"})).ToNot(HaveOccurred())
		Expect(artworkRefreshCmd.Args(artworkRefreshCmd, []string{"ar", "id1"})).ToNot(HaveOccurred())
	})
})

var _ = Describe("artwork reprocess selection", func() {
	It("errors when no selector is given", func() {
		_, err := selectedKinds(nil, nil, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--all"))
	})

	It("returns every kind for --all", func() {
		ks, err := selectedKinds(nil, nil, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(ConsistOf(artwork.RecheckKinds))
	})

	It("returns every kind for a source filter given without a kind", func() {
		ks, err := selectedKinds(nil, []string{"folder"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(ConsistOf(artwork.RecheckKinds), "--source alone is already a complete selection")
	})

	It("returns only the named kinds", func() {
		ks, err := selectedKinds([]string{"ar"}, nil, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(Equal([]model.Kind{model.KindArtistArtwork}))
	})

	It("keeps a named kind selection alongside a source filter", func() {
		ks, err := selectedKinds([]string{"ar"}, []string{"folder"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(Equal([]model.Kind{model.KindArtistArtwork}))
	})

	It("rejects an unknown kind", func() {
		_, err := selectedKinds([]string{"zz"}, nil, false)
		Expect(err).To(HaveOccurred())
	})

	It("counts a repeated kind once", func() {
		ks, err := selectedKinds([]string{"ar", "ar"}, nil, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(ks).To(Equal([]model.Kind{model.KindArtistArtwork}))
	})
})

var _ = Describe("explain/reprocess source round trip", func() {
	ctx := context.Background()

	// storedSource reads back the Source line explain printed, as an operator would copy it.
	storedSource := func(out string) string {
		GinkgoHelper()
		for line := range strings.SplitSeq(out, "\n") {
			if after, ok := strings.CutPrefix(strings.TrimSpace(line), "Source:"); ok {
				return strings.TrimSpace(after)
			}
		}
		Fail("explain printed no Source line")
		return ""
	}

	It("names the absent state as reprocess --source accepts it", func() {
		ds := &tests.MockDataStore{}
		art := ds.Artwork(ctx).(*tests.MockArtworkRepo)
		Expect(art.PutItemArtwork(&model.ItemArtwork{ItemKind: model.KindArtistArtwork.Prefix(),
			ItemID: "ar-1", ImageType: model.ImageTypePrimary})).To(Succeed())

		shown := storedSource(formatExplain(explainReport{kind: model.KindArtistArtwork, id: "ar-1",
			stored: &model.ItemArtwork{AttemptedAt: time.Now()}}))

		q := ds.ArtworkQueue(ctx)
		Expect(validateSources(q, repositorySources([]string{shown}))).To(Succeed(),
			"explain's spelling of a source must be pasteable into --source")
		Expect(validateSources(q, repositorySources([]string{"(" + shown + ")"}))).ToNot(Succeed(),
			"a parenthesised name would be rejected, so explain must not print one")
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
		Expect(promptConfirm(strings.NewReader("y\n"), "re-resolve")(&out, 42, 7)).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("re-resolve 42 items"))
		Expect(out.String()).To(ContainSubstring("External lookups: ~7 estimated"))
	})

	It("defaults to no on anything else", func() {
		Expect(promptConfirm(strings.NewReader("\n"), "re-resolve")(&out, 1, 1)).To(BeFalse())
		Expect(promptConfirm(strings.NewReader("nope\n"), "re-resolve")(&out, 1, 1)).To(BeFalse())
		Expect(promptConfirm(strings.NewReader(""), "re-resolve")(&out, 1, 1)).To(BeFalse())
	})

	It("drops the external clause when no lookup will be made", func() {
		Expect(promptConfirm(strings.NewReader("y\n"), "cancel")(&out, 3, 0)).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("cancel 3 items."))
		Expect(out.String()).ToNot(ContainSubstring("External lookups"))
	})
})

var _ = Describe("confirmUnlessYes", func() {
	var out strings.Builder

	BeforeEach(func() { out.Reset() })

	It("prompts when --yes was not given", func() {
		Expect(confirmUnlessYes(false, strings.NewReader("n\n"), "re-resolve")(&out, 5, 5)).To(BeFalse())
		Expect(out.String()).To(ContainSubstring("Continue?"))
	})

	It("bypasses the prompt only for --yes", func() {
		Expect(confirmUnlessYes(true, strings.NewReader(""), "re-resolve")(&out, 5, 5)).To(BeTrue())
		Expect(out.String()).To(BeEmpty(), "--yes must not print a prompt it never reads")
	})
})

var _ = Describe("reprocessArtwork", func() {
	var ds *tests.MockDataStore
	var art *tests.MockArtworkRepo
	var queue *tests.MockArtworkQueueRepo
	var out strings.Builder
	var imageAgents artwork.ImageAgentCount
	ctx := context.Background()
	kinds := []model.Kind{model.KindArtistArtwork, model.KindAlbumArtwork}
	accept := func(io.Writer, int64, int64) bool { return true }
	decline := func(io.Writer, int64, int64) bool { return false }

	put := func(kind model.Kind, id, source string) {
		Expect(art.PutItemArtwork(&model.ItemArtwork{ItemKind: kind.Prefix(), ItemID: id,
			ImageType: model.ImageTypePrimary, Hash: "h" + id, Source: source})).To(Succeed())
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CoverArtPriority = "cover.*, external"
		conf.Server.ArtistArtPriority = "artist.*, external"
		conf.Server.EnableM3UExternalAlbumArt = false
		imageAgents = artwork.ImageAgentCount{Artist: 1, Album: 1}
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
		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, imageAgents, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("external:deezer"))
		Expect(out.String()).To(ContainSubstring("artist"))
		Expect(out.String()).To(ContainSubstring("album"))
		Expect(out.String()).To(ContainSubstring("TOTAL"))
		Expect(out.String()).To(ContainSubstring("Dry run"))
		Expect(queue.Count()).To(BeZero())
	})

	It("queues nothing when the operator declines", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, nil, imageAgents, false, decline, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Aborted"))
		Expect(queue.Count()).To(BeZero())
	})

	It("queues the matching items at recheck priority, leaving their artwork state alone", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, imageAgents, false, accept, &out)).To(Succeed())

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
		Expect(reprocessArtwork(ctx, ds, kinds, []string{""}, imageAgents, false, accept, &out)).To(Succeed())

		Expect(queue.Count()).To(Equal(int64(1)))
		_, err := queue.Get(model.KindArtistArtwork, "ar-2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
	})

	It("reports matched and queued separately when part of the set is already queued", func() {
		Expect(queue.Enqueue(model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar-1",
			ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBump})).To(Succeed())

		Expect(reprocessArtwork(ctx, ds, kinds, []string{"external:deezer"}, imageAgents, false, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Queued 1 of 2 matched items"))
		Expect(out.String()).To(ContainSubstring("Already queued, left unchanged: 1"))
		queued, err := queue.Get(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(queued.Priority).To(Equal(model.ArtworkPriorityBump),
			"an already-queued row keeps its priority and backoff")
	})

	It("stops at a selection that matches nothing instead of prompting", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindRadioArtwork}, nil, imageAgents, false,
			func(io.Writer, int64, int64) bool {
				Fail("must not prompt when there is nothing to queue")
				return true
			}, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing"))
		Expect(queue.Count()).To(BeZero())
	})

	It("reports an empty selection as a dry run when one was asked for", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindRadioArtwork}, nil, imageAgents, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing matches"))
		Expect(out.String()).To(ContainSubstring("Dry run"))
	})

	It("shows the external estimate on a dry run, which never reaches the prompt", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindAlbumArtwork}, nil, imageAgents, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("External lookups: ~2 estimated"))
	})

	It("bills every agent per item, not one lookup per item", func() {
		imageAgents = artwork.ImageAgentCount{Artist: 2, Album: 3}
		var external int64
		capture := func(_ io.Writer, _, e int64) bool { external = e; return false }

		Expect(reprocessArtwork(ctx, ds, kinds, nil, imageAgents, false, capture, &out)).To(Succeed())

		Expect(external).To(Equal(int64(2*2+2*3)), "2 artists at 2 agents plus 2 albums at 3 agents")
		Expect(out.String()).To(ContainSubstring("External lookups: ~10 estimated"))
	})

	It("names the estimate's blind spots instead of claiming a bound it cannot hold", func() {
		Expect(reprocessArtwork(ctx, ds, kinds, nil, imageAgents, true, accept, &out)).To(Succeed())

		// The count includes plugin agents once they load, so the caveat is about a failed load,
		// not about plugins being invisible to the CLI.
		Expect(out.String()).To(ContainSubstring("plugin agents counted only when they load"))
		Expect(out.String()).To(ContainSubstring("local hits may need fewer"))
		Expect(out.String()).ToNot(ContainSubstring("up to"), "plugin agents make any ceiling false")
		Expect(out.String()).ToNot(ContainSubstring("at least"), "a local hit makes any floor false")
	})

	It("says so when the selection needs no external lookup", func() {
		conf.Server.CoverArtPriority = "cover.*"
		put(model.KindPlaylistArtwork, "pl-1", "playlist")

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindPlaylistArtwork}, nil, imageAgents, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("External lookups: none"))
	})

	It("counts playlists as external cost when the m3u image fetch is enabled", func() {
		conf.Server.CoverArtPriority = "cover.*"
		conf.Server.EnableM3UExternalAlbumArt = true
		put(model.KindPlaylistArtwork, "pl-1", "playlist")
		var external int64
		capture := func(_ io.Writer, _, e int64) bool { external = e; return false }

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindPlaylistArtwork}, nil, imageAgents, false, capture, &out)).To(Succeed())

		Expect(external).To(Equal(int64(1)))
		Expect(out.String()).To(ContainSubstring("External lookups: ~1 estimated"))
	})

	It("bills a playlist for every album its grid samples, at every agent", func() {
		imageAgents = artwork.ImageAgentCount{Album: 3}
		put(model.KindPlaylistArtwork, "pl-1", "playlist")
		var external int64
		capture := func(_ io.Writer, _, e int64) bool { external = e; return false }

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindPlaylistArtwork}, nil, imageAgents, false, capture, &out)).To(Succeed())

		Expect(external).To(Equal(int64(artwork.PlaylistGridSamples*3)),
			"one playlist samples 4 albums, each walking all 3 album agents")
	})

	It("counts only the kinds that call an external agent as external cost", func() {
		put(model.KindRadioArtwork, "ra-1", "upload")
		var total, external int64
		capture := func(_ io.Writer, t, e int64) bool { total, external = t, e; return false }

		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindAlbumArtwork, model.KindRadioArtwork},
			nil, imageAgents, false, capture, &out)).To(Succeed())

		Expect(total).To(Equal(int64(3)))
		Expect(external).To(Equal(int64(2)), "radio artwork never reaches an external agent")
	})

	It("rejects an unknown source and names the ones in use", func() {
		err := reprocessArtwork(ctx, ds, kinds, []string{"externa:deezer"}, imageAgents, true, accept, &out)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("externa:deezer"))
		Expect(err.Error()).To(ContainSubstring("external:deezer"))
		Expect(err.Error()).To(ContainSubstring("folder"))
		Expect(err.Error()).To(ContainSubstring("absent"), "the empty source prints under its user-facing name")
		Expect(queue.Count()).To(BeZero())
	})

	It("accepts the absent filter with nothing absent, still rejecting a typo", func() {
		put(model.KindArtistArtwork, "ar-2", "folder")

		Expect(reprocessArtwork(ctx, ds, kinds, repositorySources([]string{absentSource}),
			imageAgents, false, accept, &out)).To(Succeed(),
			"a reserved source must stay valid once the library has none of it")
		Expect(out.String()).To(ContainSubstring("Nothing matches"))
		Expect(queue.Count()).To(BeZero())

		Expect(reprocessArtwork(ctx, ds, kinds, repositorySources([]string{"absnt"}),
			imageAgents, true, accept, &out)).ToNot(Succeed(), "a typo must still be rejected")
	})

	It("accepts a source another kind uses, letting the empty selection report itself", func() {
		Expect(reprocessArtwork(ctx, ds, []model.Kind{model.KindArtistArtwork}, []string{"folder"},
			imageAgents, false, decline, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing matches"),
			"a well-formed filter must not be reported as a typo because of the kinds selected")
		Expect(queue.Count()).To(BeZero())
	})
})

var _ = Describe("artwork status command", func() {
	It("takes no arguments", func() {
		Expect(artworkStatusCmd.Args(artworkStatusCmd, []string{})).ToNot(HaveOccurred())
		Expect(artworkStatusCmd.Args(artworkStatusCmd, []string{"x"})).To(HaveOccurred())
	})
})

var _ = Describe("collectStatus", func() {
	var ds *tests.MockDataStore
	var art *tests.MockArtworkRepo
	var queue *tests.MockArtworkQueueRepo
	ctx := context.Background()

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		art = ds.Artwork(ctx).(*tests.MockArtworkRepo)
		queue = ds.ArtworkQueue(ctx).(*tests.MockArtworkQueueRepo)
		put := func(kind model.Kind, id, source, hash string, attempted time.Time) {
			Expect(art.PutItemArtwork(&model.ItemArtwork{ItemKind: kind.Prefix(), ItemID: id,
				ImageType: model.ImageTypePrimary, Source: source, Hash: hash, AttemptedAt: attempted})).To(Succeed())
		}
		put(model.KindArtistArtwork, "ar-1", "external:deezer", "h1", time.Now())
		put(model.KindArtistArtwork, "ar-2", "", "", time.Now().Add(-artwork.StaleAbsentAge-time.Hour))
		put(model.KindArtistArtwork, "ar-3", "", "", time.Now())
		put(model.KindAlbumArtwork, "al-1", "folder", "h2", time.Now())
		Expect(queue.Enqueue(model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar-9",
			ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBackfill})).To(Succeed())
	})

	It("reports the queue, the source distribution and the absent ages", func() {
		rep, err := collectStatus(ctx, ds)
		Expect(err).ToNot(HaveOccurred())

		Expect(rep.queue).To(ConsistOf(model.ArtworkQueueStat{ItemKind: "ar",
			Priority: model.ArtworkPriorityBackfill, Count: 1}))
		Expect(rep.sources).To(ContainElements(
			sourceCount{kind: model.KindArtistArtwork, source: "external:deezer", count: 1},
			sourceCount{kind: model.KindArtistArtwork, source: "", count: 2},
			sourceCount{kind: model.KindAlbumArtwork, source: "folder", count: 1},
		))
		Expect(rep.absent).To(ContainElement(absentCount{kind: model.KindArtistArtwork,
			ArtworkAbsentStat: model.ArtworkAbsentStat{Total: 2, Stale: 1}}))
	})

	It("compares the stored fingerprint against the current one", func() {
		Expect(ds.Property(ctx).Put(consts.ArtConfFingerprintPropertyKey, "old-fingerprint")).To(Succeed())

		rep, err := collectStatus(ctx, ds)
		Expect(err).ToNot(HaveOccurred())
		Expect(rep.stored).To(Equal("old-fingerprint"))
		Expect(rep.current).To(Equal(artwork.ConfigFingerprint()),
			"the CLI must report the value backfill itself compares")
	})

	It("queues nothing", func() {
		_, err := collectStatus(ctx, ds)
		Expect(err).ToNot(HaveOccurred())
		Expect(queue.Count()).To(Equal(int64(1)), "status must not enqueue anything")
	})
})

var _ = Describe("formatStatus", func() {
	var rep statusReport

	BeforeEach(func() {
		rep = statusReport{
			queue: []model.ArtworkQueueStat{
				{ItemKind: "ar", Priority: model.ArtworkPriorityBackfill, Count: 2},
				{ItemKind: "al", Priority: model.ArtworkPriorityScan, Count: 1},
			},
			sources: []sourceCount{
				{kind: model.KindArtistArtwork, source: "external:deezer", count: 5},
				{kind: model.KindArtistArtwork, source: "", count: 2},
			},
			absent: []absentCount{
				{kind: model.KindArtistArtwork, ArtworkAbsentStat: model.ArtworkAbsentStat{Total: 2, Stale: 1}},
			},
			inputs:  []artwork.FingerprintInput{{Name: "Agents", Value: "deezer,lastfm"}},
			stored:  "abc123",
			current: "abc123",
		}
	})

	// block isolates one section, so an assertion cannot be satisfied by a coincidence elsewhere.
	block := func(out, header string) string {
		GinkgoHelper()
		_, after, found := strings.Cut(out, header+"\n")
		Expect(found).To(BeTrue(), "the %q block must be printed", header)
		body, _, _ := strings.Cut(after, "\n\n")
		return body
	}

	It("names the kind and the priority of every queued row", func() {
		queue := block(formatStatus(rep), "Queue")
		Expect(queue).To(MatchRegexp(`artist\s+backfill\s+2`))
		Expect(queue).To(MatchRegexp(`album\s+scan\s+1`))
	})

	It("totals the queue", func() {
		Expect(block(formatStatus(rep), "Queue")).To(MatchRegexp(`TOTAL\s+3`))
	})

	It("counts each source, naming the empty one absent", func() {
		sources := block(formatStatus(rep), "Sources")
		Expect(sources).To(MatchRegexp(`artist\s+external:deezer\s+5`))
		Expect(sources).To(MatchRegexp(`artist\s+absent\s+2`))
	})

	It("prints the absent total and how many are due for recheck", func() {
		absent := block(formatStatus(rep), "Absent (resolved, no image found)")
		Expect(absent).To(MatchRegexp(`artist\s+2\s+1`))
	})

	It("states the recheck window and the drip rate the absent counts are bucketed against", func() {
		Expect(formatStatus(rep)).To(ContainSubstring(fmt.Sprintf("%gh", artwork.StaleAbsentAge.Hours())))
		Expect(formatStatus(rep)).To(ContainSubstring("100 per kind per hour"))
	})

	It("leads with the queued backlog, which is the finding, not with the fingerprint verdict", func() {
		out := block(formatStatus(rep), "Backfill")
		Expect(out).To(MatchRegexp(`State:\s+backfill running: 2 items queued`),
			"an operator scanning for trouble must not read 'up to date' while 2 items churn")
		Expect(out).To(ContainSubstring("fingerprint up to date"))
	})

	It("keeps the re-enqueue warning while a backfill is already running", func() {
		rep.stored = "older"

		out := block(formatStatus(rep), "Backfill")
		Expect(out).To(MatchRegexp(`State:\s+backfill running: 2 items queued`))
		Expect(out).To(ContainSubstring("re-enqueued"),
			"the stored fingerprint is still stale, so a second full re-enqueue is pending on top of this one")
	})

	It("reports up to date only once the backfill has drained", func() {
		rep.queue = []model.ArtworkQueueStat{{ItemKind: "al", Priority: model.ArtworkPriorityScan, Count: 1}}

		Expect(block(formatStatus(rep), "Backfill")).To(MatchRegexp(`State:\s+up to date`))
	})

	It("echoes the config inputs a fingerprint change would have come from", func() {
		out := block(formatStatus(rep), "Backfill")
		Expect(out).To(MatchRegexp(`Agents:\s+deezer,lastfm`))
		Expect(out).To(ContainSubstring("abc123"), "the fingerprint values themselves must be printed")
	})

	It("reports a changed fingerprint as a pending re-resolve of everything", func() {
		rep.stored = "older"
		rep.queue = nil

		out := formatStatus(rep)
		Expect(out).To(ContainSubstring("fingerprint changed"))
		Expect(out).ToNot(ContainSubstring("up to date"))
	})

	It("reports a never-recorded fingerprint without printing an empty value", func() {
		rep.stored = ""

		out := formatStatus(rep)
		Expect(out).To(ContainSubstring("(none)"))
		Expect(out).To(ContainSubstring("fingerprint changed"))
	})

	It("says the queue is empty instead of printing a headless table", func() {
		rep.queue = nil

		out := formatStatus(rep)
		Expect(out).To(ContainSubstring("empty"))
		Expect(out).ToNot(ContainSubstring("PRIORITY"))
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

		Expect(refreshItems(ctx, ds, []model.ArtworkID{
			{Kind: model.KindAlbumArtwork, ID: "al-1"}, {Kind: model.KindAlbumArtwork, ID: "al-3"}}, &out)).To(BeZero())

		_, err := art.GetItemArtwork(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		queued, err := queue.Get(model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(queued.Priority).To(Equal(model.ArtworkPriorityBump))
		Expect(out.String()).To(Equal("al/al-1: queued\nal/al-3: queued\n"))
	})

	It("skips an id that does not exist instead of queuing it", func() {
		Expect(refreshItems(ctx, ds, []model.ArtworkID{{Kind: model.KindAlbumArtwork, ID: "al-2"}}, &out)).To(Equal(1))

		_, err := queue.Get(model.KindAlbumArtwork, "al-2", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "a typo must not leave an orphan queue row")
		Expect(out.String()).To(BeEmpty())
	})

	It("continues past a failing id and counts the failures", func() {
		Expect(refreshItems(ctx, ds, []model.ArtworkID{{Kind: model.KindAlbumArtwork, ID: "al-1"},
			{Kind: model.KindAlbumArtwork, ID: "al-2"}, {Kind: model.KindAlbumArtwork, ID: "al-3"}}, &out)).To(Equal(1))

		Expect(out.String()).To(Equal("al/al-1: queued\nal/al-3: queued\n"),
			"the ids after a failure are still refreshed")
	})
})

var _ = Describe("needsImageAgents", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CoverArtPriority = "cover.*, external"
		conf.Server.ArtistArtPriority = "artist.*, external"
		conf.Server.EnableM3UExternalAlbumArt = false
	})

	It("is false for a selection no agent can serve", func() {
		Expect(needsImageAgents([]model.Kind{model.KindRadioArtwork})).To(BeFalse())
	})

	// The generated playlist grid resolves album art through the image agents, so a playlist
	// selection needs the count even though no agent is asked for a playlist image directly.
	It("is true for playlists, whose grid tiles resolve through the album chain", func() {
		Expect(needsImageAgents([]model.Kind{model.KindPlaylistArtwork})).To(BeTrue())
	})

	It("is true when any one of several kinds can reach an agent", func() {
		Expect(needsImageAgents([]model.Kind{model.KindRadioArtwork, model.KindAlbumArtwork})).To(BeTrue())
	})

	It("is false once the chains no longer reach an agent", func() {
		conf.Server.CoverArtPriority = "cover.*"
		conf.Server.ArtistArtPriority = "artist.*"
		Expect(needsImageAgents(artwork.RecheckKinds)).To(BeFalse())
	})
})

var _ = Describe("configuredAgents", func() {
	BeforeEach(func() { DeferCleanup(configtest.SetupConfig()) })

	It("splits and trims the configured list", func() {
		conf.Server.Agents = "lastfm, spotify ,deezer"
		Expect(configuredAgents()).To(Equal([]string{"lastfm", "spotify", "deezer"}))
	})

	// An empty name would match no plugin, but it also must not make the list look non-empty:
	// LoadPlugins treats an empty list as "load nothing".
	It("drops empty entries rather than passing a name nothing can match", func() {
		conf.Server.Agents = " , ,"
		Expect(configuredAgents()).To(BeEmpty())
	})
})

var _ = Describe("parseArtworkPriority", func() {
	It("accepts every name status prints", func() {
		for _, p := range []int{model.ArtworkPriorityRecheck, model.ArtworkPriorityBackfill,
			model.ArtworkPriorityScan, model.ArtworkPriorityBump} {
			Expect(parseArtworkPriority(priorityName(p))).To(Equal(p))
		}
	})

	It("rejects an unknown name and lists the valid ones", func() {
		_, err := parseArtworkPriority("urgent")
		Expect(err).To(MatchError(ContainSubstring(`invalid priority "urgent"`)))
		Expect(err).To(MatchError(ContainSubstring("backfill")))
	})

	// Accepting the raw numbers would make the help text a lie and let a typo like 11 select nothing.
	It("rejects the numeric form", func() {
		_, err := parseArtworkPriority("10")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("artwork cancel selection", func() {
	It("errors when no selector is given", func() {
		_, _, err := cancelSelection(nil, nil, false)
		Expect(err).To(MatchError(ContainSubstring("no selector given")))
	})

	// Empty, not an enumeration of the known kinds: --all must also take a queue row whose kind
	// this build does not recognise.
	It("selects with no filter at all for --all", func() {
		kinds, priorities, err := cancelSelection(nil, nil, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(kinds).To(BeEmpty())
		Expect(priorities).To(BeEmpty())
	})

	// The queue holds media file rows, so --all must reach them.
	It("accepts media file artwork, which reprocess does not", func() {
		kinds, _, err := cancelSelection([]string{"mf"}, nil, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(kinds).To(Equal([]model.Kind{model.KindMediaFileArtwork}))
	})

	It("treats a priority filter on its own as a complete selection", func() {
		kinds, priorities, err := cancelSelection(nil, []string{"backfill"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(kinds).To(BeEmpty(), "no kind filter means every kind")
		Expect(priorities).To(Equal([]int{model.ArtworkPriorityBackfill}))
	})

	It("returns only the named kinds and priorities", func() {
		kinds, priorities, err := cancelSelection([]string{"ar", "al"}, []string{"backfill", "scan"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(kinds).To(Equal([]model.Kind{model.KindArtistArtwork, model.KindAlbumArtwork}))
		Expect(priorities).To(Equal([]int{model.ArtworkPriorityBackfill, model.ArtworkPriorityScan}))
	})

	It("counts a repeated kind and a repeated priority once", func() {
		kinds, priorities, err := cancelSelection([]string{"ar", "ar"}, []string{"bump", "bump"}, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(kinds).To(HaveLen(1))
		Expect(priorities).To(HaveLen(1))
	})

	It("rejects an unknown kind", func() {
		_, _, err := cancelSelection([]string{"zz"}, nil, false)
		Expect(err).To(MatchError(ContainSubstring(`invalid kind "zz"`)))
	})

	It("rejects a kind that is never queued", func() {
		_, _, err := cancelSelection([]string{"dc"}, nil, false)
		Expect(err).To(MatchError(ContainSubstring("invalid kind")))
	})

	It("rejects an unknown priority", func() {
		_, _, err := cancelSelection(nil, []string{"urgent"}, false)
		Expect(err).To(MatchError(ContainSubstring("invalid priority")))
	})
})

var _ = Describe("cancelArtwork", func() {
	var ds *tests.MockDataStore
	var queue *tests.MockArtworkQueueRepo
	var out strings.Builder
	ctx := context.Background()
	accept := func(io.Writer, int64, int64) bool { return true }
	decline := func(io.Writer, int64, int64) bool { return false }

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		queue = ds.ArtworkQueue(ctx).(*tests.MockArtworkQueueRepo)
		out.Reset()
		Expect(queue.Enqueue(
			model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar-1", ImageType: model.ImageTypePrimary,
				Priority: model.ArtworkPriorityBackfill},
			model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar-2", ImageType: model.ImageTypePrimary,
				Priority: model.ArtworkPriorityBump},
			model.ArtworkQueueItem{ItemKind: "al", ItemID: "al-1", ImageType: model.ImageTypePrimary,
				Priority: model.ArtworkPriorityBackfill},
		)).To(Succeed())
	})

	It("previews the per-kind breakdown and cancels nothing on a dry run", func() {
		Expect(cancelArtwork(ctx, ds, []model.Kind{model.KindArtistArtwork}, nil, true, accept, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("artist"))
		Expect(out.String()).To(ContainSubstring("backfill"))
		Expect(out.String()).To(ContainSubstring("TOTAL"))
		Expect(out.String()).To(ContainSubstring("Dry run"))
		Expect(queue.Count()).To(BeNumerically("==", 3))
	})

	It("cancels nothing when the operator declines", func() {
		Expect(cancelArtwork(ctx, ds, nil, nil, false, decline, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Aborted"))
		Expect(queue.Count()).To(BeNumerically("==", 3))
	})

	It("deletes the selected rows and leaves the rest queued", func() {
		Expect(cancelArtwork(ctx, ds, nil, []int{model.ArtworkPriorityBackfill}, false, accept, &out)).To(Succeed())

		Expect(queue.Count()).To(BeNumerically("==", 1))
		_, err := queue.Get(model.KindArtistArtwork, "ar-2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred(), "a non-matching priority must stay queued")
		Expect(out.String()).To(ContainSubstring("Cancelled 2 of 2 matched items."))
	})

	It("cancels every kind and priority when neither filter is given", func() {
		Expect(cancelArtwork(ctx, ds, nil, nil, false, accept, &out)).To(Succeed())
		Expect(queue.Count()).To(BeZero())
	})

	It("stops at a selection that matches nothing instead of prompting", func() {
		refuse := func(io.Writer, int64, int64) bool {
			Fail("must not prompt when nothing matches")
			return false
		}
		Expect(cancelArtwork(ctx, ds, []model.Kind{model.KindPlaylistArtwork}, nil, false, refuse, &out)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("Nothing matches this selection."))
		Expect(queue.Count()).To(BeNumerically("==", 3))
	})

	It("reports a queue read failure instead of reporting nothing to cancel", func() {
		queue.Err = errors.New("read failed")
		Expect(cancelArtwork(ctx, ds, nil, nil, false, accept, &out)).To(MatchError(ContainSubstring("read failed")))
	})
})
