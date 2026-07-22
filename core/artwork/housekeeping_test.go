package artwork

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// orderTrackingQueueRepo records the item kind of each Enqueue call, so tests can
// assert phase ordering (artists-first) that same-priority timestamps can't guarantee.
type orderTrackingQueueRepo struct {
	*tests.MockArtworkQueueRepo
	callKinds []string
}

func (o *orderTrackingQueueRepo) Enqueue(items ...model.ArtworkQueueItem) error {
	if len(items) > 0 {
		o.callKinds = append(o.callKinds, items[0].ItemKind)
	}
	return o.MockArtworkQueueRepo.Enqueue(items...)
}

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

	seedEntities := func() {
		artistRepo := tests.CreateMockArtistRepo()
		artistRepo.SetData(model.Artists{{ID: "ar1"}, {ID: "ar2"}})
		ds.MockedArtist = artistRepo

		albumRepo := tests.CreateMockAlbumRepo()
		albumRepo.SetData(model.Albums{{ID: "al1"}})
		ds.MockedAlbum = albumRepo

		playlistRepo := tests.CreateMockPlaylistRepo()
		playlistRepo.SetData(model.Playlists{{ID: "pl1"}})
		ds.MockedPlaylist = playlistRepo

		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.All = model.Radios{{ID: "ra1"}}
		ds.MockedRadio = radioRepo
	}

	Describe("Fingerprint", func() {
		It("changes when a fingerprint-affecting config value changes", func() {
			f1 := Fingerprint()
			conf.Server.CoverArtPriority = "folder, embedded"
			f2 := Fingerprint()
			Expect(f1).NotTo(Equal(f2))
		})
	})

	Describe("Backfill", func() {
		It("enqueues nothing and returns false when the stored fingerprint matches", func() {
			seedEntities()
			Expect(propRepo.Put(FingerprintPropertyKey, Fingerprint())).To(Succeed())

			did, err := Backfill(ctx, ds)
			Expect(err).ToNot(HaveOccurred())
			Expect(did).To(BeFalse())

			count, err := queueRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeZero())
		})

		It("runs the backfill when no fingerprint was ever stored", func() {
			seedEntities()

			did, err := Backfill(ctx, ds)
			Expect(err).ToNot(HaveOccurred())
			Expect(did).To(BeTrue())

			count, err := queueRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(5))) // 2 artists + 1 album + 1 playlist + 1 radio

			stored, err := propRepo.Get(FingerprintPropertyKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(stored).To(Equal(Fingerprint()))
		})

		It("enqueues artists before albums/playlists/radios, all at Backfill priority", func() {
			seedEntities()
			Expect(propRepo.Put(FingerprintPropertyKey, "stale-fingerprint")).To(Succeed())

			did, err := Backfill(ctx, ds)
			Expect(err).ToNot(HaveOccurred())
			Expect(did).To(BeTrue())

			Expect(queueRepo.callKinds).ToNot(BeEmpty())
			artistCallIdx := -1
			for i, k := range queueRepo.callKinds {
				if k == "ar" {
					artistCallIdx = i
					break
				}
			}
			Expect(artistCallIdx).To(Equal(0), "artists must be the first Enqueue call")
			for i, k := range queueRepo.callKinds {
				if k != "ar" {
					Expect(i).To(BeNumerically(">", artistCallIdx))
				}
			}

			for _, it := range queueRepo.Data {
				Expect(it.Priority).To(Equal(model.ArtworkPriorityBackfill))
				Expect(it.ItemKind).To(BeElementOf("ar", "al", "pl", "ra"))
			}
		})
	})

	Describe("EnqueueStaleAbsentAll", func() {
		var artRepo *tests.MockArtworkRepo

		BeforeEach(func() {
			artRepo = tests.CreateMockArtworkRepo()
			ds.MockedArtwork = artRepo
			queueRepo.ItemArtworkSource = artRepo
		})

		It("enqueues only absent entries older than the recheck window, across all kinds", func() {
			old := time.Now().Add(-48 * time.Hour)
			recent := time.Now().Add(-time.Hour)

			artRepo.ItemData["ar-stale"] = model.ItemArtwork{ItemKind: "ar", ItemID: "ar1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old}
			artRepo.ItemData["al-stale"] = model.ItemArtwork{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old}
			artRepo.ItemData["pl-stale"] = model.ItemArtwork{ItemKind: "pl", ItemID: "pl1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old}
			artRepo.ItemData["ra-stale"] = model.ItemArtwork{ItemKind: "ra", ItemID: "ra1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old}
			// Not stale: too recent.
			artRepo.ItemData["ar-recent"] = model.ItemArtwork{ItemKind: "ar", ItemID: "ar2", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: recent}
			// Not absent: has a resolved hash.
			artRepo.ItemData["al-resolved"] = model.ItemArtwork{ItemKind: "al", ItemID: "al2", ImageType: model.ImageTypePrimary, Hash: "somehash", AttemptedAt: old}

			err := EnqueueStaleAbsentAll(ctx, ds)
			Expect(err).ToNot(HaveOccurred())

			Expect(queueRepo.Data).To(HaveLen(4))
			for _, it := range queueRepo.Data {
				Expect(it.Priority).To(Equal(model.ArtworkPriorityRecheck))
			}
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "ar", "ar1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "al", "al1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "pl", "pl1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "ra", "ra1")).ToNot(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "ar", "ar2")).To(BeNil())
			Expect(findQueued(queueRepo.MockArtworkQueueRepo, "al", "al2")).To(BeNil())
		})
	})
})
