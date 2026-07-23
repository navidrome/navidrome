package artwork

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

func TestArtwork(t *testing.T) {
	// Runs unconditionally: the two leaks below are pre-existing and out of this
	// package's control, so they're ignored by exact top-function instead.
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2"),
		// notify's own init() starts a singleton tree the moment it's imported (via
		// core/storage/local or plugins); recursive on darwin, nonrecursive on linux.
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).internal"),
	)

	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artwork Suite")
}

// osDirFS wraps os.DirFS as a storage.MusicFS for integration tests.
// ReadTags is not exercised by these tests, so it is left as a stub.
type osDirFS struct{ fs.FS }

func (o osDirFS) ReadTags(...string) (map[string]metadata.Info, error) { return nil, nil }

// testFileScheme is the URL scheme registered to expose a tempdir as a
// storage.MusicFS for artwork integration tests.
const testFileScheme = "testfile"

// testFileLibPath builds a `testfile://` library URL for the given absolute
// filesystem path. On Windows, the native path (e.g. `C:\foo`) has no leading
// slash after ToSlash, which makes url.Parse treat the drive letter as a
// host. We prepend a `/` so parsing yields `u.Path == /C:/foo`, and the
// registered constructor below strips that leading slash back off.
func testFileLibPath(absPath string) string {
	p := filepath.ToSlash(absPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return testFileScheme + "://" + p
}

func init() {
	// Register the testfile storage scheme (os.DirFS-backed MusicFS). Used by
	// integration tests that need real files but not the taglib extractor.
	storage.Register(testFileScheme, func(u url.URL) storage.Storage {
		root := u.Path
		// Undo the leading slash added by testFileLibPath on Windows so that
		// os.Stat / os.DirFS receive a native path like `C:\foo`.
		if runtime.GOOS == "windows" && len(root) >= 3 && root[0] == '/' && root[2] == ':' {
			root = root[1:]
		}
		return &osDirStorage{root: filepath.FromSlash(root)}
	})
}

type osDirStorage struct{ root string }

func (s *osDirStorage) FS() (storage.MusicFS, error) {
	if _, err := os.Stat(s.root); err != nil {
		return nil, err
	}
	return osDirFS{os.DirFS(s.root)}, nil
}
