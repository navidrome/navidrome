package cuesheet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

const (
	delims          = "\t\n\r "
	eol             = "\n"
	framesPerSecond = 75 // is based on audio CD sectors and 44100 Hz sample rate
	remGenre        = "GENRE"
	remComment      = "COMMENT"
	remDate         = "DATE"
	remDiskID       = "DISCID"
	remRGAlbumGain  = "REPLAYGAIN_ALBUM_GAIN"
	remRGAlbumPeak  = "REPLAYGAIN_ALBUM_PEAK"
	remRGTrackGain  = "REPLAYGAIN_TRACK_GAIN"
	remRGTrackPeak  = "REPLAYGAIN_TRACK_PEAK"
	remDiscNumber   = "DISCNUMBER"
	remTotalDiscs   = "TOTALDISCS"
)

type ExpectedFields int

const (
	ExpectCommon ExpectedFields = iota
	ExpectTracks
	ExpectTrack
)

// Frame represent track frame
type Frame uint64

// Flags for track
type Flags int

// RemData data from REM lines
type RemData map[string]string

const (
	// None - no flags
	None   Flags = iota
	Dcp          = 1 << iota
	FourCh       = 1 << iota
	Pre          = 1 << iota
	Scms         = 1 << iota
)

var (
	ErrorParseCUE                 = fmt.Errorf("no CUE data")
	ErrorFrameFormat              = fmt.Errorf("invalid frame format")
	ErrorInvalidISRC              = fmt.Errorf("invalid ISRC")
	ErrorInvalidCatalog           = fmt.Errorf("invalid CATALOG")
	ErrorInvalidText              = fmt.Errorf("invalid text")
	ErrorMissingFile              = fmt.Errorf("no FILE")
	ErrorMissingIndex             = fmt.Errorf("no INDEX")
	ErrorMissingTrack             = fmt.Errorf("no TRACK")
	ErrorTrackOutOfOrder          = fmt.Errorf("track out of order")
	ErrorIndexOutOfOrder          = fmt.Errorf("index out of order")
	ErrorExpectedTrackIndent      = fmt.Errorf("expected track indent")
	ErrorDuplicateCatalog         = fmt.Errorf("duplicate CATALOG")
	ErrorDuplicateCdTextFile      = fmt.Errorf("duplicate CDTEXTFILE")
	ErrorDuplicatePerformer       = fmt.Errorf("duplicate PERFORMER")
	ErrorDuplicateTitle           = fmt.Errorf("duplicate TITLE")
	ErrorDuplicateSongwriter      = fmt.Errorf("duplicate SONGWRITER")
	ErrorDuplicateTrackPerformer  = fmt.Errorf("duplicate track PERFORMER")
	ErrorDuplicateTrackTitle      = fmt.Errorf("duplicate track TITLE")
	ErrorDuplicateTrackSongwriter = fmt.Errorf("duplicate track SONGWRITER")
	ErrorDuplicateTrackFlags      = fmt.Errorf("duplicate track FLAGS")
	ErrorDuplicateTrackISRC       = fmt.Errorf("duplicate track ISRC")
	ErrorDuplicateTrackPostGap    = fmt.Errorf("duplicate track POSTGAP")
	ErrorDuplicateTrackPreGap     = fmt.Errorf("duplicate track PREGAP")
)

var (
	catalogRegex = regexp.MustCompile(`^0\d{12}$`)
	isrcRegex    = regexp.MustCompile(`^[\da-zA-Z]{12}$`)
	cueRegex     = regexp.MustCompile(`^\S+( )+\S+.+`)
)

// TrackIndex data
type TrackIndex struct {
	Number uint
	Frame  Frame
}

func frameFromString(str string) (Frame, error) {
	v := strings.Split(str, ":")
	if len(str) < 8 {
		return 0, ErrorFrameFormat
	}
	if len(v) == 3 {
		mm, _ := strconv.ParseUint(v[0], 10, 32)
		ss, _ := strconv.ParseUint(v[1], 10, 32)
		if ss > 59 {
			return 0, ErrorFrameFormat
		}
		ff, _ := strconv.ParseUint(v[2], 10, 32)
		if ff >= framesPerSecond {
			return 0, ErrorFrameFormat
		}
		return Frame((mm*60+ss)*framesPerSecond + ff), nil
	}
	return 0, ErrorFrameFormat
}

