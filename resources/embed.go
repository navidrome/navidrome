package resources

import (
	"embed"
	"io/fs"
	"io/ioutil"
)

//go:embed *
var filesystem embed.FS

func Asset(path string) ([]byte, error) {
	f, err := filesystem.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func Assets() fs.FS {
	return filesystem
}
