package plugins

import (
	"context"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("syncPlugins", func() {
	var m *Manager
	var repo *tests.MockPluginRepo
	var folder string

	BeforeEach(func() {
		folder = GinkgoT().TempDir()
		repo = tests.CreateMockPluginRepo()
		repo.SetData(model.Plugins{})
		m = &Manager{ds: &tests.MockDataStore{MockedPlugin: repo}}
	})

	writePackage := func(name string) {
		GinkgoHelper()
		manifest := &Manifest{Name: "Test Plugin", Author: "Test Author", Version: "1.0.0"}
		wasm := []byte{0x00, 0x61, 0x73, 0x6d} // Minimal wasm header
		Expect(createTestPackage(filepath.Join(folder, name), manifest, wasm)).To(Succeed())
	}

	It("registers a plugin with a usable ID", func() {
		writePackage("my-plugin" + PackageExtension)

		Expect(m.syncPlugins(context.Background(), folder)).To(Succeed())

		_, err := repo.Get("my-plugin")
		Expect(err).ToNot(HaveOccurred())
	})

	It("skips packages whose name yields a path-like ID", func() {
		writePackage("." + PackageExtension)
		writePackage(".." + PackageExtension)

		Expect(m.syncPlugins(context.Background(), folder)).To(Succeed())

		all, err := repo.GetAll()
		Expect(err).ToNot(HaveOccurred())
		Expect(all).To(BeEmpty())
	})
})

var _ = Describe("removePluginFromDB", func() {
	It("discards buffered scrobbles for the removed plugin", func() {
		ctx := context.Background()
		buffer := tests.CreateMockedScrobbleBufferRepo()
		Expect(buffer.Enqueue("my-plugin", "user1", "track1", time.Now(), "", "", "", "")).To(Succeed())
		Expect(buffer.Enqueue("other-plugin", "user1", "track2", time.Now(), "", "", "", "")).To(Succeed())

		repo := tests.CreateMockPluginRepo()
		plugin := model.Plugin{ID: "my-plugin", Enabled: false}
		repo.SetData(model.Plugins{plugin})

		// No broker: sendPluginRefreshEvent is nil-safe, and testBroker is
		// defined in manager_test.go, which is excluded on Windows.
		m := &Manager{
			ds: &tests.MockDataStore{MockedScrobbleBuffer: buffer},
		}
		Expect(m.removePluginFromDB(ctx, repo, &plugin)).To(Succeed())

		_, err := repo.Get("my-plugin")
		Expect(err).To(MatchError(model.ErrNotFound))

		remaining, err := buffer.Length()
		Expect(err).ToNot(HaveOccurred())
		Expect(remaining).To(Equal(int64(1)))
		entry, err := buffer.Next("other-plugin", "user1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entry).ToNot(BeNil(), "entries of other services must be kept")
	})

	It("keeps buffered scrobbles of a builtin scrobbler sharing the removed plugin's name", func() {
		ctx := context.Background()
		scrobbler.Register("builtin-svc", func(model.DataStore) scrobbler.Scrobbler { return nil })
		buffer := tests.CreateMockedScrobbleBufferRepo()
		Expect(buffer.Enqueue("builtin-svc", "user1", "track1", time.Now(), "", "", "", "")).To(Succeed())

		repo := tests.CreateMockPluginRepo()
		plugin := model.Plugin{ID: "builtin-svc", Enabled: false}
		repo.SetData(model.Plugins{plugin})

		m := &Manager{
			ds: &tests.MockDataStore{MockedScrobbleBuffer: buffer},
		}
		Expect(m.removePluginFromDB(ctx, repo, &plugin)).To(Succeed())

		remaining, err := buffer.Length()
		Expect(err).ToNot(HaveOccurred())
		Expect(remaining).To(Equal(int64(1)), "builtin scrobbler queue must not be wiped")
	})
})

var _ = Describe("addPluginToDB", func() {
	It("stamps the new row with CurrentManifestSchemaVersion", func() {
		ctx := context.Background()
		repo := tests.CreateMockPluginRepo()
		m := &Manager{}

		metadata := &PluginMetadata{
			Manifest: &Manifest{Name: "S", Author: "a", Version: "1.0.0"},
			SHA256:   "abc123",
		}
		Expect(m.addPluginToDB(ctx, repo, "my-plugin", "/plugins/my-plugin.ndp", metadata)).To(Succeed())

		stored, err := repo.Get("my-plugin")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.ManifestSchemaVersion).To(Equal(CurrentManifestSchemaVersion))
		Expect(stored.SHA256).To(Equal("abc123"))
	})
})

var _ = Describe("updatePluginInDB", func() {
	It("re-extracts and stamps the current schema version even when the file hash is unchanged", func() {
		// Regression test: a plugin scanned by an older build (before a field
		// like Actions existed on the Manifest struct) would otherwise never
		// have that field reach the DB, since the sync loop previously only
		// re-extracted on a file hash change.
		ctx := context.Background()
		repo := tests.CreateMockPluginRepo()
		existing := model.Plugin{
			ID:                    "my-plugin",
			SHA256:                "abc123",
			ManifestSchemaVersion: CurrentManifestSchemaVersion - 1,
			Enabled:               false,
		}
		repo.SetData(model.Plugins{existing})

		m := &Manager{}
		metadata := &PluginMetadata{
			Manifest: &Manifest{
				Name: "S", Author: "a", Version: "1.0.0",
				Actions: []Action{{Name: "testModel", Label: "Test Model"}},
			},
			SHA256: "abc123", // file itself did not change
		}
		Expect(m.updatePluginInDB(ctx, repo, &existing, "/plugins/my-plugin.ndp", metadata)).To(Succeed())

		stored, err := repo.Get("my-plugin")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.ManifestSchemaVersion).To(Equal(CurrentManifestSchemaVersion))
		Expect(stored.SHA256).To(Equal("abc123"))
		Expect(stored.Manifest).To(ContainSubstring("testModel"))
		Expect(stored.Enabled).To(BeFalse(), "was already disabled, re-extraction alone should not enable it")
	})

	It("disables and force-disables re-approval when the file itself actually changed", func() {
		ctx := context.Background()
		repo := tests.CreateMockPluginRepo()
		existing := model.Plugin{
			ID:                    "my-plugin",
			Path:                  "/plugins/my-plugin.ndp",
			SHA256:                "abc123",
			ManifestSchemaVersion: CurrentManifestSchemaVersion,
			Enabled:               true,
		}
		repo.SetData(model.Plugins{existing})

		m := &Manager{}
		metadata := &PluginMetadata{
			Manifest: &Manifest{Name: "S", Author: "a", Version: "1.0.0"},
			SHA256:   "def456", // file content actually changed
		}
		Expect(m.updatePluginInDB(ctx, repo, &existing, "/plugins/my-plugin.ndp", metadata)).To(Succeed())

		stored, err := repo.Get("my-plugin")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.SHA256).To(Equal("def456"))
		Expect(stored.Enabled).To(BeFalse(), "a real binary change always requires manual re-enable, regardless of prior state")
	})

	It("falls back to disabled with LastError set when a schema-only re-extraction can't actually reload the plugin", func() {
		// Was enabled, file unchanged, but the reload attempt itself fails
		// (here: because the plugin's package can't be found on disk) - must
		// not silently leave Enabled=true with nothing actually running,
		// which is the exact "enabled but dead" state this whole fix exists
		// to stop happening.
		ctx := context.Background()
		repo := tests.CreateMockPluginRepo()
		existing := model.Plugin{
			ID:                    "my-plugin",
			Path:                  "/nonexistent/my-plugin.ndp",
			SHA256:                "abc123",
			ManifestSchemaVersion: CurrentManifestSchemaVersion - 1,
			Enabled:               true,
		}
		repo.SetData(model.Plugins{existing})

		m := &Manager{}
		metadata := &PluginMetadata{
			Manifest: &Manifest{Name: "S", Author: "a", Version: "1.0.0"},
			SHA256:   "abc123", // file itself did not change
		}
		Expect(m.updatePluginInDB(ctx, repo, &existing, "/nonexistent/my-plugin.ndp", metadata)).To(Succeed())

		stored, err := repo.Get("my-plugin")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.ManifestSchemaVersion).To(Equal(CurrentManifestSchemaVersion))
		Expect(stored.Enabled).To(BeFalse())
		Expect(stored.LastError).ToNot(BeEmpty())
	})
})

var _ = Describe("ComputeFileSHA256", func() {
	It("returns a consistent 64-char lowercase hex hash for the same file", func() {
		dir := GinkgoT().TempDir()
		ndpPath := filepath.Join(dir, "test.ndp")
		err := createTestPackage(ndpPath, &Manifest{Name: "S", Author: "a", Version: "1.0.0"}, []byte{0x00, 0x61, 0x73, 0x6d})
		Expect(err).ToNot(HaveOccurred())

		hash1, err := ComputeFileSHA256(ndpPath)
		Expect(err).ToNot(HaveOccurred())
		hash2, err := ComputeFileSHA256(ndpPath)
		Expect(err).ToNot(HaveOccurred())

		Expect(hash1).To(Equal(hash2))
		Expect(hash1).To(MatchRegexp(`^[0-9a-f]{64}$`))
	})

	It("returns an error for a non-existent path", func() {
		_, err := ComputeFileSHA256(filepath.Join(GinkgoT().TempDir(), "does-not-exist.ndp"))
		Expect(err).To(HaveOccurred())
	})
})
