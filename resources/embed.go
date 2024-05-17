package resources

import (
	"embed"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils/merge"
)

//go:embed *
var embedFS embed.FS

var FS = sync.OnceValue(func() fs.FS {
	return merge.FS{
		Base:    embedFS,
		Overlay: os.DirFS(path.Join(conf.Server.DataFolder, "resources")),
	}
})
