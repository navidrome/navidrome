package scanner

import (
	"bufio"
	"errors"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
)

type Metadata struct {
	filePath string
	suffix   string
	fileInfo os.FileInfo
	tags     map[string]string
}

func (m *Metadata) Title() string       { return m.getTag("title", "sort_name") }
func (m *Metadata) Album() string       { return m.getTag("album", "sort_album") }
func (m *Metadata) Artist() string      { return m.getTag("artist", "sort_artist") }
func (m *Metadata) AlbumArtist() string { return m.getTag("album_artist", "albumartist") }
func (m *Metadata) SortTitle() string   { return m.getSortTag("", "title", "name") }
func (m *Metadata) SortAlbum() string   { return m.getSortTag("", "album") }
func (m *Metadata) SortArtist() string  { return m.getSortTag("", "artist") }
func (m *Metadata) SortAlbumArtist() string {
	return m.getSortTag("tso2", "albumartist", "album_artist")
}
func (m *Metadata) Composer() string            { return m.getTag("composer", "tcm", "sort_composer") }
func (m *Metadata) Genre() string               { return m.getTag("genre") }
func (m *Metadata) Year() int                   { return m.parseYear("date") }
func (m *Metadata) TrackNumber() (int, int)     { return m.parseTuple("track") }
func (m *Metadata) DiscNumber() (int, int)      { return m.parseTuple("tpa", "disc") }
func (m *Metadata) DiscSubtitle() string        { return m.getTag("tsst", "discsubtitle", "setsubtitle") }
func (m *Metadata) HasPicture() bool            { return m.getTag("has_picture", "metadata_block_picture") != "" }
func (m *Metadata) Comment() string             { return m.getTag("comment") }
func (m *Metadata) Compilation() bool           { return m.parseBool("compilation") }
func (m *Metadata) Duration() float32           { return m.parseDuration("duration") }
func (m *Metadata) BitRate() int                { return m.parseInt("bitrate") }
func (m *Metadata) ModificationTime() time.Time { return m.fileInfo.ModTime() }
func (m *Metadata) FilePath() string            { return m.filePath }
func (m *Metadata) Suffix() string              { return m.suffix }
func (m *Metadata) Size() int64                 { return m.fileInfo.Size() }

func LoadAllAudioFiles(dirPath string) (map[string]os.FileInfo, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	audioFiles := make(map[string]os.FileInfo)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		filePath := filepath.Join(dirPath, f.Name())
		extension := path.Ext(filePath)
		if !isAudioFile(extension) {
			continue
		}
		fi, err := os.Stat(filePath)
		if err != nil {
			log.Error("Could not stat file", "filePath", filePath, err)
		} else {
			audioFiles[filePath] = fi
		}
	}

	return audioFiles, nil
}

func ExtractAllMetadata(inputs []string) (map[string]*Metadata, error) {
	args := createProbeCommand(inputs)

	log.Trace("Executing command", "args", args)
	cmd := exec.Command(args[0], args[1:]...) // #nosec
	output, _ := cmd.CombinedOutput()
	mds := map[string]*Metadata{}
	if len(output) == 0 {
		return mds, errors.New("error extracting metadata files")
	}
	infos := parseOutput(string(output))
	for file, info := range infos {
		md, err := extractMetadata(file, info)
		// Skip files with errors
		if err == nil {
			mds[file] = md
		}
	}
	return mds, nil
}

var (
	// Input #0, mp3, from 'groovin.mp3':
	inputRegex = regexp.MustCompile(`(?m)^Input #\d+,.*,\sfrom\s'(.*)'`)

	//    TITLE           : Back In Black
	tagsRx = regexp.MustCompile(`(?i)^\s{4,6}([\w-]+)\s*:(.*)`)

	//  Duration: 00:04:16.00, start: 0.000000, bitrate: 995 kb/s`
	durationRx = regexp.MustCompile(`^\s\sDuration: ([\d.:]+).*bitrate: (\d+)`)

	//    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s
	bitRateRx = regexp.MustCompile(`^\s{4}Stream #\d+:\d+: (Audio):.*, (\d+) kb/s`)

	//    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 600x600 [SAR 1:1 DAR 1:1], 90k tbr, 90k tbn, 90k tbc`
	coverRx = regexp.MustCompile(`^\s{4}Stream #\d+:\d+: (Video):.*`)
)