func (frame Frame) Duration() time.Duration {
	seconds := time.Second * time.Duration(frame/framesPerSecond)
	milliseconds := time.Millisecond * time.Duration(float64(frame%framesPerSecond)/0.075)
	return seconds + milliseconds
}

func (frame Frame) String() string {
	seconds := frame / framesPerSecond
	minutes := seconds / 60
	seconds %= 60
	ff := frame % framesPerSecond
	return fmt.Sprintf("%.2d:%.2d:%.2d", minutes, seconds, ff)
}

// Track instance
type Track struct {
	Rem           RemData
	TrackNumber   uint
	TrackDataType string
	Flags         Flags
	ISRC          string
	Title         string
	Performer     string
	SongWriter    string
	PreGap        Frame
	PostGap       Frame
	Index         []TrackIndex
}

// File instance
type File struct {
	FileName string
	FileType string
	Tracks   []Track
}

// Cuesheet instance
type Cuesheet struct {
	Rem        RemData
	Catalog    string
	CdTextFile string
	Title      string
	Performer  string
	SongWriter string
	Pregap     Frame
	Postgap    Frame
	File       []File
}

func (track *Track) GetStartOffset() time.Duration {
	if len(track.Index) > 1 && track.Index[0].Number == 0 {
		return track.Index[1].Frame.Duration()
	}
	return track.Index[0].Frame.Duration()
}

// ReadFromFile load CUESheet from file
func ReadFromFile(filePath string) (*Cuesheet, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error("Can't close CUE file", "filePath", filePath)
		}
	}()
	return ReadCue(file)
}

func setTagOrError(value *string, newValue string, err error, noEmpty bool) error {
	if len(*value) > 0 {
		return err
	}
	*value = newValue
	if noEmpty && *value == "" {
		return ErrorInvalidText
	}
	return nil
}

func readCUEFields(cuesheet *Cuesheet, line string) error {
	command := strings.ToUpper(readString(&line))
	var err error
	switch command {
	case "REM":
		if cuesheet.Rem == nil {
			cuesheet.Rem = RemData{}
		}
		cuesheet.Rem[readString(&line)] = line
	case "CATALOG":
		if len(cuesheet.Catalog) > 0 {
			return ErrorDuplicateCatalog
		}
		if !catalogRegex.MatchString(line) {
			return ErrorInvalidCatalog
		}
		cuesheet.Catalog = line
	case "CDTEXTFILE":
		err = setTagOrError(&cuesheet.CdTextFile, readString(&line), ErrorDuplicateCdTextFile, false)
	case "TITLE":
		err = setTagOrError(&cuesheet.Title, readString(&line), ErrorDuplicateTitle, true)
		if errors.Is(err, ErrorDuplicateTitle) {
			log.Error(fmt.Sprintf("already has title '%s' / '%s'", cuesheet.Title, line))
		}
	case "PERFORMER":
		err = setTagOrError(&cuesheet.Performer, readString(&line), ErrorDuplicatePerformer, true)
	case "SONGWRITER":
		err = setTagOrError(&cuesheet.SongWriter, readString(&line), ErrorDuplicateSongwriter, true)
	case "PREGAP":
		cuesheet.Pregap, err = frameFromString(readString(&line))
		if err != nil {
			return err
		}
	case "POSTGAP":
		cuesheet.Postgap, err = frameFromString(readString(&line))
		if err != nil {
			return err
		}
	case "FILE":
		cuesheet.File = append(cuesheet.File,
			File{FileName: readString(&line), FileType: readString(&line)})
		return nil
	default:
	}

	return err
}

func readFileFields(file *File, line string) (bool, error) {
	command := strings.ToUpper(readString(&line))
	switch command {
	case "TRACK":
		track := Track{}
		track.TrackNumber = readUint(&line)
		if len(file.Tracks) != int(track.TrackNumber)-1 {
			return false, ErrorTrackOutOfOrder
		}
		track.TrackDataType = readString(&line)
		file.Tracks = append(file.Tracks, track)
		return true, nil
	default:
	}
	return false, nil
}

