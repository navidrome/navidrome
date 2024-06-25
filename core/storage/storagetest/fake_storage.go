//nolint:unused
package storagetest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils/random"
)

// FakeStorage is a fake storage that provides a FakeFS.
// It is used for testing purposes.
type FakeStorage struct{ fs *FakeFS }

// Register registers the FakeStorage for the given scheme. To use it, set the model.Library's Path to "fake:///music",
// and register a FakeFS with schema = "fake". The storage registered will always return the same FakeFS instance.
func Register(schema string, fs *FakeFS) {
	storage.Register(schema, func(url url.URL) storage.Storage { return &FakeStorage{fs: fs} })
}

func (s FakeStorage) FS() (storage.MusicFS, error) {
	return s.fs, nil
}

// FakeFS is a fake filesystem that can be used for testing purposes.
// It implements the storage.MusicFS interface and keeps all files in memory, by using a fstest.MapFS internally.
// You must NOT add files directly in the MapFS property, but use SetFiles and its other methods instead.
// This is because the FakeFS keeps track of the latest modification time of directories, simulating the
// behavior of a real filesystem, and you should not bypass this logic.
type FakeFS struct {
	fstest.MapFS
}

func (ffs *FakeFS) SetFiles(files fstest.MapFS) {
	ffs.MapFS = files
	ffs.createDirTimestamps()
}

func (ffs *FakeFS) Add(filePath string, file *fstest.MapFile, when ...time.Time) {
	if len(when) == 0 {
		when = append(when, time.Now())
	}
	ffs.MapFS[filePath] = file
	ffs.touchContainingFolder(filePath, when[0])
	ffs.createDirTimestamps()
}

func (ffs *FakeFS) Remove(filePath string, when ...time.Time) *fstest.MapFile {
	filePath = path.Clean(filePath)
	if len(when) == 0 {
		when = append(when, time.Now())
	}
	if f, ok := ffs.MapFS[filePath]; ok {
		ffs.touchContainingFolder(filePath, when[0])
		delete(ffs.MapFS, filePath)
		return f
	}
	return nil
}

func (ffs *FakeFS) Move(srcPath string, destPath string, when ...time.Time) {
	if len(when) == 0 {
		when = append(when, time.Now())
	}
	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	ffs.MapFS[destPath] = ffs.MapFS[srcPath]
	ffs.touchContainingFolder(destPath, when[0])
	ffs.Remove(srcPath, when...)
}

// Touch sets the modification time of a file.
func (ffs *FakeFS) Touch(filePath string, when ...time.Time) {
	if len(when) == 0 {
		when = append(when, time.Now())
	}
	filePath = path.Clean(filePath)
	file, ok := ffs.MapFS[filePath]
	if ok {
		file.ModTime = when[0]
	} else {
		ffs.MapFS[filePath] = &fstest.MapFile{ModTime: when[0]}
	}
	ffs.touchContainingFolder(filePath, file.ModTime)
}

func (ffs *FakeFS) touchContainingFolder(filePath string, ts time.Time) {
	dir := path.Dir(filePath)
	dirFile, ok := ffs.MapFS[dir]
	if !ok {
		log.Fatal("Directory not found. Forgot to call SetFiles?", "file", filePath)
	}
	if dirFile.ModTime.Before(ts) {
		dirFile.ModTime = ts
	}
}

func (ffs *FakeFS) UpdateTags(filePath string, newTags map[string]any, when ...time.Time) {
	f, ok := ffs.MapFS[filePath]
	if !ok {
		panic(fmt.Errorf("file %s not found", filePath))
	}
	var tags map[string]any
	err := json.Unmarshal(f.Data, &tags)
	if err != nil {
		panic(err)
	}
	for k, v := range newTags {
		tags[k] = v
	}
	data, _ := json.Marshal(tags)
	f.Data = data
	ffs.Touch(filePath, when...)
}

