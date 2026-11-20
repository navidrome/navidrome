package persistencetest

import (
	"context"
	"strconv"

	"github.com/navidrome/navidrome/conf"
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
