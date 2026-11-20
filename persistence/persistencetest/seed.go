package persistencetest

import (
	"cmp"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// SetLibraries points library 1 at paths[0], creates libraries 2, 3, ... for the other paths, and
// grants every existing user access to all of them. It returns the library IDs in order.
func SetLibraries(ctx context.Context, ds model.DataStore, paths ...string) []int {
	ginkgo.GinkgoHelper()
	// Library().Put refuses to change library 1's path; StoreMusicFolder is the only way.
	conf.Update(func(c *conf.Options) { c.MusicFolder = paths[0] })
	gomega.Expect(ds.Library().StoreMusicFolder(ctx)).To(gomega.Succeed())
	ids := []int{model.DefaultLibraryID}
	for i, path := range paths[1:] {
		lib := &model.Library{ID: i + 2, Name: "Library " + strconv.Itoa(i+2), Path: path}
		gomega.Expect(ds.Library().Put(ctx, lib)).To(gomega.Succeed())
		ids = append(ids, lib.ID)
	}
	users, err := ds.User().GetAll(request.WithUser(ctx, model.User{IsAdmin: true}))
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	for _, u := range users {
		gomega.Expect(ds.User().SetUserLibraries(ctx, u.ID, ids)).To(gomega.Succeed())
	}
	return ids
}

// PutMediaFiles stores copies of mfs, in library 1 unless they name another and at ID+".mp3" unless
// they have a path. It returns the stored copies, which carry the IDs the repository generated.
func PutMediaFiles(ctx context.Context, ds model.DataStore, mfs ...model.MediaFile) model.MediaFiles {
	ginkgo.GinkgoHelper()
	stored := make(model.MediaFiles, len(mfs))
	for i, mf := range mfs {
		mf.LibraryID = cmp.Or(mf.LibraryID, model.DefaultLibraryID)
		mf.Path = cmp.Or(mf.Path, mf.ID+".mp3")
		gomega.Expect(ds.MediaFile().Put(ctx, &mf)).To(gomega.Succeed())
		stored[i] = mf
	}
	return stored
}

// PutAlbums stores copies of albums, in library 1 unless they name another, sorted by their
// lowercased name unless they set OrderAlbumName.
func PutAlbums(ctx context.Context, ds model.DataStore, albums ...model.Album) {
	ginkgo.GinkgoHelper()
	for _, al := range albums {
		al.LibraryID = cmp.Or(al.LibraryID, model.DefaultLibraryID)
		al.OrderAlbumName = cmp.Or(al.OrderAlbumName, strings.ToLower(al.Name))
		gomega.Expect(ds.Album().Put(ctx, &al)).To(gomega.Succeed())
	}
}

// PutArtists stores copies of artists linked to library libID: no query returns an artist outside
// every library.
func PutArtists(ctx context.Context, ds model.DataStore, libID int, artists ...model.Artist) {
	ginkgo.GinkgoHelper()
	for _, ar := range artists {
		gomega.Expect(ds.Artist().Put(ctx, &ar)).To(gomega.Succeed())
		gomega.Expect(ds.Library().AddArtist(ctx, libID, ar.ID)).To(gomega.Succeed())
	}
}

// SetArtistStats replaces the per-role stats of artists already linked to library libID. The
// scanner computes them, and role-filtered artist queries skip an artist without them.
func SetArtistStats(ctx context.Context, libID int, stats map[model.Role]model.ArtistStats, artistIDs ...string) {
	ginkgo.GinkgoHelper()
	if len(artistIDs) == 0 {
		return
	}
	counters := make(map[string]map[string]int64, len(stats))
	for role, s := range stats {
		counters[role.String()] = map[string]int64{"m": int64(s.SongCount), "a": int64(s.AlbumCount), "s": s.Size}
	}
	js, err := json.Marshal(counters)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	args := []any{string(js), libID}
	for _, id := range artistIDs {
		args = append(args, id)
	}
	res, err := db.Db().ExecContext(ctx, "UPDATE library_artist SET stats = ? WHERE library_id = ? AND artist_id IN (?"+
		strings.Repeat(",?", len(artistIDs)-1)+")", args...)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(res.RowsAffected()).To(gomega.BeEquivalentTo(len(artistIDs)), "artists not linked to library %d", libID)
}
