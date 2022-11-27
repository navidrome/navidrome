package resources

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils"
)

var (
	//go:embed *
	fsys embed.FS
)

func FS() fs.FS {
	return utils.MergeFS{
		Base:    fsys,
		Overlay: os.DirFS(path.Join(conf.Server.DataFolder, "resources")),
	}
}

func Asset(path string) ([]byte, error) {
	f, err := FS().Open(path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}