func readTrackFields(track *Track, line string) (err error) {
	command := strings.ToUpper(readString(&line))
	switch command {
	case "FLAGS":
		if track.Flags != None {
			return ErrorDuplicateTrackFlags
		}
		track.Flags = None
		for len(line) > 0 {
			switch readString(&line) {
			case "DCP":
				track.Flags |= Dcp
			case "4CH":
				track.Flags |= FourCh
			case "PRE":
				track.Flags |= Pre
			case "SCMS":
				track.Flags |= Scms
			default:
			}
		}
	case "ISRC":
		if len(track.ISRC) > 0 {
			err = ErrorDuplicateTrackISRC
			break
		}
		if !isrcRegex.MatchString(line) {
			err = ErrorInvalidISRC
			break
		}
		track.ISRC = line
	case "TITLE":
		err = setTagOrError(&track.Title, unquote(line), ErrorDuplicateTrackTitle, true)
	case "PERFORMER":
		err = setTagOrError(&track.Performer, unquote(line), ErrorDuplicateTrackPerformer, true)
	case "SONGWRITER":
		err = setTagOrError(&track.SongWriter, unquote(line), ErrorDuplicateTrackSongwriter, true)
	case "PREGAP":
		if track.PreGap > 0 {
			err = ErrorDuplicateTrackPreGap
			break
		}
		track.PreGap, err = frameFromString(readString(&line))
	case "POSTGAP":
		if track.PostGap > 0 {
			err = ErrorDuplicateTrackPostGap
			break
		}
		track.PostGap, err = frameFromString(readString(&line))
		if err != nil {
			break
		}
	case "INDEX":
		index := TrackIndex{}
		index.Number = readUint(&line)
		index.Frame, err = frameFromString(readString(&line))
		if err != nil {
			break
		}
		if len(track.Index) == 0 && index.Number > 1 {
			return ErrorIndexOutOfOrder
		} else if len(track.Index) > 0 && track.Index[len(track.Index)-1].Number != index.Number-1 {
			return ErrorIndexOutOfOrder
		}
		track.Index = append(track.Index, index)
	case "REM":
		if track.Rem == nil {
			track.Rem = RemData{}
		}
		track.Rem[readString(&line)] = line
	default:
	}

	return err
}

// ReadCue loads and parses CUESHEET from reader
func ReadCue(r io.Reader) (*Cuesheet, error) {
	s := bufio.NewScanner(r)
	cuesheet := &Cuesheet{}

	firstLine := true
	var fields ExpectedFields
	trackJustInserted := false

	for s.Scan() {
		line := s.Text()

		if firstLine {
			firstLine = false
			if !cueRegex.MatchString(line) {
				return nil, ErrorParseCUE
			}
		}

		if strings.HasPrefix(line, "    ") {
			line = line[4:]
			fields = ExpectTrack
			trackJustInserted = false
		} else if strings.HasPrefix(line, "  ") {
			line = line[2:]
			fields = ExpectTracks
			if trackJustInserted {
				return nil, ErrorExpectedTrackIndent
			}
		} else {
			fields = ExpectCommon
			if trackJustInserted {
				return nil, ErrorExpectedTrackIndent
			}
		}

		var err error
		switch fields {
		case ExpectCommon:
			err = readCUEFields(cuesheet, line)
		case ExpectTracks:
			if len(cuesheet.File) == 0 {
				return nil, ErrorMissingFile
			}
			trackJustInserted, err = readFileFields(&cuesheet.File[len(cuesheet.File)-1], line)
		case ExpectTrack:
			if len(cuesheet.File) == 0 {
				return nil, ErrorMissingFile
			}
			tracks := cuesheet.File[len(cuesheet.File)-1].Tracks
			if len(tracks) == 0 {
				return nil, ErrorMissingTrack
			}
			err = readTrackFields(&tracks[len(tracks)-1], line)
		}

		if err != nil {
			return nil, err
		}
	}

	if len(cuesheet.File) == 0 {
		return nil, ErrorMissingFile
	}

	for _, file := range cuesheet.File {
		if len(file.Tracks) == 0 {
			return nil, ErrorMissingTrack
		}
		for _, track := range file.Tracks {
			if len(track.Index) == 0 || (len(track.Index) == 1 && track.Index[0].Number == 0) {
				return nil, ErrorMissingIndex
			}
		}
	}

	return cuesheet, nil
}

