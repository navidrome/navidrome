package resources

import (
	"embed"
	"io"
)

//go:embed *
var FS embed.FS

func Asset(path string) ([]byte, error) {
	f, err := FS.Open(path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}
