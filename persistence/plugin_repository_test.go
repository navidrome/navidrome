package persistence

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PluginRepository", func() {
	var repo model.PluginRepository
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	Describe("Admin User", func() {
		BeforeEach(func() {
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			repo = NewPluginRepository(GetDBXBuilder())

			// Clean up any existing plugins
			all, _ := repo.GetAll(ctx)
			for _, p := range all {
				_ = repo.Delete(ctx, p.ID)
			}
		})

		AfterEach(func() {
			// Clean up after tests
			all, _ := repo.GetAll(ctx)
			for _, p := range all {
				_ = repo.Delete(ctx, p.ID)
			}
		})

		Describe("CountAll", func() {
			It("returns 0 when no plugins exist", func() {
				Expect(repo.CountAll(ctx)).To(Equal(int64(0)))
			})

			It("returns the number of plugins in the DB", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "test-plugin-1", Path: "/plugins/test1.wasm", Manifest: "{}", SHA256: "abc123"})
				_ = repo.Put(ctx, &model.Plugin{ID: "test-plugin-2", Path: "/plugins/test2.wasm", Manifest: "{}", SHA256: "def456"})

				Expect(repo.CountAll(ctx)).To(Equal(int64(2)))
			})
		})

		Describe("Delete", func() {
			It("deletes existing item", func() {
				plugin := &model.Plugin{ID: "to-delete", Path: "/plugins/delete.wasm", Manifest: "{}", SHA256: "hash"}
				_ = repo.Put(ctx, plugin)

				err := repo.Delete(ctx, plugin.ID)
				Expect(err).To(BeNil())

				_, err = repo.Get(ctx, plugin.ID)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("Get", func() {
			It("returns an existing item", func() {
				plugin := &model.Plugin{ID: "test-get", Path: "/plugins/test.wasm", Manifest: `{"name":"test"}`, SHA256: "hash123"}
				_ = repo.Put(ctx, plugin)

				res, err := repo.Get(ctx, plugin.ID)
				Expect(err).To(BeNil())
				Expect(res.ID).To(Equal(plugin.ID))
				Expect(res.Path).To(Equal(plugin.Path))
				Expect(res.Manifest).To(Equal(plugin.Manifest))
			})

			It("errors when missing", func() {
				_, err := repo.Get(ctx, "notanid")
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "plugin-a", Path: "/plugins/a.wasm", Manifest: "{}", SHA256: "hash1"})
				_ = repo.Put(ctx, &model.Plugin{ID: "plugin-b", Path: "/plugins/b.wasm", Manifest: "{}", SHA256: "hash2"})

				all, err := repo.GetAll(ctx)
				Expect(err).To(BeNil())
				Expect(all).To(HaveLen(2))
			})

			It("supports pagination", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "plugin-1", Path: "/plugins/1.wasm", Manifest: "{}", SHA256: "h1"})
				_ = repo.Put(ctx, &model.Plugin{ID: "plugin-2", Path: "/plugins/2.wasm", Manifest: "{}", SHA256: "h2"})
				_ = repo.Put(ctx, &model.Plugin{ID: "plugin-3", Path: "/plugins/3.wasm", Manifest: "{}", SHA256: "h3"})

				page1, err := repo.GetAll(ctx, model.QueryOptions{Max: 2, Offset: 0, Sort: "id"})
				Expect(err).To(BeNil())
				Expect(page1).To(HaveLen(2))

				page2, err := repo.GetAll(ctx, model.QueryOptions{Max: 2, Offset: 2, Sort: "id"})
				Expect(err).To(BeNil())
				Expect(page2).To(HaveLen(1))
			})
		})

		Describe("Put", func() {
			It("successfully creates a new plugin", func() {
				plugin := &model.Plugin{
					ID:       "new-plugin",
					Path:     "/plugins/new.wasm",
					Manifest: `{"name":"new","version":"1.0"}`,
					Config:   `{"setting":"value"}`,
					SHA256:   "sha256hash",
					Enabled:  false,
				}

				err := repo.Put(ctx, plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(ctx, plugin.ID)
				Expect(err).To(BeNil())
				Expect(saved.Path).To(Equal(plugin.Path))
				Expect(saved.Manifest).To(Equal(plugin.Manifest))
				Expect(saved.Config).To(Equal(plugin.Config))
				Expect(saved.Enabled).To(BeFalse())
				Expect(saved.CreatedAt).NotTo(BeZero())
				Expect(saved.UpdatedAt).NotTo(BeZero())
			})

			It("successfully updates an existing plugin", func() {
				plugin := &model.Plugin{
					ID:       "update-plugin",
					Path:     "/plugins/update.wasm",
					Manifest: `{"name":"test"}`,
					SHA256:   "original",
					Enabled:  false,
				}
				_ = repo.Put(ctx, plugin)

				plugin.Enabled = true
				plugin.Config = `{"new":"config"}`
				plugin.SHA256 = "updated"
				err := repo.Put(ctx, plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(ctx, plugin.ID)
				Expect(err).To(BeNil())
				Expect(saved.Enabled).To(BeTrue())
				Expect(saved.Config).To(Equal(`{"new":"config"}`))
				Expect(saved.SHA256).To(Equal("updated"))
			})

			It("stores and retrieves last_error", func() {
				plugin := &model.Plugin{
					ID:        "error-plugin",
					Path:      "/plugins/error.wasm",
					Manifest:  "{}",
					SHA256:    "hash",
					LastError: "failed to load: missing export",
				}
				err := repo.Put(ctx, plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(ctx, plugin.ID)
				Expect(err).To(BeNil())
				Expect(saved.LastError).To(Equal("failed to load: missing export"))
			})

			It("fails when ID is empty", func() {
				plugin := &model.Plugin{
					Path:     "/plugins/noid.wasm",
					Manifest: "{}",
					SHA256:   "hash",
				}
				err := repo.Put(ctx, plugin)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ID cannot be empty"))
			})
		})

		Describe("ClearErrors", func() {
			It("clears last_error on all plugins with errors", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "ok-plugin", Path: "/plugins/ok.wasm", Manifest: "{}", SHA256: "h1"})
				_ = repo.Put(ctx, &model.Plugin{ID: "err-plugin-1", Path: "/plugins/e1.wasm", Manifest: "{}", SHA256: "h2", LastError: "incompatible version"})
				_ = repo.Put(ctx, &model.Plugin{ID: "err-plugin-2", Path: "/plugins/e2.wasm", Manifest: "{}", SHA256: "h3", LastError: "missing export"})

				err := repo.ClearErrors(ctx)
				Expect(err).To(BeNil())

				all, err := repo.GetAll(ctx)
				Expect(err).To(BeNil())
				for _, p := range all {
					Expect(p.LastError).To(BeEmpty(), "plugin %s should have no error", p.ID)
				}
			})

			It("succeeds when no plugins have errors", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "clean-plugin", Path: "/plugins/c.wasm", Manifest: "{}", SHA256: "h1"})

				err := repo.ClearErrors(ctx)
				Expect(err).To(BeNil())
			})

			It("does not need the write lock when no plugins have errors", func() {
				_ = repo.Put(ctx, &model.Plugin{ID: "clean-plugin", Path: "/plugins/c.wasm", Manifest: "{}", SHA256: "h1"})
				conn, err := db.Db().Conn(GinkgoT().Context())
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(conn.Close)
				_, err = conn.ExecContext(GinkgoT().Context(), "BEGIN IMMEDIATE")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") })

				Expect(repo.ClearErrors(ctx)).To(Succeed())
			})
		})
	})

	Describe("Regular User", func() {
		BeforeEach(func() {
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: false})
			repo = NewPluginRepository(GetDBXBuilder())
		})

		Describe("CountAll", func() {
			It("fails to count items", func() {
				_, err := repo.CountAll(ctx)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Delete", func() {
			It("fails to delete items", func() {
				err := repo.Delete(ctx, "any-id")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Get", func() {
			It("fails to get items", func() {
				_, err := repo.Get(ctx, "any-id")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("GetAll", func() {
			It("fails to get all items", func() {
				_, err := repo.GetAll(ctx)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Put", func() {
			It("fails to create/update item", func() {
				err := repo.Put(ctx, &model.Plugin{
					ID:       "user-create",
					Path:     "/plugins/create.wasm",
					Manifest: "{}",
					SHA256:   "hash",
				})
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})
	})
})
