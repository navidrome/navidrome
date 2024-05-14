package tag

import (
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/navidrome/navidrome/log"
)

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

func New(filePath string, fileInfo os.FileInfo, props Properties) Tags {
	var bTime time.Time
	if ts := times.Get(fileInfo); ts.HasBirthTime() {
		bTime = ts.BirthTime()
	} else {
		bTime = time.Now() // Maybe not the best option
	}

	return Tags{
		filePath:   filePath,
		fileInfo:   fileInfo,
		birthTime:  bTime,
		tags:       clean(props.Tags),
		audioProps: props.AudioProperties,
		hasPicture: props.HasPicture,
	}
}

type Tags struct {
	filePath   string
	fileInfo   os.FileInfo
	birthTime  time.Time
	tags       map[string][]string
	audioProps AudioProperties
	hasPicture bool
}

func (t Tags) FilePath() string                 { return t.filePath }
func (t Tags) ModTime() time.Time               { return t.fileInfo.ModTime() }
func (t Tags) BirthTime() time.Time             { return t.birthTime }
func (t Tags) Size() int64                      { return t.fileInfo.Size() }
func (t Tags) Suffix() string                   { return strings.ToLower(strings.TrimPrefix(path.Ext(t.filePath), ".")) }
func (t Tags) AudioProperties() AudioProperties { return t.audioProps }
func (t Tags) HasPicture() bool                 { return t.hasPicture }
func (t Tags) All() map[string][]string         { return t.tags }
func (t Tags) Strings(key Name) []string        { return t.tags[string(key)] }
func (t Tags) String(key Name) string           { return t.first(key) }
func (t Tags) Int(key Name) int64               { v, _ := strconv.Atoi(t.first(key)); return int64(v) }
func (t Tags) Float(key Name) float64           { v, _ := strconv.ParseFloat(t.first(key), 64); return v }
func (t Tags) Bool(key Name) bool               { v, _ := strconv.ParseBool(t.first(key)); return v }
func (t Tags) Date(key Name) Date               { return t.date(key) }
func (t Tags) NumAndTotal(key Name) (int, int)  { return t.tuple(key) }

func (t Tags) first(key Name) string {
	if v, ok := t.tags[string(key)]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// Used for tracks and discs
func (t Tags) tuple(key Name) (int, int) {
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

func (t Tags) date(key Name) Date {
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
	if len(tag) < 5 {
		return Date(match[1])
	}

	// then try YYYY-MM-DD
	if len(tag) > 10 {
		tag = tag[:10]
	}
	_, err := time.Parse("2006-01-02", tag)
	if err != nil {
		_, err = time.Parse("2006-01", tag)
		if err != nil {
			log.Warn("Error parsing "+key+" field for month + day", "file", t.filePath, "date", tag)
			return Date(match[1])
		}
	}
	return Date(tag)
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
