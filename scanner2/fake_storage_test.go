//nolint:unused
package scanner2_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils/random"
	"github.com/stretchr/testify/assert"
)

type FakeStorage struct{ fs *FakeFS }

func RegisterFakeStorage(fs *FakeFS) {
	storage.Register("fake", func(url url.URL) storage.Storage { return &FakeStorage{fs: fs} })
}

func (s FakeStorage) FS() (storage.MusicFS, error) {
	return s.fs, nil
}

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

type _t = map[string]any

// This is to make sure our FakeFS implements the fs.FS interface.
func TestFakeFS(t *testing.T) {
	sgtPeppers := template(_t{"albumartist": "The Beatles", "album": "Sgt. Pepper's Lonely Hearts Club Band", "year": 1967})
	files := fstest.MapFS{
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/01 - Sgt. Pepper's Lonely Hearts Club Band.mp3": sgtPeppers(track(1, "Sgt. Pepper's Lonely Hearts Club Band")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/02 - With a Little Help from My Friends.mp3":    sgtPeppers(track(2, "With a Little Help from My Friends")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/03 - Lucy in the Sky with Diamonds.mp3":         sgtPeppers(track(3, "Lucy in the Sky with Diamonds")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3":                        sgtPeppers(track(4, "Getting Better")),
	}
	ffs := FakeFS{MapFS: files}

	assert.NoError(t, fstest.TestFS(ffs, "The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3"))
}

func modTime(ts string) _t   { return _t{fakeFileInfoModTime: ts} }
func birthTime(ts string) _t { return _t{fakeFileInfoBirthTime: ts} }

func template(t _t) func(..._t) *fstest.MapFile {
	return func(tags ..._t) *fstest.MapFile {
		return mp3(append([]_t{t}, tags...)...)
	}
}

func track(num int, title string) _t {
	t := audioProperties("mp3", 320)
	t["title"] = title
	t["track"] = num
	return t
}

func mp3(tags ..._t) *fstest.MapFile {
	ts := audioProperties("mp3", 320)
	if _, ok := ts[fakeFileInfoSize]; !ok {
		duration := ts["duration"].(int64)
		bitrate := ts["bitrate"].(int)
		ts[fakeFileInfoSize] = duration * int64(bitrate) / 8 * 1000
	}
	return file(append([]_t{ts}, tags...)...)
}

func file(tags ..._t) *fstest.MapFile {
	ts := _t{}
	for _, t := range tags {
		for k, v := range t {
			ts[k] = v
		}
	}
	if _, ok := ts[fakeFileInfoBirthTime]; !ok {
		ts[fakeFileInfoBirthTime] = time.Now().Format(time.RFC3339)
	}
	if _, ok := ts[fakeFileInfoModTime]; !ok {
		ts[fakeFileInfoModTime] = time.Now().Format(time.RFC3339)
	}
	if _, ok := ts[fakeFileInfoMode]; !ok {
		ts[fakeFileInfoMode] = fs.ModePerm
	}
	data, _ := json.Marshal(ts)
	if _, ok := ts[fakeFileInfoSize]; !ok {
		ts[fakeFileInfoSize] = int64(len(data))
	}
	return &fstest.MapFile{Data: data}
}

func audioProperties(suffix string, bitrate int) _t {
	duration := random.Int64N(300) + 120
	return _t{
		"suffix":     suffix,
		"bitrate":    bitrate,
		"duration":   duration,
		"samplerate": 44100,
		"bitdepth":   16,
		"channels":   2,
	}
}

func (ffs *FakeFS) ReadTags(paths ...string) (map[string]metadata.Info, error) {
	result := make(map[string]metadata.Info)
	for _, file := range paths {
		p, err := ffs.parseFile(file)
		if err != nil {
			return nil, err
		}
		result[file] = *p
	}
	return result, nil
}

func (ffs *FakeFS) parseFile(filePath string) (*metadata.Info, error) {
	contents, err := fs.ReadFile(ffs, filePath)
	if err != nil {
		return nil, err
	}
	data := map[string]any{}
	err = json.Unmarshal(contents, &data)
	if err != nil {
		return nil, err
	}
	p := metadata.Info{
		Tags:            map[string][]string{},
		AudioProperties: metadata.AudioProperties{},
		HasPicture:      data["has_picture"] == "true",
	}
	if d, ok := data["duration"].(float64); ok {
		p.AudioProperties.Duration = time.Duration(d) * time.Second
	}
	getInt := func(key string) int { v, _ := data[key].(float64); return int(v) }
	p.AudioProperties.BitRate = getInt("bitrate")
	p.AudioProperties.BitDepth = getInt("bitdepth")
	p.AudioProperties.SampleRate = getInt("samplerate")
	p.AudioProperties.Channels = getInt("channels")
	for k, v := range data {
		p.Tags[k] = []string{fmt.Sprintf("%v", v)}
	}
	p.FileInfo = ffs.parseFileInfo(filePath, data)
	return &p, nil
}

func (ffs *FakeFS) parseFileInfo(path string, tags map[string]any) metadata.FileInfo {
	return &fakeFileInfo{path: path, tags: tags}
}

const (
	fakeFileInfoMode      = "_mode"
	fakeFileInfoSize      = "_size"
	fakeFileInfoModTime   = "_modtime"
	fakeFileInfoBirthTime = "_birthtime"
)

type fakeFileInfo struct {
	path string
	tags map[string]any
}

func (ffi *fakeFileInfo) Name() string {
	name := path.Base(ffi.path)
	name = strings.TrimSuffix(name, path.Ext(name))
	return path.Base(name)
}
func (ffi *fakeFileInfo) Size() int64          { v, _ := ffi.tags[fakeFileInfoSize].(float64); return int64(v) }
func (ffi *fakeFileInfo) Mode() fs.FileMode    { return ffi.tags[fakeFileInfoMode].(fs.FileMode) }
func (ffi *fakeFileInfo) IsDir() bool          { return false }
func (ffi *fakeFileInfo) Sys() any             { return nil }
func (ffi *fakeFileInfo) ModTime() time.Time   { return ffi.parseTime(fakeFileInfoModTime) }
func (ffi *fakeFileInfo) BirthTime() time.Time { return ffi.parseTime(fakeFileInfoBirthTime) }
func (ffi *fakeFileInfo) parseTime(key string) time.Time {
	t, _ := time.Parse(time.RFC3339, ffi.tags[key].(string))
	return t
}
