package cache_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djherbis/fscache"
	"github.com/navidrome/navidrome/utils/cache"
)

func TestFileHaunterMaxSize(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "spread_fs")
	cacheDir := filepath.Join(tempDir, "cache1")
	fs, err := fscache.NewFs(cacheDir, 0700)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	defer os.RemoveAll(tempDir)

	c, err := fscache.NewCacheWithHaunter(fs, fscache.NewLRUHaunterStrategy(cache.NewFileHaunter("", 0, 24, 400*time.Millisecond)))
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer c.Clean() //nolint:errcheck

	// Create 5 normal files and 1 empty
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("stream-%v", i)
		var r fscache.ReadAtCloser
		if i < 5 {
			r = createCachedStream(c, name, "hello")
		} else { // Last one is empty
			r = createCachedStream(c, name, "")
		}

		if !c.Exists(name) {
			t.Errorf(name + " should exist")
		}

		<-time.After(10 * time.Millisecond)

		err := r.Close()
		if err != nil {
			t.Error(err)
		}
	}

	<-time.After(400 * time.Millisecond)

	if c.Exists("stream-0") {
		t.Errorf("stream-0 should have been scrubbed")
	}

	if c.Exists("stream-5") {
		t.Errorf("stream-5 should have been scrubbed")
	}

	files, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if len(files) != 4 {
		t.Errorf("expected 4 items in directory")
	}
}

func createCachedStream(c *fscache.FSCache, name string, contents string) fscache.ReadAtCloser {
	r, w, _ := c.Get(name)
	_, _ = w.Write([]byte(contents))
	_ = w.Close()
	_, _ = io.Copy(io.Discard, r)
	return r
}
