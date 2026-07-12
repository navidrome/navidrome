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
