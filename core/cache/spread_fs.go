package cache

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/djherbis/fscache"
	"github.com/karrick/godirwalk"
	"gopkg.in/djherbis/atime.v1"
	"gopkg.in/djherbis/stream.v1"
)

type spreadFs struct {
	root string
	mode os.FileMode
	init func() error
}

// NewSpreadFs returns a FileSystem rooted at directory dir. It
// Dir is created with perms if it doesn't exist.
func NewSpreadFs(dir string, mode os.FileMode) (fscache.FileSystem, error) {
	fs := &spreadFs{root: dir, mode: mode, init: func() error {
		return os.MkdirAll(dir, mode)
	}}
	return fs, fs.init()
}

func (fs *spreadFs) Reload(f func(key string, name string)) error {
	return godirwalk.Walk(fs.root, &godirwalk.Options{
		Callback: func(absoluteFilePath string, de *godirwalk.Dirent) error {
			path, err := filepath.Rel(fs.root, absoluteFilePath)
			if err != nil {
				return nil
			}

			parts := strings.Split(path, string(os.PathSeparator))
			if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 {
				return nil
			}

			key := filepath.Base(path)
			f(key, absoluteFilePath)
			return nil
		},
		Unsorted: true,
	})
}

func (fs *spreadFs) Create(name string) (stream.File, error) {
	key := fmt.Sprintf("%x", md5.Sum([]byte(name)))
	path := fmt.Sprintf("%s%c%s", key[0:2], os.PathSeparator, key[2:4])
	err := os.MkdirAll(filepath.Join(fs.root, path), fs.mode)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(fs.root, path, key), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

func (fs *spreadFs) Open(name string) (stream.File, error) {
	return os.Open(name)
}

func (fs *spreadFs) Remove(name string) error {
	return os.Remove(name)
}

func (fs *spreadFs) Stat(name string) (fscache.FileInfo, error) {
	stat, err := os.Stat(name)
	if err != nil {
		return fscache.FileInfo{}, err
	}
	return fscache.FileInfo{FileInfo: stat, Atime: atime.Get(stat)}, nil
}

func (fs *spreadFs) RemoveAll() error {
	if err := os.RemoveAll(fs.root); err != nil {
		return err
	}
	return fs.init()
}
