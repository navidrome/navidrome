package artwork

import (
	"context"
	"slices"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// orderTrackingQueueRepo records the kind of each bulk insert, so tests can assert
// phase ordering (artists-first) that same-priority timestamps can't guarantee.
type orderTrackingQueueRepo struct {
	*tests.MockArtworkQueueRepo
	callKinds []string
}

func (o *orderTrackingQueueRepo) EnqueueAllMissing(kind model.Kind, priority int) (int64, error) {
	o.callKinds = append(o.callKinds, kind.Prefix())
	return o.MockArtworkQueueRepo.EnqueueAllMissing(kind, priority)
}

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
		ctx       context.Context
		ds        *tests.MockDataStore
		queueRepo *orderTrackingQueueRepo
		propRepo  *tests.MockedPropertyRepo
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		conf.Server.CoverArtPriority = "embedded, folder"
		conf.Server.ArtistArtPriority = "artist.jpg"
		conf.Server.Agents = "spotify"
		conf.Server.EnableExternalServices = true

		queueRepo = &orderTrackingQueueRepo{MockArtworkQueueRepo: tests.CreateMockArtworkQueueRepo()}
		propRepo = &tests.MockedPropertyRepo{}
		ds = &tests.MockDataStore{MockedArtworkQueue: queueRepo, MockedProperty: propRepo}
	})

	Describe("Fingerprint", func() {
		It("changes when a fingerprint-affecting config value changes", func() {
			f1 := ConfigFingerprint()
			conf.Server.CoverArtPriority = "folder, embedded"
			f2 := ConfigFingerprint()
			Expect(f1).NotTo(Equal(f2))
		})

		It("changes when ArtistImageFolder changes", func() {
			conf.Server.ArtistImageFolder = "/before"
			f1 := ConfigFingerprint()
			conf.Server.ArtistImageFolder = "/after"
			Expect(ConfigFingerprint()).NotTo(Equal(f1))
		})

		It("changes when EnableM3UExternalAlbumArt is toggled", func() {
			conf.Server.EnableM3UExternalAlbumArt = false
			f1 := ConfigFingerprint()
			conf.Server.EnableM3UExternalAlbumArt = true
			Expect(ConfigFingerprint()).NotTo(Equal(f1))
		})

		// Pinned: a changed formula re-resolves every library on upgrade, flooding external providers.
		It("hashes a given config to a stable value", func() {
			conf.Server.CoverArtPriority = "cover.*, embedded"
			conf.Server.ArtistArtPriority = "artist.*, external"
			conf.Server.ArtistImageFolder = ""
			conf.Server.Agents = "lastfm,spotify"
			conf.Server.EnableExternalServices = true
			conf.Server.EnableM3UExternalAlbumArt = false

			Expect(ConfigFingerprint()).To(Equal("7b538a83a870c16d"))
		})

		It("reports the config inputs it hashes, so a change can be traced to a setting", func() {
			conf.Server.Agents = "lastfm,spotify"
			conf.Server.CoverArtPriority = "cover.*, embedded"

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
				"the version must not invalidate artwork state: it would re-resolve every entity on every build")
		})
	})

	Describe("ReconcileConfigFingerprint", func() {
		It("records the current fingerprint when none was ever stored", func() {
			Expect(ReconcileConfigFingerprint(ctx, ds)).To(Succeed())

			Expect(propRepo.Get(consts.ArtConfFingerprintPropertyKey)).To(Equal(ConfigFingerprint()))
			Expect(queueRepo.Count()).To(BeZero())
		})

		It("leaves a stale fingerprint stored, so the warning survives a restart", func() {
			Expect(propRepo.Put(consts.ArtConfFingerprintPropertyKey, "stale-fingerprint")).To(Succeed())

			Expect(ReconcileConfigFingerprint(ctx, ds)).To(Succeed())

			Expect(propRepo.Get(consts.ArtConfFingerprintPropertyKey)).To(Equal("stale-fingerprint"))
		})
	})

	Describe("MarkConfigApplied", func() {
		It("overwrites a stale fingerprint with the current one", func() {
			Expect(propRepo.Put(consts.ArtConfFingerprintPropertyKey, "stale-fingerprint")).To(Succeed())

			Expect(MarkConfigApplied(ctx, ds)).To(Succeed())

			Expect(propRepo.Get(consts.ArtConfFingerprintPropertyKey)).To(Equal(ConfigFingerprint()))
		})
	})

	Describe("EnqueueMissingAll", func() {
		var artRepo *tests.MockArtworkRepo

		BeforeEach(func() {
			artRepo = tests.CreateMockArtworkRepo()
			ds.MockedArtwork = artRepo
			queueRepo.ItemArtworkSource = artRepo
			queueRepo.ExistingIDs = map[string]map[string]bool{
				"al": {"al1": true, "al2": true},
				"ar": {"ar1": true},
				"pl": {"pl1": true},
				"ra": {"ra1": true},
			}
		})

		It("enqueues only entities that have no item_artwork row, across all kinds", func() {
			artRepo.ItemData["al-resolved"] = model.ItemArtwork{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary, Hash: "somehash"}
			artRepo.ItemData["ar-absent"] = model.ItemArtwork{ItemKind: "ar", ItemID: "ar1", ImageType: model.ImageTypePrimary, Hash: ""}

			err := enqueueMissingAll(ctx, ds)
			Expect(err).ToNot(HaveOccurred())

			for _, it := range queueRepo.Data {
				Expect(it.Priority).To(Equal(model.ArtworkPriorityRecheck))
			}
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "al", "al2")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "pl", "pl1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "ra", "ra1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "al", "al1")).To(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "ar", "ar1")).To(BeNil())
		})

		It("enqueues artists before every other kind", func() {
			Expect(enqueueMissingAll(ctx, ds)).To(Succeed())

			Expect(queueRepo.callKinds).ToNot(BeEmpty())
			firstOther := slices.IndexFunc(queueRepo.callKinds, func(k string) bool { return k != "ar" })
			Expect(firstOther).ToNot(Equal(0), "artists must be the first enqueue call")
			if firstOther >= 0 {
				Expect(queueRepo.callKinds[firstOther:]).ToNot(ContainElement("ar"),
					"no artist enqueue may follow another kind")
			}
		})
	})
})

var _ = Describe("ItemName", func() {
	var ds *tests.MockDataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		albumRepo := tests.CreateMockAlbumRepo()
		albumRepo.SetData(model.Albums{
			{ID: "al-1", Name: "Kid A"},
			{ID: "al-2", Name: "Sandinista!", Discs: model.Discs{2: "Side Three"}},
		})
		ds = &tests.MockDataStore{MockedAlbum: albumRepo}
		Expect(ds.Artist(ctx).(*tests.MockArtistRepo).Put(&model.Artist{ID: "ar-1", Name: "Radiohead"})).To(Succeed())
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
