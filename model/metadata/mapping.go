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

func (t Metadata) FilePath() string     { return t.filePath }
func (t Metadata) ModTime() time.Time   { return t.fileInfo.ModTime() }
func (t Metadata) BirthTime() time.Time { return t.fileInfo.BirthTime() }
func (t Metadata) Size() int64          { return t.fileInfo.Size() }
func (t Metadata) Suffix() string {
	return strings.ToLower(strings.TrimPrefix(path.Ext(t.filePath), "."))
}
func (t Metadata) AudioProperties() AudioProperties { return t.audioProps }
func (t Metadata) HasPicture() bool                 { return t.hasPicture }
func (t Metadata) All() map[string][]string         { return t.tags }
func (t Metadata) Strings(key Name) []string        { return t.tags[string(key)] }
func (t Metadata) String(key Name) string           { return t.first(key) }
func (t Metadata) Int(key Name) int64               { v, _ := strconv.Atoi(t.first(key)); return int64(v) }
func (t Metadata) Float(key Name) float64           { v, _ := strconv.ParseFloat(t.first(key), 64); return v }
func (t Metadata) Bool(key Name) bool               { v, _ := strconv.ParseBool(t.first(key)); return v }
func (t Metadata) Date(key Name) Date               { return t.date(key) }
func (t Metadata) NumAndTotal(key Name) (int, int)  { return t.tuple(key) }

func (t Metadata) first(key Name) string {
	if v, ok := t.tags[string(key)]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// Used for tracks and discs
func (t Metadata) tuple(key Name) (int, int) {
	tag := t.first(key)
	if tag == "" {
		return 0, 0
	}
	tuple := strings.Split(tag, "/")
	t1, t2 := 0, 0
	t1, _ = strconv.Atoi(tuple[0])
	if len(tuple) > 1 {
		t2, _ = strconv.Atoi(tuple[1])
	} else {
		t2tag := t.first(key + "total")
		t2, _ = strconv.Atoi(t2tag)
	}
	return t1, t2
}

var dateRegex = regexp.MustCompile(`([12]\d\d\d)`)

// date tries to parse a date from a tag, it tries to get at least the year. See the tests for examples.
func (t Metadata) date(key Name) Date {
	tag := t.first(key)
	if len(tag) < 4 {
		return ""
	}

	// first get just the year
	match := dateRegex.FindStringSubmatch(tag)
	if len(match) == 0 {
		log.Warn("Error parsing "+key+" field for year", "file", t.filePath, "date", tag)
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

	log.Warn("Error parsing "+key+" field for month + day", "file", t.filePath, "date", tag)
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