// createDirTimestamps loops through all entries and create/updates directories entries in the map with the
// latest ModTime from any children of that directory.
func (ffs *FakeFS) createDirTimestamps() bool {
	var changed bool
	for filePath, file := range ffs.MapFS {
		dir := path.Dir(filePath)
		dirFile, ok := ffs.MapFS[dir]
		if !ok {
			dirFile = &fstest.MapFile{Mode: fs.ModeDir}
			ffs.MapFS[dir] = dirFile
		}
		if dirFile.ModTime.IsZero() {
			dirFile.ModTime = file.ModTime
			changed = true
		}
	}
	if changed {
		// If we updated any directory, we need to re-run the loop to create any parent directories
		ffs.createDirTimestamps()
	}
	return changed
}

func ModTime(ts string) map[string]any   { return map[string]any{fakeFileInfoModTime: ts} }
func BirthTime(ts string) map[string]any { return map[string]any{fakeFileInfoBirthTime: ts} }

func Template(t map[string]any) func(...map[string]any) *fstest.MapFile {
	return func(tags ...map[string]any) *fstest.MapFile {
		return MP3(append([]map[string]any{t}, tags...)...)
	}
}

func Track(num int, title string, tags ...map[string]any) map[string]any {
	ts := audioProperties("mp3", 320)
	ts["title"] = title
	ts["track"] = num
	for _, t := range tags {
		for k, v := range t {
			ts[k] = v
		}
	}
	return ts
}

func MP3(tags ...map[string]any) *fstest.MapFile {
	ts := audioProperties("mp3", 320)
	if _, ok := ts[fakeFileInfoSize]; !ok {
		duration := ts["duration"].(int64)
		bitrate := ts["bitrate"].(int)
		ts[fakeFileInfoSize] = duration * int64(bitrate) / 8 * 1000
	}
	return File(append([]map[string]any{ts}, tags...)...)
}

func File(tags ...map[string]any) *fstest.MapFile {
	ts := map[string]any{}
	for _, t := range tags {
		for k, v := range t {
			ts[k] = v
		}
	}
	modTime := time.Now()
	if mt, ok := ts[fakeFileInfoModTime]; !ok {
		ts[fakeFileInfoModTime] = time.Now().Format(time.RFC3339)
	} else {
		modTime, _ = time.Parse(time.RFC3339, mt.(string))
	}
	if _, ok := ts[fakeFileInfoBirthTime]; !ok {
		ts[fakeFileInfoBirthTime] = time.Now().Format(time.RFC3339)
	}
	if _, ok := ts[fakeFileInfoMode]; !ok {
		ts[fakeFileInfoMode] = fs.ModePerm
	}
	data, _ := json.Marshal(ts)
	if _, ok := ts[fakeFileInfoSize]; !ok {
		ts[fakeFileInfoSize] = int64(len(data))
	}
	return &fstest.MapFile{Data: data, ModTime: modTime, Mode: ts[fakeFileInfoMode].(fs.FileMode)}
}

func audioProperties(suffix string, bitrate int) map[string]any {
	duration := random.Int64N(300) + 120
	return map[string]any{
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
			log.Warn("Error reading metadata from file", "file", file, "err", err)
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
	file := ffs.MapFS[filePath]
	p.FileInfo = &fakeFileInfo{path: filePath, tags: data, file: file}
	return &p, nil
}

const (
	fakeFileInfoMode      = "_mode"
	fakeFileInfoSize      = "_size"
	fakeFileInfoModTime   = "_modtime"
	fakeFileInfoBirthTime = "_birthtime"
)

type fakeFileInfo struct {
	path string
	file *fstest.MapFile
	tags map[string]any
}

func (ffi *fakeFileInfo) Name() string         { return path.Base(ffi.path) }
func (ffi *fakeFileInfo) Size() int64          { v, _ := ffi.tags[fakeFileInfoSize].(float64); return int64(v) }
func (ffi *fakeFileInfo) Mode() fs.FileMode    { return ffi.file.Mode }
func (ffi *fakeFileInfo) IsDir() bool          { return false }
func (ffi *fakeFileInfo) Sys() any             { return nil }
func (ffi *fakeFileInfo) ModTime() time.Time   { return ffi.file.ModTime }
func (ffi *fakeFileInfo) BirthTime() time.Time { return ffi.parseTime(fakeFileInfoBirthTime) }
func (ffi *fakeFileInfo) parseTime(key string) time.Time {
	t, _ := time.Parse(time.RFC3339, ffi.tags[key].(string))
	return t
}
