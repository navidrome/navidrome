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
		Expect(libAfter.TotalMissingFiles).To(BeZero())
	})
})
