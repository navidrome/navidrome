package scanner2

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/model/tag"
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

type _tag map[string]any

func TestFakeFS(t *testing.T) {
	sgtPeppers := template(_tag{"albumartist": "The Beatles", "album": "Sgt. Pepper's Lonely Hearts Club Band", "year": 1967})
	files := fstest.MapFS{
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/01 - Sgt. Pepper's Lonely Hearts Club Band.mp3": sgtPeppers(track(1, "Sgt. Pepper's Lonely Hearts Club Band")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/02 - With a Little Help from My Friends.mp3":    sgtPeppers(track(2, "With a Little Help from My Friends")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/03 - Lucy in the Sky with Diamonds.mp3":         sgtPeppers(track(3, "Lucy in the Sky with Diamonds")),
		"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3":                        sgtPeppers(track(4, "Getting Better")),
	}
	ffs := FakeFS{MapFS: files}

	assert.NoError(t, fstest.TestFS(ffs, "The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3"))
}

func template(t _tag) func(..._tag) *fstest.MapFile {
	return func(tags ..._tag) *fstest.MapFile {
		return file(append([]_tag{t}, tags...)...)
	}
}

func track(num int, title string) _tag {
	t := audioProperties("mp3", 320)
	t["title"] = title
	t["track"] = num
	return t
}

func file(tags ..._tag) *fstest.MapFile {
	ts := audioProperties("mp3", 320)
	for _, t := range tags {
		for k, v := range t {
			ts[k] = v
		}
	}
	data, _ := json.Marshal(ts)
	return &fstest.MapFile{Data: data, ModTime: time.Now()}
}

func audioProperties(suffix string, bitrate int64) _tag {
	duration := random.Int64(300) + 120
	return _tag{
		"suffix":     suffix,
		"bitrate":    bitrate,
		"duration":   duration,
		"size":       duration * bitrate / 8,
		"samplerate": 44100,
		"bitdepth":   16,
		"channels":   2,
	}
}

func RegisterFakeExtractor(fakeFS fs.FS) {
	tag.RegisterExtractor("fake", func(fs.FS, string) tag.Extractor {
		return fakeExtractor{fs: fakeFS}
	})
}

type fakeExtractor struct{ fs fs.FS }

func (e fakeExtractor) Parse(files ...string) (map[string]tag.Properties, error) {
	result := make(map[string]tag.Properties)
	for _, file := range files {
		p, err := e.parseFile(file)
		if err != nil {
			return nil, err
		}
		result[file] = *p
	}
	return result, nil
}

func (e fakeExtractor) parseFile(filePath string) (*tag.Properties, error) {
	contents, err := fs.ReadFile(e.fs, filePath)
	if err != nil {
		return nil, err
	}
	data := map[string]any{}
	err = json.Unmarshal(contents, &data)
	if err != nil {
		return nil, err
	}
	p := tag.Properties{
		Tags:            map[string][]string{},
		AudioProperties: tag.AudioProperties{},
		HasPicture:      data["has_picture"] == "true",
	}
	if d, ok := data["duration"].(int64); ok {
		p.AudioProperties.Duration = time.Duration(d) * time.Second
	}
	p.AudioProperties.BitRate, _ = data["bitrate"].(int)
	p.AudioProperties.BitDepth, _ = data["bitdepth"].(int)
	p.AudioProperties.SampleRate, _ = data["samplerate"].(int)
	p.AudioProperties.Channels, _ = data["channels"].(int)
	for k, v := range data {
		p.Tags[k] = []string{fmt.Sprintf("%v", v)}
	}
	return &p, nil
}

func (e fakeExtractor) Version() string {
	return "0.0.0"
}

var _ tag.Extractor = (*fakeExtractor)(nil)
