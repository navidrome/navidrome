package persistence

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TranscodingRepository", func() {
	var repo model.TranscodingRepository
	var ctx, adminCtx context.Context

	BeforeEach(func() {
		ctx = request.WithUser(log.NewContext(GinkgoT().Context()), regularUser)
		adminCtx = request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
		repo = NewTranscodingRepository(GetDBXBuilder())
	})

	AfterEach(func() {
		// Clean up any transcoding created during the tests
		tc, err := repo.FindByFormat(adminCtx, "test_format")
		if err == nil {
			err = repo.Delete(adminCtx, tc.ID)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Describe("Admin User", func() {
		It("creates a new transcoding", func() {
			base, err := repo.CountAll(adminCtx)
			Expect(err).ToNot(HaveOccurred())

			err = repo.Put(adminCtx, &model.Transcoding{ID: "new", Name: "new", TargetFormat: "test_format", DefaultBitRate: 320, Command: "ffmpeg"})
			Expect(err).ToNot(HaveOccurred())

			count, err := repo.CountAll(adminCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(base + 1))
		})

		It("updates an existing transcoding", func() {
			tr := &model.Transcoding{ID: "upd", Name: "old", TargetFormat: "test_format", DefaultBitRate: 100, Command: "ffmpeg"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())
			tr.Name = "updated"
			err := repo.Put(adminCtx, tr)
			Expect(err).ToNot(HaveOccurred())
			res, err := repo.FindByFormat(adminCtx, "test_format")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal("updated"))
		})

		It("deletes a transcoding", func() {
			err := repo.Put(adminCtx, &model.Transcoding{ID: "to-delete", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 256, Command: "ffmpeg"})
			Expect(err).ToNot(HaveOccurred())
			err = repo.Delete(adminCtx, "to-delete")
			Expect(err).ToNot(HaveOccurred())
			_, err = repo.Get(adminCtx, "to-delete")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns not found when deleting a missing transcoding", func() {
			err := repo.(*transcodingRepository).Delete(adminCtx, "does-not-exist")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("reads the Command field via the REST Read method", func() {
			tr := &model.Transcoding{ID: "adminread", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg -secret"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			res, err := repo.Read(adminCtx, "adminread")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Command).To(Equal("ffmpeg -secret"))
		})
	})

	Describe("Regular User", func() {
		It("reads a transcoding but with the Command field redacted", func() {
			tr := &model.Transcoding{ID: "readreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg -secret"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			t, err := repo.Read(ctx, "readreg")
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Name).To(Equal("temp"))
			Expect(t.TargetFormat).To(Equal("test_format"))
			Expect(t.Command).To(BeEmpty())
		})

		It("lists transcodings but with the Command field redacted", func() {
			tr := &model.Transcoding{ID: "listreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg -secret"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			list, err := repo.ReadAll(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).ToNot(BeEmpty())
			for _, t := range list {
				Expect(t.Command).To(BeEmpty())
			}
		})

		It("counts transcodings", func() {
			count, err := repo.Count(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeNumerically(">=", 0))
		})

		It("can still resolve a transcoding for streaming via Get (Command not redacted)", func() {
			tr := &model.Transcoding{ID: "streamreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg -secret"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			res, err := repo.Get(ctx, "streamreg")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.ID).To(Equal("streamreg"))
			Expect(res.Command).To(Equal("ffmpeg -secret"))
		})

		It("can still resolve a transcoding for streaming via FindByFormat (Command not redacted)", func() {
			tr := &model.Transcoding{ID: "fmtreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg -secret"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			res, err := repo.FindByFormat(ctx, "test_format")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.ID).To(Equal("fmtreg"))
			Expect(res.Command).To(Equal("ffmpeg -secret"))
		})

		It("fails to create", func() {
			err := repo.Put(ctx, &model.Transcoding{ID: "bad", Name: "bad", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"})
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})

		It("fails to update", func() {
			tr := &model.Transcoding{ID: "updreg", Name: "old", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			tr.Name = "bad"
			err := repo.Put(ctx, tr)
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})

		It("fails to delete", func() {
			tr := &model.Transcoding{ID: "delreg", Name: "temp", TargetFormat: "test_format", DefaultBitRate: 64, Command: "ffmpeg"}
			Expect(repo.Put(adminCtx, tr)).To(Succeed())

			err := repo.Delete(ctx, "delreg")
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})
	})
})
