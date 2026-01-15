package persistence

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PluginRepository", func() {
	var repo model.PluginRepository

	Describe("Admin User", func() {
		BeforeEach(func() {
			ctx := GinkgoT().Context()
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			repo = NewPluginRepository(ctx, GetDBXBuilder())

			// Clean up any existing plugins
			all, _ := repo.GetAll()
			for _, p := range all {
				_ = repo.Delete(p.ID)
			}
		})

		AfterEach(func() {
			// Clean up after tests
			all, _ := repo.GetAll()
			for _, p := range all {
				_ = repo.Delete(p.ID)
			}
		})

		Describe("CountAll", func() {
			It("returns 0 when no plugins exist", func() {
				Expect(repo.CountAll()).To(Equal(int64(0)))
			})

			It("returns the number of plugins in the DB", func() {
				_ = repo.Put(&model.Plugin{ID: "test-plugin-1", Path: "/plugins/test1.wasm", Manifest: "{}", SHA256: "abc123"})
				_ = repo.Put(&model.Plugin{ID: "test-plugin-2", Path: "/plugins/test2.wasm", Manifest: "{}", SHA256: "def456"})

				Expect(repo.CountAll()).To(Equal(int64(2)))
			})
		})

		Describe("Delete", func() {
			It("deletes existing item", func() {
				plugin := &model.Plugin{ID: "to-delete", Path: "/plugins/delete.wasm", Manifest: "{}", SHA256: "hash"}
				_ = repo.Put(plugin)

				err := repo.Delete(plugin.ID)
				Expect(err).To(BeNil())

				_, err = repo.Get(plugin.ID)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("Get", func() {
			It("returns an existing item", func() {
				plugin := &model.Plugin{ID: "test-get", Path: "/plugins/test.wasm", Manifest: `{"name":"test"}`, SHA256: "hash123"}
				_ = repo.Put(plugin)

				res, err := repo.Get(plugin.ID)
				Expect(err).To(BeNil())
				Expect(res.ID).To(Equal(plugin.ID))
				Expect(res.Path).To(Equal(plugin.Path))
				Expect(res.Manifest).To(Equal(plugin.Manifest))
			})

			It("errors when missing", func() {
				_, err := repo.Get("notanid")
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				_ = repo.Put(&model.Plugin{ID: "plugin-a", Path: "/plugins/a.wasm", Manifest: "{}", SHA256: "hash1"})
				_ = repo.Put(&model.Plugin{ID: "plugin-b", Path: "/plugins/b.wasm", Manifest: "{}", SHA256: "hash2"})

				all, err := repo.GetAll()
				Expect(err).To(BeNil())
				Expect(all).To(HaveLen(2))
			})

			It("supports pagination", func() {
				_ = repo.Put(&model.Plugin{ID: "plugin-1", Path: "/plugins/1.wasm", Manifest: "{}", SHA256: "h1"})
				_ = repo.Put(&model.Plugin{ID: "plugin-2", Path: "/plugins/2.wasm", Manifest: "{}", SHA256: "h2"})
				_ = repo.Put(&model.Plugin{ID: "plugin-3", Path: "/plugins/3.wasm", Manifest: "{}", SHA256: "h3"})

				page1, err := repo.GetAll(model.QueryOptions{Max: 2, Offset: 0, Sort: "id"})
				Expect(err).To(BeNil())
				Expect(page1).To(HaveLen(2))

				page2, err := repo.GetAll(model.QueryOptions{Max: 2, Offset: 2, Sort: "id"})
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

				err := repo.Put(plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(plugin.ID)
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
				_ = repo.Put(plugin)

				plugin.Enabled = true
				plugin.Config = `{"new":"config"}`
				plugin.SHA256 = "updated"
				err := repo.Put(plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(plugin.ID)
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
				err := repo.Put(plugin)
				Expect(err).To(BeNil())

				saved, err := repo.Get(plugin.ID)
				Expect(err).To(BeNil())
				Expect(saved.LastError).To(Equal("failed to load: missing export"))
			})

			It("fails when ID is empty", func() {
				plugin := &model.Plugin{
					Path:     "/plugins/noid.wasm",
					Manifest: "{}",
					SHA256:   "hash",
				}
				err := repo.Put(plugin)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ID cannot be empty"))
			})
		})
	})

	Describe("Regular User", func() {
		BeforeEach(func() {
			ctx := GinkgoT().Context()
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: false})
			repo = NewPluginRepository(ctx, GetDBXBuilder())
		})

		Describe("CountAll", func() {
			It("fails to count items", func() {
				_, err := repo.CountAll()
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Delete", func() {
			It("fails to delete items", func() {
				err := repo.Delete("any-id")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Get", func() {
			It("fails to get items", func() {
				_, err := repo.Get("any-id")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("GetAll", func() {
			It("fails to get all items", func() {
				_, err := repo.GetAll()
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Put", func() {
			It("fails to create/update item", func() {
				err := repo.Put(&model.Plugin{
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
