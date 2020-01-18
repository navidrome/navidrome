package scanner

import (
	"testing"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScanner(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}

var _ = XDescribe("TODO: REMOVE", func() {
	conf.Sonic.DbPath = "./testDB"
	log.SetLevel(log.LevelDebug)
	repos := Repositories{
		folder:    persistence.NewMediaFolderRepository(),
		mediaFile: persistence.NewMediaFileRepository(),
		album:     persistence.NewAlbumRepository(),
		artist:    persistence.NewArtistRepository(),
		playlist:  nil,
	}
	It("WORKS!", func() {
		t := NewTagScanner("/Users/deluan/Music/iTunes/iTunes Media/Music", repos)
		//t := NewTagScanner("/Users/deluan/Development/cloudsonic/sonic-server/tests/fixtures", repos)
		Expect(t.Scan(nil, time.Time{})).To(BeNil())
	})
})
