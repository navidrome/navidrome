package scanner

import (
	"bufio"
	"bytes"
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

func ExtractMetadata(filePath string) (*Metadata, error) {
	m := &Metadata{filePath: filePath, tags: map[string]string{}}
	extension := path.Ext(filePath)
	if !isAudioFile(extension) {
		return nil, errors.New("not an audio file")
	}
	m.suffix = strings.ToLower(strings.TrimPrefix(extension, "."))
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	m.fileInfo = fi

	err = m.probe(filePath)
	if len(m.tags) == 0 {
		return nil, errors.New("not a media file")
	}
	return m, err
}

func isAudioFile(extension string) bool {
	typ := mime.TypeByExtension(extension)
	return strings.HasPrefix(typ, "audio/")
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
func (m *Metadata) Compilation() bool           { return m.parseBool("compilation") }
func (m *Metadata) Duration() int               { return m.parseDuration("duration") }
func (m *Metadata) BitRate() int                { return m.parseInt("bitrate") }
func (m *Metadata) ModificationTime() time.Time { return m.fileInfo.ModTime() }
func (m *Metadata) FilePath() string            { return m.filePath }
func (m *Metadata) Suffix() string              { return m.suffix }
func (m *Metadata) Size() int                   { return int(m.fileInfo.Size()) }

func (m *Metadata) probe(filePath string) error {
	cmdLine, args := createProbeCommand(filePath)

	log.Trace("Executing command", "cmdLine", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 || bytes.Contains(output, []byte("No such file or directory")) {
		return errors.New("error extracting metadata from " + filePath)
	}
	return m.parseOutput(output)
}

var (
	tagsRx = map[*regexp.Regexp]string{
		regexp.MustCompile(`^\s+compilation\s+:(.*)`):     "compilation",
		regexp.MustCompile(`^\s+genre\s+:\s(.*)`):         "genre",
		regexp.MustCompile(`^\s+title\s+:\s(.*)`):         "title",
		regexp.MustCompile(`^\s{4}comment\s+:\s(.*)`):     "comment",
		regexp.MustCompile(`^\s+artist\s+:\s(.*)`):        "artist",
		regexp.MustCompile(`^\s+album_artist\s+:\s(.*)`):  "album_artist",
		regexp.MustCompile(`^\s+TCM\s+:\s(.*)`):           "composer",
		regexp.MustCompile(`^\s+album\s+:\s(.*)`):         "album",
		regexp.MustCompile(`^\s+track\s+:\s(.*)`):         "trackNum",
		regexp.MustCompile(`^\s+disc\s+:\s(.*)`):          "discNum",
		regexp.MustCompile(`^\s+TPA\s+:\s(.*)`):           "discNum",
		regexp.MustCompile(`^\s+date\s+:\s(.*)`):          "year",
		regexp.MustCompile(`^\s{4}Stream #0:1: (.+)\:\s`): "hasPicture",
	}

	durationRx = regexp.MustCompile(`^\s\sDuration: ([\d.:]+).*bitrate: (\d+)`)
)

func (m *Metadata) parseOutput(output []byte) error {
	reader := strings.NewReader(string(output))
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
	return nil
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
		i, _ := strconv.Atoi(v)
		return i == 0
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

func createProbeCommand(filePath string) (string, []string) {
	cmd := conf.Sonic.ProbeCommand

	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.Replace(s, "%s", filePath, -1)
		split[i] = s
	}

	return split[0], split[1:]
}
