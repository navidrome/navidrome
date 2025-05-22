package persistence

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TranscodingRepository", func() {
	var repo model.TranscodingRepository
	var adminRepo model.TranscodingRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		ctx = request.WithUser(ctx, regularUser)
		repo = NewTranscodingRepository(ctx, GetDBXBuilder())

		adminCtx := log.NewContext(GinkgoT().Context())
		adminCtx = request.WithUser(adminCtx, adminUser)
		adminRepo = NewTranscodingRepository(adminCtx, GetDBXBuilder())
	})

	AfterEach(func() {
		// Clean up any transcoding created during the tests
		tc, err := adminRepo.FindByFormat("test_format")
		if err == nil {
			err = adminRepo.(*transcodingRepository).Delete(tc.ID)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Describe("Admin User", func() {
		It("creates a new transcoding", func() {
			base, err := adminRepo.CountAll()
			Expect(err).ToNot(HaveOccurred())

			err = adminRepo.Put(&model.Transcoding{ID: "new", Name: "new", TargetFormat: "test_format", DefaultBitRate: 320, Command: "ffmpeg"})
			Expect(err).ToNot(HaveOccurred())

			count, err := adminRepo.CountAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(base + 1))
		})

		It("updates an existing transcoding", func() {
			tr := &model.Transcoding{ID: "upd", Name: "old", TargetFormat: "test_format", DefaultBitRate: 100, Command: "ffmpeg"}
			Expect(adminRepo.Put(tr)).To(Succeed())
			tr.Name = "updated"
			err := adminRepo.Put(tr)
			Expect(err).ToNot(HaveOccurred())
			res, err := adminRepo.FindByFormat("test_format")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal("updated"))
		})

		It("deletes a transcoding", func() {
			err := adminRepo.Put(&model.Transcoding{ID: "to-delete", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 256, Command: "ffmpeg"})
			Expect(err).ToNot(HaveOccurred())
			err = adminRepo.(*transcodingRepository).Delete("to-delete")
			Expect(err).ToNot(HaveOccurred())
			_, err = adminRepo.Get("to-delete")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Regular User", func() {
		It("fails to create", func() {
			err := repo.Put(&model.Transcoding{ID: "bad", Name: "bad", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"})
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})

		It("fails to update", func() {
			tr := &model.Transcoding{ID: "updreg", Name: "old", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"}
			Expect(adminRepo.Put(tr)).To(Succeed())

			tr.Name = "bad"
			err := repo.Put(tr)
			Expect(err).To(Equal(rest.ErrPermissionDenied))

			//_ = adminRepo.(*transcodingRepository).Delete("updreg")
		})

		It("fails to delete", func() {
			tr := &model.Transcoding{ID: "delreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"}
			Expect(adminRepo.Put(tr)).To(Succeed())

			err := repo.(*transcodingRepository).Delete("delreg")
			Expect(err).To(Equal(rest.ErrPermissionDenied))

			//_ = adminRepo.(*transcodingRepository).Delete("delreg")
		})
	})
})
