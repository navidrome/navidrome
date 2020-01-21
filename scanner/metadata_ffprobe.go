//+build ignored

package scanner

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/dhowden/tag"
)

type Metadata struct {
	filePath    string
	suffix      string
	fileInfo    os.FileInfo
	t           tag.Metadata
	duration    int
	bitRate     int
	compilation bool
}

func ExtractMetadata(filePath string) (*Metadata, error) {
	m := &Metadata{filePath: filePath}
	m.suffix = strings.ToLower(strings.TrimPrefix(path.Ext(filePath), "."))
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	m.fileInfo = fi

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	m.t = t

	err = m.probe(filePath)
	return m, err
}

func (m *Metadata) Title() string               { return m.t.Title() }
func (m *Metadata) Album() string               { return m.t.Album() }
func (m *Metadata) Artist() string              { return m.t.Artist() }
func (m *Metadata) AlbumArtist() string         { return m.t.AlbumArtist() }
func (m *Metadata) Composer() string            { return m.t.Composer() }
func (m *Metadata) Genre() string               { return m.t.Genre() }
func (m *Metadata) Year() int                   { return m.t.Year() }
func (m *Metadata) TrackNumber() (int, int)     { return m.t.Track() }
func (m *Metadata) DiscNumber() (int, int)      { return m.t.Disc() }
func (m *Metadata) HasPicture() bool            { return m.t.Picture() != nil }
func (m *Metadata) Compilation() bool           { return m.compilation }
func (m *Metadata) Duration() int               { return m.duration }
func (m *Metadata) BitRate() int                { return m.bitRate }
func (m *Metadata) ModificationTime() time.Time { return m.fileInfo.ModTime() }
func (m *Metadata) FilePath() string            { return m.filePath }
func (m *Metadata) Suffix() string              { return m.suffix }
func (m *Metadata) Size() int                   { return int(m.fileInfo.Size()) }

// probe analyzes the file and returns duration in seconds and bitRate in kb/s.
// It uses the ffprobe external tool, configured in conf.Sonic.ProbeCommand
func (m *Metadata) probe(filePath string) error {
	cmdLine, args := createProbeCommand(filePath)

	log.Trace("Executing command", "cmdLine", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return m.parseOutput(output)
}

func (m *Metadata) parseInt(objItf interface{}, field string) (int, error) {
	obj := objItf.(map[string]interface{})
	s, ok := obj[field].(string)
	if !ok {
		return -1, errors.New("invalid ffprobe output field obj." + field)
	}
	fDuration, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1, err
	}
	return int(fDuration), nil
}

func (m *Metadata) parseOutput(output []byte) error {
	var data map[string]map[string]interface{}
	err := json.Unmarshal(output, &data)
	if err != nil {
		return err
	}

	format, ok := data["format"]
	if !ok {
		err = errors.New("invalid ffprobe output. no format found")
		return err
	}

	if tags, ok := format["tags"]; ok {
		c, _ := m.parseInt(tags, "compilation")
		m.compilation = c == 1
	}

	m.duration, err = m.parseInt(format, "duration")
	if err != nil {
		return err
	}

	m.bitRate, err = m.parseInt(format, "bit_rate")
	m.bitRate = m.bitRate / 1000
	if err != nil {
		return err
	}

	return nil
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
