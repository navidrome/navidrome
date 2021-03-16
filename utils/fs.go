package utils

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
)

type cachedFile struct {
	*os.File
	f fs.File
}

func NewCachedFile(fsys fs.FS, path string) (*cachedFile, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tmp, err := ioutil.TempFile(os.TempDir(), "navidrome-")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(tmp, f); err != nil {
		return nil, err
	}

	return &cachedFile{tmp, f}, err
}

func (fc *cachedFile) Close() error {
	err := fc.File.Close()
	if err != nil {
		return fmt.Errorf("closing tmpfile %q: %v", fc.File.Name(), err)
	}

	err = os.Remove(fc.File.Name())
	if err != nil {
		return fmt.Errorf("removing tmpfile %q: %v", fc.File.Name(), err)
	}

	return nil
}

var (
	_ io.ReadSeekCloser = (*cachedFile)(nil)
)
