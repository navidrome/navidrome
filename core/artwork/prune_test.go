package artwork

import (
	"bytes"
	"context"
	"errors"
	"os"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type flakyGetArtworkRepo struct {
	*tests.MockArtworkRepo
}

func (f *flakyGetArtworkRepo) GetAllHashes() ([]string, error) {
	return nil, errors.New("db locked")
}

var _ = Describe("Prune", func() {
	var ds *tests.MockDataStore
	var store *ImageStore
	var awRepo *tests.MockArtworkRepo

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		awRepo = ds.Artwork(context.Background()).(*tests.MockArtworkRepo)
		store = NewImageStore(GinkgoT().TempDir())
	})

	It("deletes orphan rows and their store files, keeps referenced ones", func() {
		data := []byte("orphan-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg",
			CreatedAt: time.Now().Add(-2 * time.Hour)})).To(Succeed())
		awRepo.OrphanHashes = []string{h}

		kept := []byte("kept-bytes")
		hk, _ := HashImage(bytes.NewReader(kept))
		Expect(store.Write(hk, "image/jpeg", bytes.NewReader(kept))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hk, Mime: "image/jpeg"})).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(hk, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("sweeps store files that have no artwork row", func() {
		stray := []byte("no-row-bytes")
		h, _ := HashImage(bytes.NewReader(stray))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(stray))).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("never sweeps files on a transient DB error", func() {
		ds.MockedArtwork = &flakyGetArtworkRepo{MockArtworkRepo: tests.CreateMockArtworkRepo()}

		data := []byte("live-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).ToNot(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})
})