// WriteCue writes CUESHEET to writer
//
//gocyclo:ignore
func WriteCue(w io.Writer, cuesheet *Cuesheet) error {
	ws := bufio.NewWriter(w)
	for k := range cuesheet.Rem {
		_, err := ws.WriteString("REM " + k + " " + cuesheet.Rem[k] + eol)
		if err != nil {
			return err
		}
	}

	if len(cuesheet.Catalog) > 0 {
		_, err := ws.WriteString("CATALOG " + cuesheet.Catalog + eol)
		if err != nil {
			return err
		}
	}

	if len(cuesheet.CdTextFile) > 0 {
		_, err := ws.WriteString("CDTEXTFILE " + formatString(cuesheet.CdTextFile) + eol)
		if err != nil {
			return err
		}
	}

	if len(cuesheet.Title) > 0 {
		_, err := ws.WriteString("TITLE " + formatString(cuesheet.Title) + eol)
		if err != nil {
			return err
		}
	}

	if len(cuesheet.Performer) > 0 {
		_, err := ws.WriteString("PERFORMER " + formatString(cuesheet.Performer) + eol)
		if err != nil {
			return err
		}
	}

	if len(cuesheet.SongWriter) > 0 {
		_, err := ws.WriteString("SONGWRITER " + formatString(cuesheet.SongWriter) + eol)
		if err != nil {
			return err
		}
	}

	if cuesheet.Pregap > 0 {
		_, err := ws.WriteString("PREGAP " + cuesheet.Pregap.String() + eol)
		if err != nil {
			return err
		}
	}

	if cuesheet.Postgap > 0 {
		_, err := ws.WriteString("POSTGAP " + cuesheet.Postgap.String() + eol)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(cuesheet.File); i++ {
		file := &cuesheet.File[i]
		_, err := ws.WriteString("FILE " + formatString(file.FileName) +
			" " + file.FileType + eol)
		if err != nil {
			return err
		}

		for i := 0; i < len(file.Tracks); i++ {
			track := &file.Tracks[i]

			_, err := ws.WriteString("  TRACK " + formatTrackNumber(track.TrackNumber) +
				" " + track.TrackDataType + eol)
			if err != nil {
				return err
			}

			if track.Flags != None {
				_, err := ws.WriteString("    FLAGS")
				if err != nil {
					return err
				}
				if (track.Flags & Dcp) != 0 {
					_, err := ws.WriteString(" DCP")
					if err != nil {
						return err
					}
				}
				if (track.Flags & FourCh) != 0 {
					_, err := ws.WriteString(" 4CH")
					if err != nil {
						return err
					}
				}
				if (track.Flags & Pre) != 0 {
					_, err := ws.WriteString(" PRE")
					if err != nil {
						return err
					}
				}
				if (track.Flags & Scms) != 0 {
					_, err := ws.WriteString(" SCMS")
					if err != nil {
						return err
					}
				}
				if _, err := ws.WriteString(eol); err != nil {
					return err
				}
			}

			if len(track.ISRC) > 0 {
				_, err := ws.WriteString("    ISRC " + track.ISRC + eol)
				if err != nil {
					return err
				}
			}

			if len(track.Title) > 0 {
				_, err := ws.WriteString("    TITLE " + formatString(track.Title) + eol)
				if err != nil {
					return err
				}
			}

			if len(track.Performer) > 0 {
				_, err := ws.WriteString("    PERFORMER " + formatString(track.Performer) + eol)
				if err != nil {
					return err
				}
			}

			if len(track.SongWriter) > 0 {
				_, err := ws.WriteString("    SONGWRITER " + formatString(track.SongWriter) + eol)
				if err != nil {
					return err
				}
			}

			if track.PreGap > 0 {
				_, err := ws.WriteString("    PREGAP " + track.PreGap.String() + eol)
				if err != nil {
					return err
				}
			}

			if track.PostGap > 0 {
				_, err := ws.WriteString("    POSTGAP " + track.PostGap.String() + eol)
				if err != nil {
					return err
				}
			}

			if track.Rem != nil {
				for k := range track.Rem {
					_, err := ws.WriteString("    REM " + k + " " + track.Rem[k] + eol)
					if err != nil {
						return err
					}
				}
			}

			for i := 0; i < len(track.Index); i++ {
				index := &track.Index[i]
				_, err := ws.WriteString("    INDEX " + formatTrackNumber(index.Number) +
					" " + index.Frame.String() + eol)
				if err != nil {
					return err
				}
			}
		}
	}

	return ws.Flush()
}

func readString(s *string) string {
	*s = strings.TrimLeft(*s, delims)

	if len(*s) > 0 && isQuoted(*s) {
		v := unquote(*s)
		*s = (*s)[len(v)+2:]
		return v
	}
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == ' ' {
			v := (*s)[0:i]
			*s = (*s)[i+1:]
			return v
		}
	}
	v := *s
	*s = ""
	return v
}

