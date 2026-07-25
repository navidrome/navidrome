package artwork

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("findImageInFolder", func() {
	var ctx context.Context
	var dir string

	BeforeEach(func() {
		ctx = context.Background()
		dir = GinkgoT().TempDir()
	})

	It("returns the first matching image", func() {
		Expect(os.WriteFile(filepath.Join(dir, "artist.jpg"), []byte("img"), 0o600)).To(Succeed())

		r, hit, err := findImageInFolder(ctx, os.DirFS(dir), ".", dir, "artist.*")
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
		Expect(hit).To(HaveSuffix("artist.jpg"))
	})

	// The glob matched, so the image exists; failing to open it says nothing about whether the
	// artist has one, and must not let the resolver settle on absent.
	It("reports a matched but unreadable image as unreadable, not as a miss", func() {
		img := filepath.Join(dir, "artist.jpg")
		Expect(os.WriteFile(img, []byte("img"), 0o600)).To(Succeed())
		Expect(os.Chmod(img, 0o000)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(img, 0o600) })

		_, _, err := findImageInFolder(ctx, os.DirFS(dir), ".", dir, "artist.*")
		Expect(err).To(MatchError(errSourceUnreadable))
	})

	It("reports a plain miss when nothing matches", func() {
		_, _, err := findImageInFolder(ctx, os.DirFS(dir), ".", dir, "artist.*")
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, errSourceUnreadable)).To(BeFalse(), "no match is definitive, not transient")
	})
})
