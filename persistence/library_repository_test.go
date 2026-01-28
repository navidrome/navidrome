package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("LibraryRepository", func() {
	var repo model.LibraryRepository
	var ctx context.Context
	var conn *dbx.DB

	BeforeEach(func() {
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid"})
		conn = GetDBXBuilder()
		repo = NewLibraryRepository(ctx, conn)
	})

	AfterEach(func() {
		// Clean up test libraries (keep ID 1 which is the default library)
		_, _ = conn.NewQuery("DELETE FROM library WHERE id > 1").Execute()
	})

	Describe("Put", func() {
		Context("when ID is 0", func() {
			It("inserts a new library with autoassigned ID", func() {
				lib := &model.Library{
					ID:   0,
					Name: "Test Library",
					Path: "/music/test",
				}

				err := repo.Put(lib)
				Expect(err).ToNot(HaveOccurred())
				Expect(lib.ID).To(BeNumerically(">", 0))
				Expect(lib.CreatedAt).ToNot(BeZero())
				Expect(lib.UpdatedAt).ToNot(BeZero())

				// Verify it was inserted
				savedLib, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(savedLib.Name).To(Equal("Test Library"))
				Expect(savedLib.Path).To(Equal("/music/test"))
			})
		})

		Context("when ID is non-zero and record exists", func() {
			It("updates the existing record", func() {
				// First create a library
				lib := &model.Library{
					ID:   0,
					Name: "Original Library",
					Path: "/music/original",
				}
				err := repo.Put(lib)
				Expect(err).ToNot(HaveOccurred())

				originalID := lib.ID
				originalCreatedAt := lib.CreatedAt

				// Now update it
				lib.Name = "Updated Library"
				lib.Path = "/music/updated"
				err = repo.Put(lib)
				Expect(err).ToNot(HaveOccurred())

				// Verify it was updated, not inserted
				Expect(lib.ID).To(Equal(originalID))
				Expect(lib.CreatedAt).To(Equal(originalCreatedAt))
				Expect(lib.UpdatedAt).To(BeTemporally(">", originalCreatedAt))

				// Verify the changes were saved
				savedLib, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(savedLib.Name).To(Equal("Updated Library"))
				Expect(savedLib.Path).To(Equal("/music/updated"))
			})
		})

		Context("when ID is non-zero but record doesn't exist", func() {
			It("inserts a new record with the specified ID", func() {
				lib := &model.Library{
					ID:   999,
					Name: "New Library with ID",
					Path: "/music/new",
				}

				// Ensure the record doesn't exist
				_, err := repo.Get(999)
				Expect(err).To(HaveOccurred())

				// Put should insert it
				err = repo.Put(lib)
				Expect(err).ToNot(HaveOccurred())
				Expect(lib.ID).To(Equal(999))
				Expect(lib.CreatedAt).ToNot(BeZero())
				Expect(lib.UpdatedAt).ToNot(BeZero())

				// Verify it was inserted with the correct ID
				savedLib, err := repo.Get(999)
				Expect(err).ToNot(HaveOccurred())
				Expect(savedLib.ID).To(Equal(999))
				Expect(savedLib.Name).To(Equal("New Library with ID"))
				Expect(savedLib.Path).To(Equal("/music/new"))
			})
		})
	})

	It("refreshes stats", func() {
		libBefore, err := repo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.RefreshStats(1)).To(Succeed())
		libAfter, err := repo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		Expect(libAfter.UpdatedAt).To(BeTemporally(">", libBefore.UpdatedAt))

		var songsRes, albumsRes, artistsRes, foldersRes, filesRes, missingRes struct{ Count int64 }
		var sizeRes struct{ Sum int64 }
		var durationRes struct{ Sum float64 }

		Expect(conn.NewQuery("select count(*) as count from media_file where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&songsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from album where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&albumsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from library_artist la join artist a on la.artist_id = a.id where la.library_id = {:id} and a.missing = 0").Bind(dbx.Params{"id": 1}).One(&artistsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from folder where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&foldersRes)).To(Succeed())
		Expect(conn.NewQuery("select ifnull(sum(num_audio_files + num_playlists + json_array_length(image_files)),0) as count from folder where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&filesRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from media_file where library_id = {:id} and missing = 1").Bind(dbx.Params{"id": 1}).One(&missingRes)).To(Succeed())
		Expect(conn.NewQuery("select ifnull(sum(size),0) as sum from album where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&sizeRes)).To(Succeed())
		Expect(conn.NewQuery("select ifnull(sum(duration),0) as sum from album where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&durationRes)).To(Succeed())

		Expect(libAfter.TotalSongs).To(Equal(int(songsRes.Count)))
		Expect(libAfter.TotalAlbums).To(Equal(int(albumsRes.Count)))
		Expect(libAfter.TotalArtists).To(Equal(int(artistsRes.Count)))
		Expect(libAfter.TotalFolders).To(Equal(int(foldersRes.Count)))
		Expect(libAfter.TotalFiles).To(Equal(int(filesRes.Count)))
		Expect(libAfter.TotalMissingFiles).To(Equal(int(missingRes.Count)))
		Expect(libAfter.TotalSize).To(Equal(sizeRes.Sum))
		Expect(libAfter.TotalDuration).To(Equal(durationRes.Sum))
	})

	Describe("ScanBegin and ScanEnd", func() {
		var lib *model.Library

		BeforeEach(func() {
			lib = &model.Library{
				ID:   0,
				Name: "Test Scan Library",
				Path: "/music/test-scan",
			}
			err := repo.Put(lib)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("ScanBegin",
			func(fullScan bool, expectedFullScanInProgress bool) {
				err := repo.ScanBegin(lib.ID, fullScan)
				Expect(err).ToNot(HaveOccurred())

				updatedLib, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedLib.LastScanStartedAt).ToNot(BeZero())
				Expect(updatedLib.FullScanInProgress).To(Equal(expectedFullScanInProgress))
			},
			Entry("sets FullScanInProgress to true for full scan", true, true),
			Entry("sets FullScanInProgress to false for quick scan", false, false),
		)

		Context("ScanEnd", func() {
			BeforeEach(func() {
				err := repo.ScanBegin(lib.ID, true)
				Expect(err).ToNot(HaveOccurred())
			})

			It("sets LastScanAt and clears FullScanInProgress and LastScanStartedAt", func() {
				err := repo.ScanEnd(lib.ID)
				Expect(err).ToNot(HaveOccurred())

				updatedLib, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedLib.LastScanAt).ToNot(BeZero())
				Expect(updatedLib.FullScanInProgress).To(BeFalse())
				Expect(updatedLib.LastScanStartedAt).To(BeZero())
			})

			It("sets LastScanAt to be after LastScanStartedAt", func() {
				libBefore, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())

				err = repo.ScanEnd(lib.ID)
				Expect(err).ToNot(HaveOccurred())

				libAfter, err := repo.Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(libAfter.LastScanAt).To(BeTemporally(">=", libBefore.LastScanStartedAt))
			})
		})
	})

	Describe("Ignored libraries", func() {
		var normalLib, ignoredLib *model.Library

		BeforeEach(func() {
			// Create a normal library
			normalLib = &model.Library{
				ID:   0,
				Name: "Normal Library",
				Path: "/music/normal",
			}
			err := repo.Put(normalLib)
			Expect(err).ToNot(HaveOccurred())

			// Create an ignored library
			ignoredLib = &model.Library{
				ID:      0,
				Name:    "Ignored Library",
				Path:    "/music/ignored",
				Ignored: true,
			}
			err = repo.Put(ignoredLib)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters out ignored libraries in GetAll for non-admin users", func() {
			libs, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			// Check that ignored library is not in the list for non-admin users
			var foundNormal, foundIgnored bool
			for _, lib := range libs {
				if lib.ID == normalLib.ID {
					foundNormal = true
				}
				if lib.ID == ignoredLib.ID {
					foundIgnored = true
				}
			}

			Expect(foundNormal).To(BeTrue(), "Normal library should be in the list")
			Expect(foundIgnored).To(BeFalse(), "Ignored library should not be in the list for non-admin users")
		})

		It("includes ignored libraries in GetAll for admin users", func() {
			// Create admin context
			adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "adminid", IsAdmin: true})
			adminRepo := NewLibraryRepository(adminCtx, conn)

			libs, err := adminRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			// Check that both libraries are in the list for admin users
			var foundNormal, foundIgnored bool
			for _, lib := range libs {
				if lib.ID == normalLib.ID {
					foundNormal = true
				}
				if lib.ID == ignoredLib.ID {
					foundIgnored = true
				}
			}

			Expect(foundNormal).To(BeTrue(), "Normal library should be in the list for admin")
			Expect(foundIgnored).To(BeTrue(), "Ignored library should be in the list for admin users")
		})

		It("filters out ignored libraries in CountAll for non-admin users", func() {
			// Get count before
			countBefore, err := repo.CountAll()
			Expect(err).ToNot(HaveOccurred())

			// Count should not include ignored library for non-admin users
			// (it should be the same as before adding the ignored library)
			libs, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(countBefore).To(Equal(int64(len(libs))))
		})

		It("includes ignored libraries in CountAll for admin users", func() {
			// Create admin context
			adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "adminid", IsAdmin: true})
			adminRepo := NewLibraryRepository(adminCtx, conn)

			countAll, err := adminRepo.CountAll()
			Expect(err).ToNot(HaveOccurred())

			libs, err := adminRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			// Count should match the number of libraries returned (including ignored)
			Expect(countAll).To(Equal(int64(len(libs))))

			// Verify the list includes both normal and ignored libraries
			var foundIgnored bool
			for _, lib := range libs {
				if lib.ID == ignoredLib.ID {
					foundIgnored = true
				}
			}
			Expect(foundIgnored).To(BeTrue(), "Ignored library should be counted for admin users")
		})

		It("can still retrieve ignored libraries by ID", func() {
			lib, err := repo.Get(ignoredLib.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(lib.Ignored).To(BeTrue())
			Expect(lib.Name).To(Equal("Ignored Library"))
		})

		It("can update a library to set ignored status", func() {
			// Get the normal library
			lib, err := repo.Get(normalLib.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(lib.Ignored).To(BeFalse())

			// Set it to ignored
			lib.Ignored = true
			err = repo.Put(lib)
			Expect(err).ToNot(HaveOccurred())

			// Verify it's now ignored and not in the list for non-admin users
			libs, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			for _, l := range libs {
				Expect(l.ID).ToNot(Equal(normalLib.ID), "Ignored library should not be in the list for non-admin users")
			}

			// But admin users should still see it
			adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "adminid", IsAdmin: true})
			adminRepo := NewLibraryRepository(adminCtx, conn)
			adminLibs, err := adminRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			var foundInAdminList bool
			for _, l := range adminLibs {
				if l.ID == normalLib.ID {
					foundInAdminList = true
					Expect(l.Ignored).To(BeTrue())
				}
			}
			Expect(foundInAdminList).To(BeTrue(), "Ignored library should be in the list for admin users")

			// And can still get it by ID
			lib, err = repo.Get(normalLib.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(lib.Ignored).To(BeTrue())
		})

		It("includes ignored libraries for headless processes (background operations)", func() {
			// Create a headless context (no user context, like a background scanner)
			headlessCtx := log.NewContext(context.TODO())
			headlessRepo := NewLibraryRepository(headlessCtx, conn)

			// Headless processes should see all libraries including ignored ones
			// This is important for operations like library scanning
			libs, err := headlessRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			var foundNormal, foundIgnored bool
			for _, lib := range libs {
				if lib.ID == normalLib.ID {
					foundNormal = true
				}
				if lib.ID == ignoredLib.ID {
					foundIgnored = true
				}
			}

			Expect(foundNormal).To(BeTrue(), "Normal library should be visible to headless processes")
			Expect(foundIgnored).To(BeTrue(), "Ignored library should be visible to headless processes for background operations")
		})
	})
})