func parseOutput(output string) map[string]string {
	split := map[string]string{}
	all := inputRegex.FindAllStringSubmatchIndex(output, -1)
	for i, loc := range all {
		// Filename is the first captured group
		file := output[loc[2]:loc[3]]

		// File info is everything from the match, up until the beginning of the next match
		info := ""
		initial := loc[1]
		if i < len(all)-1 {
			end := all[i+1][0] - 1
			info = output[initial:end]
		} else {
			// if this is the last match
			info = output[initial:]
		}
		split[file] = info
	}
	return split
}

func extractMetadata(filePath, info string) (*Metadata, error) {
	m := &Metadata{filePath: filePath, tags: map[string]string{}}
	m.suffix = strings.ToLower(strings.TrimPrefix(path.Ext(filePath), "."))
	var err error
	m.fileInfo, err = os.Stat(filePath)
	if err != nil {
		log.Warn("Error stating file. Skipping", "filePath", filePath, err)
		return nil, errors.New("error stating file")
	}

	m.parseInfo(info)
	if len(m.tags) == 0 {
		log.Trace("Not a media file. Skipping", "filePath", filePath)
		return nil, errors.New("not a media file")
	}
	return m, nil
}

func isAudioFile(extension string) bool {
	typ := mime.TypeByExtension(extension)
	return strings.HasPrefix(typ, "audio/")
}

func (m *Metadata) parseInfo(info string) {
	reader := strings.NewReader(info)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		match := tagsRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tagName := strings.ToLower(match[1])
			tagValue := strings.TrimSpace(match[2])

			// Skip when the tag was previously found
			if _, ok := m.tags[tagName]; !ok {
				m.tags[tagName] = tagValue
			}
			continue
		}

		match = coverRx.FindStringSubmatch(line)
		if len(match) > 0 {
			m.tags["has_picture"] = "true"
			continue
		}

		match = durationRx.FindStringSubmatch(line)
		if len(match) > 0 {
			m.tags["duration"] = match[1]
			if len(match) > 1 {
				m.tags["bitrate"] = match[2]
			}
			continue
		}

		match = bitRateRx.FindStringSubmatch(line)
		if len(match) > 0 {
			m.tags["bitrate"] = match[2]
		}
	}
}

func (m *Metadata) parseInt(tagName string) int {
	if v, ok := m.tags[tagName]; ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

var dateRegex = regexp.MustCompile(`^([12]\d\d\d)`)

func (m *Metadata) parseYear(tagName string) int {
	if v, ok := m.tags[tagName]; ok {
		match := dateRegex.FindStringSubmatch(v)
		if len(match) == 0 {
			log.Warn("Error parsing year from ffmpeg date field", "file", m.filePath, "date", v)
			return 0
		}
		year, _ := strconv.Atoi(match[1])
		return year
	}
	return 0
}

func (m *Metadata) getTag(tags ...string) string {
	for _, t := range tags {
		if v, ok := m.tags[t]; ok {
			return v
		}
	}
	return ""
}

func (m *Metadata) getSortTag(originalTag string, tags ...string) string {
	formats := []string{"sort%s", "sort_%s", "sort-%s", "%ssort", "%s_sort", "%s-sort"}
	all := []string{originalTag}
	for _, tag := range tags {
		for _, format := range formats {
			name := fmt.Sprintf(format, tag)
			all = append(all, name)
		}
	}
	return m.getTag(all...)
}

func (m *Metadata) parseTuple(tags ...string) (int, int) {
	for _, tagName := range tags {
		if v, ok := m.tags[tagName]; ok {
			tuple := strings.Split(v, "/")
			t1, t2 := 0, 0
			t1, _ = strconv.Atoi(tuple[0])
			if len(tuple) > 1 {
				t2, _ = strconv.Atoi(tuple[1])
			} else {
				t2, _ = strconv.Atoi(m.tags[tagName+"total"])
			}
			return t1, t2
		}
	}
	return 0, 0
}

func (m *Metadata) parseBool(tagName string) bool {
	if v, ok := m.tags[tagName]; ok {
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i == 1
	}
	return false
}

var zeroTime = time.Date(0000, time.January, 1, 0, 0, 0, 0, time.UTC)

func (m *Metadata) parseDuration(tagName string) float32 {
	if v, ok := m.tags[tagName]; ok {
		d, err := time.Parse("15:04:05", v)
		if err != nil {
			return 0
		}
		return float32(d.Sub(zeroTime).Seconds())
	}
	return 0
}

// Inputs will always be absolute paths
func createProbeCommand(inputs []string) []string {
	split := strings.Split(conf.Server.ProbeCommand, " ")
	args := make([]string, 0)

	for _, s := range split {
		if s == "%s" {
			for _, inp := range inputs {
				args = append(args, "-i", inp)
			}
		} else {
			args = append(args, s)
		}
	}
	return args
}
