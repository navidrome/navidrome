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

var (
	//go:embed *
	embedFS embed.FS
	fsOnce  sync.Once
	fsys    fs.FS
)

func FS() fs.FS {
	fsOnce.Do(func() {
		fsys = merge.FS{
			Base:    embedFS,
			Overlay: os.DirFS(path.Join(conf.Server.DataFolder, "resources")),
		}
	})
	return fsys
}
