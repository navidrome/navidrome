package artwork

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CoverArtReader", func() {
	var fake *fakeArtwork
	var reader *CoverArtReader

	BeforeEach(func() {
		fake = &fakeArtwork{}
		reader = NewCoverArtReader(fake)
	})

	It("returns the image served for the artwork id", func() {
		fake.img = &Image{ReadCloser: io.NopCloser(strings.NewReader("image"))}
		artID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: "1"}

		r, err := reader.Read(context.Background(), artID, 500, false)
		Expect(err).ToNot(HaveOccurred())
		data, _ := io.ReadAll(r)
		Expect(string(data)).To(Equal("image"))
		Expect(fake.gotID).To(Equal(artID))
		Expect(fake.gotSize).To(Equal(500))
		Expect(fake.gotSquare).To(BeFalse())
	})

	It("reports an item without artwork as not found", func() {
		fake.err = ErrUnavailable

		_, err := reader.Read(context.Background(), model.ArtworkID{Kind: model.KindAlbumArtwork, ID: "1"}, 500, false)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("passes other errors through", func() {
		boom := errors.New("boom")
		fake.err = boom

		_, err := reader.Read(context.Background(), model.ArtworkID{Kind: model.KindAlbumArtwork, ID: "1"}, 500, false)
		Expect(err).To(MatchError(boom))
	})
})

type fakeArtwork struct {
	Artwork
	img       *Image
	err       error
	gotID     model.ArtworkID
	gotSize   int
	gotSquare bool
}

func (f *fakeArtwork) Get(_ context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	f.gotID, f.gotSize, f.gotSquare = artID, size, square
	return f.img, f.err
}
