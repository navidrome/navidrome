package scanner

import (
	"bufio"
	"errors"
	"mime"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
)

type Metadata struct {
	filePath string
	suffix   string
	fileInfo os.FileInfo
	tags     map[string]string
}

func (m *Metadata) Title() string               { return m.tags["title"] }
func (m *Metadata) Album() string               { return m.tags["album"] }
func (m *Metadata) Artist() string              { return m.tags["artist"] }
func (m *Metadata) AlbumArtist() string         { return m.tags["album_artist"] }
func (m *Metadata) Composer() string            { return m.tags["composer"] }
func (m *Metadata) Genre() string               { return m.tags["genre"] }
func (m *Metadata) Year() int                   { return m.parseInt("year") }
func (m *Metadata) TrackNumber() (int, int)     { return m.parseTuple("trackNum") }
func (m *Metadata) DiscNumber() (int, int)      { return m.parseTuple("discNum") }
func (m *Metadata) HasPicture() bool            { return m.tags["hasPicture"] == "Video" }
func (m *Metadata) Comment() string             { return m.tags["comment"] }
func (m *Metadata) Compilation() bool           { return m.parseBool("compilation") }
func (m *Metadata) Duration() int               { return m.parseDuration("duration") }
func (m *Metadata) BitRate() int                { return m.parseInt("bitrate") }
func (m *Metadata) ModificationTime() time.Time { return m.fileInfo.ModTime() }
func (m *Metadata) FilePath() string            { return m.filePath }
func (m *Metadata) Suffix() string              { return m.suffix }
func (m *Metadata) Size() int                   { return int(m.fileInfo.Size()) }

func ExtractAllMetadata(dirPath string) (map[string]*Metadata, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var audioFiles []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		filePath := path.Join(dirPath, f.Name())
		extension := path.Ext(filePath)
		if !isAudioFile(extension) {
			continue
		}
		audioFiles = append(audioFiles, filePath)
	}

	if len(audioFiles) == 0 {
		return map[string]*Metadata{}, nil
	}
	return probe(audioFiles)
}

func probe(inputs []string) (map[string]*Metadata, error) {
	cmdLine, args := createProbeCommand(inputs)

	log.Trace("Executing command", "cmdLine", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	output, _ := cmd.CombinedOutput()
	mds := map[string]*Metadata{}
	if len(output) == 0 {
		return mds, errors.New("error extracting metadata files")
	}
	infos := parseOutput(string(output))
	for file, info := range infos {
		md, err := extractMetadata(file, info)
		if err == nil {
			mds[file] = md
		}
	}
	return mds, nil
}

var inputRegex = regexp.MustCompile(`(?m)^Input #\d+,.*,\sfrom\s'(.*)'`)

func parseOutput(output string) map[string]string {
	split := map[string]string{}
	all := inputRegex.FindAllStringSubmatchIndex(string(output), -1)
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
	m.parseInfo(info)
	m.fileInfo, _ = os.Stat(filePath)
	if len(m.tags) == 0 {
		return nil, errors.New("not a media file")
	}
	return m, nil
}

func isAudioFile(extension string) bool {
	typ := mime.TypeByExtension(extension)
	return strings.HasPrefix(typ, "audio/")
}

var (
	tagsRx = map[*regexp.Regexp]string{
		regexp.MustCompile(`^\s{4}compilation\s+:(.*)`):    "compilation",
		regexp.MustCompile(`^\s{4}genre\s+:\s(.*)`):        "genre",
		regexp.MustCompile(`^\s{4}title\s+:\s(.*)`):        "title",
		regexp.MustCompile(`^\s{4}comment\s+:\s(.*)`):      "comment",
		regexp.MustCompile(`^\s{4}artist\s+:\s(.*)`):       "artist",
		regexp.MustCompile(`^\s{4}album_artist\s+:\s(.*)`): "album_artist",
		regexp.MustCompile(`^\s{4}TCM\s+:\s(.*)`):          "composer",
		regexp.MustCompile(`^\s{4}album\s+:\s(.*)`):        "album",
		regexp.MustCompile(`^\s{4}track\s+:\s(.*)`):        "trackNum",
		regexp.MustCompile(`^\s{4}disc\s+:\s(.*)`):         "discNum",
		regexp.MustCompile(`^\s{4}TPA\s+:\s(.*)`):          "discNum",
		regexp.MustCompile(`^\s{4}date\s+:\s(.*)`):         "year",
		regexp.MustCompile(`^\s{4}Stream #\d+:1: (.+):\s`): "hasPicture",
	}

	durationRx = regexp.MustCompile(`^\s\sDuration: ([\d.:]+).*bitrate: (\d+)`)
)

func (m *Metadata) parseInfo(info string) {
	reader := strings.NewReader(info)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		for rx, tag := range tagsRx {
			match := rx.FindStringSubmatch(line)
			if len(match) > 0 {
				m.tags[tag] = match[1]
				break
			}
			match = durationRx.FindStringSubmatch(line)
			if len(match) == 0 {
				continue
			}
			m.tags["duration"] = match[1]
			if len(match) > 1 {
				m.tags["bitrate"] = match[2]
			}
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

func (m *Metadata) parseTuple(tagName string) (int, int) {
	if v, ok := m.tags[tagName]; ok {
		tuple := strings.Split(v, "/")
		t1, t2 := 0, 0
		t1, _ = strconv.Atoi(tuple[0])
		if len(tuple) > 1 {
			t2, _ = strconv.Atoi(tuple[1])
		}
		return t1, t2
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

func (m *Metadata) parseDuration(tagName string) int {
	if v, ok := m.tags[tagName]; ok {
		d, err := time.Parse("15:04:05", v)
		if err != nil {
			return 0
		}
		return int(d.Sub(zeroTime).Seconds())
	}
	return 0
}

func createProbeCommand(inputs []string) (string, []string) {
	cmd := conf.Sonic.ProbeCommand

	split := strings.Split(cmd, " ")
	args := make([]string, 0)
	for _, s := range split {
		if s == "%s" {
			for _, inp := range inputs {
				args = append(args, "-i", inp)
			}
			continue
		}
		args = append(args, s)
	}

	return args[0], args[1:]
}
