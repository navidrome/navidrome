package resources

import (
	"embed"
	"io/fs"
	"os"
	"path"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils/merge"
)

//go:embed *
var embedFS embed.FS

func FS() fs.FS {
	return merge.FS{
		Base:    embedFS,
		Overlay: os.DirFS(path.Join(conf.Server.DataFolder, "resources")),
	}
}