func readUint(s *string) uint {
	v := readString(s)
	if n, err := strconv.ParseUint(v, 10, 32); err == nil {
		return uint(n)
	}
	return 0
}

func formatString(s string) string {
	return quote(s, '"')
}

func formatTrackNumber(n uint) string {
	return leftPad(strconv.FormatUint(uint64(n), 10), "0", 2)
}

func isQuoted(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] == '"' || s[0] == '\''
}

func quote(s string, quote byte) string {
	buf := make([]byte, 0, 3*len(s)/2)
	buf = append(buf, quote)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == quote || c == '\\' {
			buf = append(buf, '\\')
			buf = append(buf, c)
		} else {
			buf = append(buf, c)
		}
	}
	buf = append(buf, quote)
	return string(buf)
}

func unquote(s string) string {
	quote := s[0]
	i := 1
	for ; i < len(s); i++ {
		if s[i] == quote {
			break
		}
		if s[i] == '\\' {
			i++
		}
	}
	return s[1:i]
}

func leftPad(s, padStr string, overallLen int) string {
	padCountInt := 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = strings.Repeat(padStr, padCountInt) + s
	return retStr[(len(retStr) - overallLen):]
}

func (rem *RemData) DiscNumber() int {
	s, ok := (*rem)[remDiscNumber]
	if ok {
		result, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return result
	}
	return 0
}

func (rem *RemData) TotalDiscs() int {
	s, ok := (*rem)[remTotalDiscs]
	if ok {
		result, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return result
	}
	return 0
}

// Genre returns genre from data
func (rem *RemData) Genre() string {
	s, ok := (*rem)[remGenre]
	if ok {
		return s
	}
	return ""
}

// Comment returns comment field
func (rem *RemData) Comment() string {
	s, ok := (*rem)[remComment]
	if ok {
		return s
	}
	return ""
}

// DiskID returns disk id from data
func (rem *RemData) DiskID() string {
	s, ok := (*rem)[remDiskID]
	if ok {
		return s
	}
	return ""
}

// Date returns release year
func (rem *RemData) Date() string {
	s, ok := (*rem)[remDate]
	if ok {
		return s
	}
	return ""
}

// AlbumGain returns album replay gain value
func (rem *RemData) AlbumGain() string {
	s, ok := (*rem)[remRGAlbumGain]
	if ok {
		return s
	}
	return ""
}

// AlbumPeak returns album replay gain peak value
func (rem *RemData) AlbumPeak() string {
	s, ok := (*rem)[remRGAlbumPeak]
	if ok {
		return s
	}
	return ""
}

// TrackGain returns track replay gain value
func (rem *RemData) TrackGain() string {
	s, ok := (*rem)[remRGTrackGain]
	if ok {
		return s
	}
	return ""
}

// TrackPeak returns track replay gain peak value
func (rem *RemData) TrackPeak() string {
	s, ok := (*rem)[remRGTrackPeak]
	if ok {
		return s
	}
	return ""
}
