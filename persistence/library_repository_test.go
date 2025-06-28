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
})
