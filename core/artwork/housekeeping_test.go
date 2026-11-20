package artwork

import (
	"context"
	"slices"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RefreshableKinds", func() {
	// The two are meant to describe the same fact. Nothing but this test stops them from drifting,
	// and a drift would have `artwork explain` report state for a kind that keeps none.
	It("holds exactly the kinds that keep state", func() {
		for _, k := range []model.Kind{
			model.KindArtistArtwork, model.KindAlbumArtwork, model.KindPlaylistArtwork,
			model.KindRadioArtwork, model.KindMediaFileArtwork, model.KindDiscArtwork,
		} {
			Expect(slices.Contains(RefreshableKinds, k)).To(Equal(KeepsState(k)), k.String())
		}
	})
})

var _ = Describe("Housekeeping", func() {
	var (
		ctx context.Context
		ds  model.DataStore
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) {
			c.CoverArtPriority = "embedded, folder"
			c.ArtistArtPriority = "artist.jpg"
			c.Agents = "spotify"
			c.EnableExternalServices = true
		})

		ds = newTestDS(ctx)
	})

	Describe("Fingerprint", func() {
		DescribeTable("changes when a fingerprint-affecting config value changes",
			func(change func(*conf.Options)) {
				before := ConfigFingerprint()
				conf.Update(change)
				Expect(ConfigFingerprint()).NotTo(Equal(before))
			},
			Entry("CoverArtPriority", func(c *conf.Options) { c.CoverArtPriority = "folder, embedded" }),
			Entry("ArtistImageFolder", func(c *conf.Options) { c.ArtistImageFolder = "/after" }),
			Entry("EnableM3UExternalAlbumArt", func(c *conf.Options) { c.EnableM3UExternalAlbumArt = true }),
		)

		// Pinned: a changed formula tells every existing install its artwork config went stale.
		It("hashes a given config to a stable value", func() {
			conf.Update(func(c *conf.Options) {
				c.CoverArtPriority = "cover.*, embedded"
				c.ArtistArtPriority = "artist.*, external"
				c.ArtistImageFolder = ""
				c.Agents = "lastfm,spotify"
				c.EnableExternalServices = true
				c.EnableM3UExternalAlbumArt = false
			})

			Expect(ConfigFingerprint()).To(Equal("7b538a83a870c16d"))
		})

		It("reports the config inputs it hashes, so a change can be traced to a setting", func() {
			conf.Update(func(c *conf.Options) {
				c.Agents = "lastfm,spotify"
				c.CoverArtPriority = "cover.*, embedded"
			})

			Expect(FingerprintInputs()).To(ContainElements(
				FingerprintInput{Name: "Agents", Value: "lastfm,spotify"},
				FingerprintInput{Name: "CoverArtPriority", Value: "cover.*, embedded"},
			))
		})

		It("does not change when the server version changes", func() {
			original := consts.Version
			DeferCleanup(func() { consts.Version = original })
			f1 := ConfigFingerprint()
			consts.Version = original + "-next"
			Expect(ConfigFingerprint()).To(Equal(f1),
				"the version must not invalidate artwork state: every build would report a stale config")
		})
	})

	Describe("ReconcileConfigFingerprint", func() {
		It("records the current fingerprint when none was ever stored", func() {
			Expect(ReconcileConfigFingerprint(ctx, ds)).To(Succeed())

			Expect(ds.Property().Get(ctx, consts.ArtConfFingerprintPropertyKey)).To(Equal(ConfigFingerprint()))
		})

		It("leaves a stale fingerprint stored, so the warning survives a restart", func() {
			Expect(ds.Property().Put(ctx, consts.ArtConfFingerprintPropertyKey, "stale-fingerprint")).To(Succeed())

			Expect(ReconcileConfigFingerprint(ctx, ds)).To(Succeed())

			Expect(ds.Property().Get(ctx, consts.ArtConfFingerprintPropertyKey)).To(Equal("stale-fingerprint"))
		})
	})

	Describe("EnqueueMissingAll", func() {
		BeforeEach(func() {
			putAlbums(ctx, ds, model.Album{ID: "al1", Name: "Album 1"}, model.Album{ID: "al2", Name: "Album 2"})
			putArtists(ctx, ds, model.Artist{ID: "ar1", Name: "Artist"})
			putPlaylist(ctx, ds, model.Playlist{ID: "pl1", Name: "Playlist"})
			putRadio(ctx, ds, model.Radio{ID: "ra1", Name: "Radio"})
		})

		It("enqueues only entities that have no item_artwork row, across all kinds", func() {
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary, Hash: "somehash"})).To(Succeed())
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "ar", ItemID: "ar1", ImageType: model.ImageTypePrimary, Hash: ""})).To(Succeed())

			err := enqueueMissingAll(ctx, ds)
			Expect(err).ToNot(HaveOccurred())

			stats, err := ds.ArtworkQueue().CountQueued(ctx, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(stats).ToNot(BeEmpty())
			for _, st := range stats {
				Expect(st.Priority).To(Equal(model.ArtworkPriorityRecheck))
			}
			Expect(queued(ctx, ds, "al", "al2")).ToNot(BeNil())
			Expect(queued(ctx, ds, "pl", "pl1")).ToNot(BeNil())
			Expect(queued(ctx, ds, "ra", "ra1")).ToNot(BeNil())
			Expect(queued(ctx, ds, "al", "al1")).To(BeNil())
			Expect(queued(ctx, ds, "ar", "ar1")).To(BeNil())
		})
	})
})

var _ = Describe("ItemName", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = newTestDS(ctx)
		putAlbums(ctx, ds,
			model.Album{ID: "al-1", Name: "Kid A"},
			model.Album{ID: "al-2", Name: "Sandinista!", Discs: model.Discs{2: "Side Three"}},
		)
		putArtists(ctx, ds, model.Artist{ID: "ar-1", Name: "Radiohead"})
	})

	It("returns the album name", func() {
		Expect(ItemName(ctx, ds, model.KindAlbumArtwork, "al-1")).To(Equal("Kid A"))
	})

	It("returns the artist name", func() {
		Expect(ItemName(ctx, ds, model.KindArtistArtwork, "ar-1")).To(Equal("Radiohead"))
	})

	It("errors for an unknown album", func() {
		_, err := ItemName(ctx, ds, model.KindAlbumArtwork, "nope")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("errors for an unsupported kind", func() {
		// model.Kind is a struct with unexported fields, so the zero value is the only
		// unsupported Kind constructible from outside package model.
		_, err := ItemName(ctx, ds, model.Kind{}, "al-1")
		Expect(err).To(HaveOccurred())
	})

	Context("disc artwork", func() {
		It("names the album, the disc and its subtitle", func() {
			Expect(ItemName(ctx, ds, model.KindDiscArtwork, "al-2:2")).
				To(Equal("Sandinista! (disc 2): Side Three"))
		})

		It("omits the subtitle when the disc has none", func() {
			Expect(ItemName(ctx, ds, model.KindDiscArtwork, "al-2:1")).
				To(Equal("Sandinista! (disc 1)"))
		})

		It("rejects an id that is not <albumID>:<disc>", func() {
			_, err := ItemName(ctx, ds, model.KindDiscArtwork, "al-2")
			Expect(err).To(HaveOccurred())
		})
	})
})
