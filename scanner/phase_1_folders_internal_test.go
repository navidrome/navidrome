package scanner

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// stubFolderRepo satisfies just enough of FolderRepository for newScanJob to run
// without touching a real database.
type stubFolderRepo struct {
	model.FolderRepository
}

func (stubFolderRepo) GetFolderUpdateInfo(model.Library, ...string) (map[string]model.FolderUpdateInfo, error) {
	return map[string]model.FolderUpdateInfo{}, nil
}

var _ = Describe("newScanJob PID state", func() {
	It("carries the library's previously scanned album spec onto the job", func() {
		libPath, err := filepath.Abs(".")
		Expect(err).ToNot(HaveOccurred())
		ds := &tests.MockDataStore{MockedFolder: stubFolderRepo{}}
		lib := model.Library{ID: 1, Name: "Test", Path: libPath, ScannedPIDAlbum: "old_spec"}

		job, err := newScanJob(context.Background(), ds, lib, false, nil)

		Expect(err).ToNot(HaveOccurred())
		Expect(job.prevAlbumPIDConf).To(Equal("old_spec"))
	})
})
