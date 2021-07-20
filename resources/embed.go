package resources

import (
	"embed"
	"io/ioutil"
)

//go:embed *
var FS embed.FS

func Asset(path string) ([]byte, error) {
	f, err := FS.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}
