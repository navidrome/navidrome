package artwork

import (
	"context"
	"errors"
	"io/fs"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// unreadableFS globs like its embedded MapFS but refuses to open anything. Injecting the error
// keeps this independent of the filesystem: os.Chmod does not restrict read access on Windows.
type unreadableFS struct{ fstest.MapFS }

func (u unreadableFS) Open(string) (fs.File, error) { return nil, fs.ErrPermission }

var _ = Describe("findImageInFolder", func() {
	var ctx context.Context
	var files fstest.MapFS

	BeforeEach(func() {
		ctx = context.Background()
		files = fstest.MapFS{"artist.jpg": &fstest.MapFile{Data: []byte("img")}}
	})

	It("returns the first matching image", func() {
		r, hit, err := findImageInFolder(ctx, files, ".", "/lib", "artist.*")
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
		Expect(hit).To(HaveSuffix("artist.jpg"))
	})

	// The glob matched, so the image exists; failing to open it says nothing about whether the
	// artist has one, and must not let the resolver settle on absent.
	It("reports a matched but unreadable image as unreadable, not as a miss", func() {
		_, _, err := findImageInFolder(ctx, unreadableFS{files}, ".", "/lib", "artist.*")
		Expect(err).To(MatchError(errSourceUnreadable))
	})

	It("reports a plain miss when nothing matches", func() {
		_, _, err := findImageInFolder(ctx, files, ".", "/lib", "nothing.*")
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, errSourceUnreadable)).To(BeFalse(), "no match is definitive, not transient")
	})
})
