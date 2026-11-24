package plugins

import (
	"context"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("syncPlugins", func() {
	var ctx context.Context
	var m *Manager
	var repo model.PluginRepository
	var folder string

	BeforeEach(func() {
		ctx = adminContext(GinkgoT().Context())
		folder = GinkgoT().TempDir()
		ds := newTestDataStore()
		repo = ds.Plugin()
		m = &Manager{ds: ds}
	})

	writePackage := func(name string) {
		GinkgoHelper()
		manifest := &Manifest{Name: "Test Plugin", Author: "Test Author", Version: "1.0.0"}
		wasm := []byte{0x00, 0x61, 0x73, 0x6d} // Minimal wasm header
		Expect(createTestPackage(filepath.Join(folder, name), manifest, wasm)).To(Succeed())
	}

	It("registers a plugin with a usable ID", func() {
		writePackage("my-plugin" + PackageExtension)

		Expect(m.syncPlugins(ctx, folder)).To(Succeed())

		_, err := repo.Get(ctx, "my-plugin")
		Expect(err).ToNot(HaveOccurred())
	})

	It("skips packages whose name yields a path-like ID", func() {
		writePackage("." + PackageExtension)
		writePackage(".." + PackageExtension)

		Expect(m.syncPlugins(ctx, folder)).To(Succeed())

		all, err := repo.GetAll(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(all).To(BeEmpty())
	})
})

var _ = Describe("removePluginFromDB", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = adminContext(GinkgoT().Context())
		ds = newTestDataStore()
		putUsers(ds, model.User{ID: "user1", UserName: "user1"})
		for _, id := range []string{"track1", "track2"} {
			Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: id, Path: id + ".mp3", LibraryID: model.DefaultLibraryID})).To(Succeed())
		}
	})

	It("discards buffered scrobbles for the removed plugin", func() {
		buffer := ds.ScrobbleBuffer()
		Expect(buffer.Enqueue(ctx, "my-plugin", "user1", "track1", time.Now())).To(Succeed())
		Expect(buffer.Enqueue(ctx, "other-plugin", "user1", "track2", time.Now())).To(Succeed())

		plugin := model.Plugin{ID: "my-plugin", Enabled: false}
		repo := newTestDataStore(plugin).Plugin()

		// No broker: sendPluginRefreshEvent is nil-safe, and testBroker is
		// defined in manager_test.go, which is excluded on Windows.
		m := &Manager{ds: ds}
		Expect(m.removePluginFromDB(ctx, repo, &plugin)).To(Succeed())

		_, err := repo.Get(ctx, "my-plugin")
		Expect(err).To(MatchError(model.ErrNotFound))

		remaining, err := buffer.Length(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(remaining).To(Equal(int64(1)))
		entry, err := buffer.Next(ctx, "other-plugin", "user1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entry).ToNot(BeNil(), "entries of other services must be kept")
	})

	It("keeps buffered scrobbles of a builtin scrobbler sharing the removed plugin's name", func() {
		scrobbler.Register("builtin-svc", func(model.DataStore) scrobbler.Scrobbler { return nil })
		buffer := ds.ScrobbleBuffer()
		Expect(buffer.Enqueue(ctx, "builtin-svc", "user1", "track1", time.Now())).To(Succeed())

		plugin := model.Plugin{ID: "builtin-svc", Enabled: false}
		repo := newTestDataStore(plugin).Plugin()

		m := &Manager{ds: ds}
		Expect(m.removePluginFromDB(ctx, repo, &plugin)).To(Succeed())

		remaining, err := buffer.Length(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(remaining).To(Equal(int64(1)), "builtin scrobbler queue must not be wiped")
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
