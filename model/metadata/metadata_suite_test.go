package metadata_test

import (
	"io/fs"
	"testing"
	"time"

	"github.com/djherbis/times"
	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetadata(t *testing.T) {
	tests.Init(t, true)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metadata Suite")
}

type testFileInfo struct {
	fs.FileInfo
}

func (t testFileInfo) BirthTime() time.Time {
	if ts := times.Get(t.FileInfo); ts.HasBirthTime() {
		return ts.BirthTime()
	}
	return t.FileInfo.ModTime()
}
