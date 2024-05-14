package scanner2_test

import (
	"encoding/json"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/utils/random"
	"github.com/stretchr/testify/assert"
)

type FakeFS struct {
	fstest.MapFS
}

// RmGlob removes all files that match the glob pattern.
func (ffs *FakeFS) RmGlob(glob string) {
	matches, err := fs.Glob(ffs, glob)
	if err != nil {
		panic(err)
	}
	for _, f := range matches {
		delete(ffs.MapFS, f)
	}
}

// Touch sets the modification time of a file.
func (ffs *FakeFS) Touch(path string, t ...time.Time) {
	if len(t) == 0 {
		t = append(t, time.Now())
	}
	f, ok := ffs.MapFS[path]
	if !ok {
		ffs.MapFS[path] = &fstest.MapFile{ModTime: t[0]}
		return
	}
	f.ModTime = t[0]
}

type tag map[string]any

func TestFakeFS(t *testing.T) {
	sgtPeppers := template(tag{"albumartist": "The Beatles", "album": "Sgt. Pepper's Lonely Hearts Club Band", "year": 1967})
	files := fstest.MapFS{
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/01 - Sgt. Pepper's Lonely Hearts Club Band.mp3": sgtPeppers(track(1, "Sgt. Pepper's Lonely Hearts Club Band")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/02 - With a Little Help from My Friends.mp3":    sgtPeppers(track(2, "With a Little Help from My Friends")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/03 - Lucy in the Sky with Diamonds.mp3":         sgtPeppers(track(3, "Lucy in the Sky with Diamonds")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3":                        sgtPeppers(track(4, "Getting Better")),
	}
	ffs := FakeFS{MapFS: files}

	assert.NoError(t, fstest.TestFS(ffs, "The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3"))
}

func template(t tag) func(...tag) *fstest.MapFile {
	return func(tags ...tag) *fstest.MapFile {
		return file(append([]tag{t}, tags...)...)
	}
}

func track(num int, title string) tag {
	t := audioProperties("mp3", 320)
	t["title"] = title
	t["track"] = num
	return t
}

func file(tags ...tag) *fstest.MapFile {
	ts := audioProperties("mp3", 320)
	for _, t := range tags {
		for k, v := range t {
			ts[k] = v
		}
	}
	data, _ := json.Marshal(ts)
	return &fstest.MapFile{Data: data, ModTime: time.Now()}
}

func audioProperties(suffix string, bitrate int64) tag {
	duration := random.Int64(300) + 120
	return tag{
		"suffix":     suffix,
		"bitrate":    bitrate,
		"duration":   duration,
		"size":       duration * bitrate / 8,
		"samplerate": 44100,
		"bitdepth":   16,
		"channels":   2,
	}
}
