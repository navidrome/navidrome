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

	It("refreshes stats", func() {
		libBefore, err := repo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.RefreshStats(1)).To(Succeed())
		libAfter, err := repo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		Expect(libAfter.UpdatedAt).To(BeTemporally(">", libBefore.UpdatedAt))

		var songsRes, albumsRes, artistsRes, foldersRes, filesRes, missingRes struct{ Count int64 }
		var sizeRes struct{ Sum int64 }

		Expect(conn.NewQuery("select count(*) as count from media_file where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&songsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from album where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&albumsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from library_artist la join artist a on la.artist_id = a.id where la.library_id = {:id} and a.missing = 0").Bind(dbx.Params{"id": 1}).One(&artistsRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from folder where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&foldersRes)).To(Succeed())
		Expect(conn.NewQuery("select ifnull(sum(num_audio_files + num_playlists + json_array_length(image_files)),0) as count from folder where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&filesRes)).To(Succeed())
		Expect(conn.NewQuery("select count(*) as count from media_file where library_id = {:id} and missing = 1").Bind(dbx.Params{"id": 1}).One(&missingRes)).To(Succeed())
		Expect(conn.NewQuery("select ifnull(sum(size),0) as sum from album where library_id = {:id} and missing = 0").Bind(dbx.Params{"id": 1}).One(&sizeRes)).To(Succeed())

		Expect(libAfter.TotalSongs).To(Equal(int(songsRes.Count)))
		Expect(libAfter.TotalAlbums).To(Equal(int(albumsRes.Count)))
		Expect(libAfter.TotalArtists).To(Equal(int(artistsRes.Count)))
		Expect(libAfter.TotalFolders).To(Equal(int(foldersRes.Count)))
		Expect(libAfter.TotalFiles).To(Equal(int(filesRes.Count)))
		Expect(libAfter.TotalMissingFiles).To(Equal(int(missingRes.Count)))
		Expect(libAfter.TotalSize).To(Equal(sizeRes.Sum))
	})
})
