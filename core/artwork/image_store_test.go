package artwork

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageStore", func() {
	var store *ImageStore
	var root string
	var ctx context.Context

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		store = NewImageStore(root)
		ctx = context.Background()
	})

	It("hashes deterministically", func() {
		h1, err := hashImage(bytes.NewReader([]byte("some image bytes")))
		Expect(err).ToNot(HaveOccurred())
		h2, _ := hashImage(bytes.NewReader([]byte("some image bytes")))
		Expect(h1).To(Equal(h2))
		Expect(h1).To(HaveLen(16))
		h3, _ := hashImage(bytes.NewReader([]byte("other bytes")))
		Expect(h3).ToNot(Equal(h1))
	})

	It("writes sharded and reads back", func() {
		data := []byte("jpeg-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(filepath.Join(root, h[0:2], h[2:4], h+".jpg")).To(BeAnExistingFile())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()
		got, _ := io.ReadAll(rc)
		Expect(got).To(Equal(data))
	})

	It("is idempotent on duplicate writes and preserves the original content", func() {
		data := []byte("dup")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		// A duplicate write only touches mtime; passing different bytes under the same
		// hash proves the second reader is never consumed to overwrite the file.
		Expect(store.Write(h, "image/png", bytes.NewReader([]byte("not-dup")))).To(Succeed())

		rc, err := store.Open(h, "image/png")
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()
		got, err := io.ReadAll(rc)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(data))
	})

	It("refreshes the mtime on a duplicate write", func() {
		data := []byte("touch-me")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())

		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())

		info, err := os.Stat(store.path(h, "image/png"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info.ModTime()).To(BeTemporally(">", time.Now().Add(-time.Minute)))
	})

	It("rewrites the bytes when the existing file vanished before the liveness touch", func() {
		data := []byte("vanishing")
		h, _ := hashImage(bytes.NewReader(data))
		for range 10 {
			Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
			Expect(os.Remove(store.path(h, "image/png"))).To(Succeed())
			Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())

			rc, err := store.Open(h, "image/png")
			Expect(err).ToNot(HaveOccurred())
			got, _ := io.ReadAll(rc)
			rc.Close()
			Expect(got).To(Equal(data))
		}
	})

	It("returns fs.ErrNotExist for missing images", func() {
		_, err := store.Open("beefbeefbeefbeef", "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("rejects invalid hashes instead of panicking", func() {
		for _, h := range []string{"", "ab", "BEEFBEEFBEEFBEEF", "../../../../etcpw", "beefbeefbeefbee/"} {
			Expect(store.Write(h, "image/jpeg", bytes.NewReader([]byte("x")))).To(MatchError(ContainSubstring("invalid hash")))
			_, err := store.Open(h, "image/jpeg")
			Expect(err).To(MatchError(ContainSubstring("invalid hash")))
		}
	})

	It("sweeps unknown files, keeps known ones", func() {
		d1 := []byte("keep-me")
		h1, _ := hashImage(bytes.NewReader(d1))
		Expect(store.Write(h1, "image/jpeg", bytes.NewReader(d1))).To(Succeed())
		d2 := []byte("orphan")
		h2, _ := hashImage(bytes.NewReader(d2))
		Expect(store.Write(h2, "image/jpeg", bytes.NewReader(d2))).To(Succeed())

		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h2, "image/jpeg"), old, old)).To(Succeed())

		removed, err := store.Sweep(ctx, time.Now().Add(-time.Hour), func(h, _ string) bool { return h == h1 })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		_, err = store.Open(h2, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h1, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("sweeps a stale mime variant of a known hash, keeps the current one", func() {
		data := []byte("same-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())

		// The recorded mime is image/jpeg, so the .png variant is obsolete.
		removed, err := store.Sweep(ctx, time.Now().Add(-time.Hour), func(hash, ext string) bool {
			return hash == h && ext == ".jpg"
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		_, err = store.Open(h, "image/png")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("keeps young unknown files inside the grace window", func() {
		d := []byte("fresh-orphan")
		h, _ := hashImage(bytes.NewReader(d))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(d))).To(Succeed())

		removed, err := store.Sweep(ctx, time.Now().Add(-time.Hour), func(string, string) bool { return false })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(0))
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("removes abandoned temp files past the grace window, keeps fresh ones", func() {
		oldTmp := filepath.Join(root, ".old.tmp")
		Expect(os.WriteFile(oldTmp, []byte("x"), 0600)).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(oldTmp, old, old)).To(Succeed())

		freshTmp := filepath.Join(root, ".fresh.tmp")
		Expect(os.WriteFile(freshTmp, []byte("y"), 0600)).To(Succeed())

		removed, err := store.Sweep(ctx, time.Now().Add(-time.Hour), func(string, string) bool { return true })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		Expect(oldTmp).ToNot(BeAnExistingFile())
		Expect(freshTmp).To(BeAnExistingFile())
	})

	It("keeps sweeping past a file it cannot remove", func() {
		tests.SkipOnWindows("uses Unix file permission bits")
		if os.Geteuid() == 0 {
			Skip("read-only dir cannot block root (e.g. tests in a container)")
		}
		old := time.Now().Add(-2 * time.Hour)
		// "blocked" sorts before "ok", so the walk hits the unremovable file first.
		blockedDir := filepath.Join(root, "blocked")
		Expect(os.MkdirAll(blockedDir, 0755)).To(Succeed())
		blocked := filepath.Join(blockedDir, "a.jpg")
		Expect(os.WriteFile(blocked, []byte("x"), 0600)).To(Succeed())
		Expect(os.Chtimes(blocked, old, old)).To(Succeed())

		okDir := filepath.Join(root, "ok")
		Expect(os.MkdirAll(okDir, 0755)).To(Succeed())
		reachable := filepath.Join(okDir, "b.jpg")
		Expect(os.WriteFile(reachable, []byte("y"), 0600)).To(Succeed())
		Expect(os.Chtimes(reachable, old, old)).To(Succeed())

		Expect(os.Chmod(blockedDir, 0500)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(blockedDir, 0755) })

		removed, err := store.Sweep(ctx, time.Now().Add(-time.Hour), func(string, string) bool { return false })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		Expect(blocked).To(BeAnExistingFile())
		Expect(reachable).ToNot(BeAnExistingFile())
	})

	// Prune holds the worker's write lock for the whole sweep, and shutdown waits on the
	// worker, so an uncancellable walk over a large store stalls it until SIGKILL.
	It("abandons the walk when the context is cancelled", func() {
		old := time.Now().Add(-2 * time.Hour)
		for _, name := range []string{"a", "b", "c", "d"} {
			p := filepath.Join(root, name+".jpg")
			Expect(os.WriteFile(p, []byte("x"), 0600)).To(Succeed())
			Expect(os.Chtimes(p, old, old)).To(Succeed())
		}
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := store.Sweep(cancelCtx, time.Now().Add(-time.Hour), func(string, string) bool { return false })
		Expect(err).To(MatchError(context.Canceled))
		matches, _ := filepath.Glob(filepath.Join(root, "*.jpg"))
		Expect(matches).To(HaveLen(4), "a cancelled sweep must not keep deleting")
	})
})
