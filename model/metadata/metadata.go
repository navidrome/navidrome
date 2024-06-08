package metadata

import (
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

type Info struct {
	FileInfo        FileInfo
	Tags            map[string][]string
	AudioProperties AudioProperties
	HasPicture      bool
}

type FileInfo interface {
	fs.FileInfo
	BirthTime() time.Time
}

type AudioProperties struct {
	Duration   time.Duration
	BitRate    int
	BitDepth   int
	SampleRate int
	Channels   int
}

type Date string

func (d Date) Year() int {
	if d == "" {
		return 0
	}
	y, _ := strconv.Atoi(string(d[:4]))
	return y
}

func New(filePath string, info Info) Metadata {
	return Metadata{
		filePath:   filePath,
		fileInfo:   info.FileInfo,
		tags:       clean(info.Tags),
		audioProps: info.AudioProperties,
		hasPicture: info.HasPicture,
	}
}

type Metadata struct {
	filePath   string
	fileInfo   FileInfo
	tags       map[string][]string
	audioProps AudioProperties
	hasPicture bool
}

func (md Metadata) FilePath() string     { return md.filePath }
func (md Metadata) ModTime() time.Time   { return md.fileInfo.ModTime() }
func (md Metadata) BirthTime() time.Time { return md.fileInfo.BirthTime() }
func (md Metadata) Size() int64          { return md.fileInfo.Size() }
func (md Metadata) Suffix() string {
	return strings.ToLower(strings.TrimPrefix(path.Ext(md.filePath), "."))
}
func (md Metadata) AudioProperties() AudioProperties { return md.audioProps }
func (md Metadata) Length() float32                  { return float32(md.audioProps.Duration.Milliseconds()) / 1000 }
func (md Metadata) HasPicture() bool                 { return md.hasPicture }
func (md Metadata) All() map[string][]string         { return md.tags }
func (md Metadata) Strings(key TagName) []string     { return md.tags[string(key)] }
func (md Metadata) String(key TagName) string        { return md.first(key) }
func (md Metadata) Int(key TagName) int64            { v, _ := strconv.Atoi(md.first(key)); return int64(v) }
func (md Metadata) Float(key TagName) float64 {
	v, _ := strconv.ParseFloat(md.first(key), 64)
	return v
}
func (md Metadata) Bool(key TagName) bool              { v, _ := strconv.ParseBool(md.first(key)); return v }
func (md Metadata) Date(key TagName) Date              { return md.date(key) }
func (md Metadata) NumAndTotal(key TagName) (int, int) { return md.tuple(key) }

func (md Metadata) first(key TagName) string {
	if v, ok := md.tags[string(key)]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// Used for tracks and discs
func (md Metadata) tuple(key TagName) (int, int) {
	tag := md.first(key)
	if tag == "" {
		return 0, 0
	}
	tuple := strings.Split(tag, "/")
	t1, t2 := 0, 0
	t1, _ = strconv.Atoi(tuple[0])
	if len(tuple) > 1 {
		t2, _ = strconv.Atoi(tuple[1])
	} else {
		t2tag := md.first(key + "total")
		t2, _ = strconv.Atoi(t2tag)
	}
	return t1, t2
}

var dateRegex = regexp.MustCompile(`([12]\d\d\d)`)

// date tries to parse a date from a tag, it tries to get at least the year. See the tests for examples.
func (md Metadata) date(key TagName) Date {
	tag := md.first(key)
	if len(tag) < 4 {
		return ""
	}

	// first get just the year
	match := dateRegex.FindStringSubmatch(tag)
	if len(match) == 0 {
		log.Warn("Error parsing "+key+" field for year", "file", md.filePath, "date", tag)
		return ""
	}

	// if the tag is just the year, return it
	if len(tag) < 5 {
		return Date(match[1])
	}

	// if the tag is too long, truncate it
	tag = tag[:min(10, len(tag))]

	// then try to parse the full date
	for _, mask := range []string{"2006-01-02", "2006-01"} {
		_, err := time.Parse(mask, tag)
		if err == nil {
			return Date(tag)
		}
	}

	log.Warn("Error parsing "+key+" field for month + day", "file", md.filePath, "date", tag)
	return Date(match[1])
}

// clean filters out tags that are not in the mappings or are empty,
// combine equivalent tags and remove duplicated values.
// It keeps the order of the tags names as they are defined in the mappings.
func clean(tags map[string][]string) map[string][]string {
	lowered := map[string][]string{}
	for k, v := range tags {
		lowered[strings.ToLower(k)] = v
	}
	cleaned := map[string][]string{}
	for name, aliases := range mappings() {
		for _, k := range aliases {
			if v, ok := lowered[k]; ok {
				cleaned[name] = append(cleaned[name], v...)
			}
		}
	}
	for k, v := range cleaned {
		clean := removeDuplicatedAndEmpty(v)
		if len(clean) == 0 {
			delete(cleaned, k)
			continue
		}
		cleaned[k] = clean
	}
	return cleaned
}

func removeDuplicatedAndEmpty(values []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
