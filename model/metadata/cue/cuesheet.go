package cue

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

const (
	delims          = "\t\n\r "
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
	ErrorUnclosedQuote            = fmt.Errorf("unclosed quote in string")
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
	catalogRegex = regexp.MustCompile(`^\d{12,14}$`)
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

func ReadFromFileFS(fs fs.FS, filePath string) (*Cuesheet, error) {
	file, err := fs.Open(filePath)
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

func readCUERem(cuesheet *Cuesheet, line string) error {
	if cuesheet.Rem == nil {
		cuesheet.Rem = RemData{}
	}
	key, err := readString(&line)
	if err != nil {
		return fmt.Errorf("REM command key: %w", err)
	}
	trimmedLine := strings.TrimLeft(line, delims)
	if len(trimmedLine) > 0 && isQuoted(trimmedLine) {
		value, closed := unquote(trimmedLine)
		if !closed {
			return ErrorUnclosedQuote
		}
		cuesheet.Rem[key] = value
	} else {
		cuesheet.Rem[key] = line
	}
	return nil
}

func readCUECatalog(cuesheet *Cuesheet, line string) error {
	if len(cuesheet.Catalog) > 0 {
		return ErrorDuplicateCatalog
	}
	if !catalogRegex.MatchString(line) {
		return ErrorInvalidCatalog
	}
	cuesheet.Catalog = line
	return nil
}

func readCUECdTextFile(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		log.Warn("Unclosed quote in CDTEXTFILE, using rest of line", "value", value)
	}
	return setTagOrError(&cuesheet.CdTextFile, value, ErrorDuplicateCdTextFile, false)
}

func readCUETitle(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		log.Warn("Unclosed quote in TITLE, using rest of line", "value", value)
	}
	err = setTagOrError(&cuesheet.Title, value, ErrorDuplicateTitle, true)
	if errors.Is(err, ErrorDuplicateTitle) {
		log.Error(fmt.Sprintf("already has title '%s' / '%s'", cuesheet.Title, line))
	}
	return err
}

func readCUEPerformer(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		log.Warn("Unclosed quote in PERFORMER, using rest of line", "value", value)
	}
	return setTagOrError(&cuesheet.Performer, value, ErrorDuplicatePerformer, true)
}

func readCUESongwriter(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		log.Warn("Unclosed quote in SONGWRITER, using rest of line", "value", value)
	}
	return setTagOrError(&cuesheet.SongWriter, value, ErrorDuplicateSongwriter, true)
}

func readCUEPreGap(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		return fmt.Errorf("PREGAP: %w", err)
	}
	cuesheet.Pregap, err = frameFromString(value)
	return err
}

func readCUEPostGap(cuesheet *Cuesheet, line string) error {
	value, err := readString(&line)
	if err != nil {
		return fmt.Errorf("POSTGAP: %w", err)
	}
	cuesheet.Postgap, err = frameFromString(value)
	return err
}

func readCUEFile(cuesheet *Cuesheet, line string) error {
	fileName, err := readString(&line)
	if err != nil {
		return fmt.Errorf("FILE command filename: %w", err)
	}
	fileType, err := readString(&line)
	if err != nil {
		return fmt.Errorf("FILE command type: %w", err)
	}
	cuesheet.File = append(cuesheet.File, File{FileName: fileName, FileType: fileType})
	return nil
}

func readCUEFields(cuesheet *Cuesheet, line string) error {
	command, err := readString(&line)
	if err != nil {
		return fmt.Errorf("reading command: %w", err)
	}
	command = strings.ToUpper(command)

	switch command {
	case "REM":
		return readCUERem(cuesheet, line)
	case "CATALOG":
		return readCUECatalog(cuesheet, line)
	case "CDTEXTFILE":
		return readCUECdTextFile(cuesheet, line)
	case "TITLE":
		return readCUETitle(cuesheet, line)
	case "PERFORMER":
		return readCUEPerformer(cuesheet, line)
	case "SONGWRITER":
		return readCUESongwriter(cuesheet, line)
	case "PREGAP":
		return readCUEPreGap(cuesheet, line)
	case "POSTGAP":
		return readCUEPostGap(cuesheet, line)
	case "FILE":
		return readCUEFile(cuesheet, line)
	default:
		return nil
	}
}

func readFileFields(file *File, line string) (bool, error) {
	command, err := readString(&line)
	if err != nil {
		return false, fmt.Errorf("reading command: %w", err)
	}
	command = strings.ToUpper(command)

	switch command {
	case "TRACK":
		track := Track{}
		track.TrackNumber, err = readUint(&line)
		if err != nil {
			return false, fmt.Errorf("TRACK number: %w", err)
		}
		if len(file.Tracks) != int(track.TrackNumber)-1 {
			return false, ErrorTrackOutOfOrder
		}
		track.TrackDataType, err = readString(&line)
		if err != nil {
			return false, fmt.Errorf("TRACK data type: %w", err)
		}
		file.Tracks = append(file.Tracks, track)
		return true, nil
	default:
	}
	return false, nil
}

func parseFlag(flag string) Flags {
	switch flag {
	case "DCP":
		return Dcp
	case "4CH":
		return FourCh
	case "PRE":
		return Pre
	case "SCMS":
		return Scms
	default:
		return None
	}
}

func readTrackFlags(track *Track, line string) error {
	if track.Flags != None {
		return ErrorDuplicateTrackFlags
	}
	track.Flags = None
	for len(line) > 0 {
		flag, err := readString(&line)
		if err != nil {
			log.Warn("Unclosed quote in FLAGS, using rest of line", "value", flag)
			if flag != "" {
				track.Flags |= parseFlag(flag)
			}
			break
		}
		track.Flags |= parseFlag(flag)
	}
	return nil
}

func readTrackISRC(track *Track, line string) error {
	if len(track.ISRC) > 0 {
		return ErrorDuplicateTrackISRC
	}
	if !isrcRegex.MatchString(line) {
		return ErrorInvalidISRC
	}
	track.ISRC = line
	return nil
}

func readTrackStringField(value *string, line string, duplicateErr error, fieldName string) error {
	v, closed := unquote(line)
	if !closed {
		log.Warn(fmt.Sprintf("Unclosed quote in track %s, using rest of line", fieldName), "value", v)
	}
	return setTagOrError(value, v, duplicateErr, true)
}

func readTrackPreGap(track *Track, line string) error {
	if track.PreGap > 0 {
		return ErrorDuplicateTrackPreGap
	}
	value, err := readString(&line)
	if err != nil {
		return fmt.Errorf("PREGAP: %w", err)
	}
	track.PreGap, err = frameFromString(value)
	return err
}

func readTrackPostGap(track *Track, line string) error {
	if track.PostGap > 0 {
		return ErrorDuplicateTrackPostGap
	}
	value, err := readString(&line)
	if err != nil {
		return fmt.Errorf("POSTGAP: %w", err)
	}
	track.PostGap, err = frameFromString(value)
	return err
}

func readTrackIndex(track *Track, line string) error {
	index := TrackIndex{}
	var err error
	index.Number, err = readUint(&line)
	if err != nil {
		return fmt.Errorf("INDEX number: %w", err)
	}
	value, err := readString(&line)
	if err != nil {
		return fmt.Errorf("INDEX frame: %w", err)
	}
	index.Frame, err = frameFromString(value)
	if err != nil {
		return err
	}
	if len(track.Index) == 0 && index.Number > 1 {
		return ErrorIndexOutOfOrder
	} else if len(track.Index) > 0 && track.Index[len(track.Index)-1].Number != index.Number-1 {
		return ErrorIndexOutOfOrder
	}
	track.Index = append(track.Index, index)
	return nil
}

func readTrackRem(track *Track, line string) error {
	if track.Rem == nil {
		track.Rem = RemData{}
	}
	key, err := readString(&line)
	if err != nil {
		return fmt.Errorf("REM command key: %w", err)
	}
	trimmedLine := strings.TrimLeft(line, delims)
	if len(trimmedLine) > 0 && isQuoted(trimmedLine) {
		value, closed := unquote(trimmedLine)
		if !closed {
			return ErrorUnclosedQuote
		}
		track.Rem[key] = value
	} else {
		track.Rem[key] = line
	}
	return nil
}

func readTrackFields(track *Track, line string) error {
	command, err := readString(&line)
	if err != nil {
		return fmt.Errorf("reading command: %w", err)
	}
	command = strings.ToUpper(command)

	switch command {
	case "FLAGS":
		return readTrackFlags(track, line)
	case "ISRC":
		return readTrackISRC(track, line)
	case "TITLE":
		return readTrackStringField(&track.Title, line, ErrorDuplicateTrackTitle, "TITLE")
	case "PERFORMER":
		return readTrackStringField(&track.Performer, line, ErrorDuplicateTrackPerformer, "PERFORMER")
	case "SONGWRITER":
		return readTrackStringField(&track.SongWriter, line, ErrorDuplicateTrackSongwriter, "SONGWRITER")
	case "PREGAP":
		return readTrackPreGap(track, line)
	case "POSTGAP":
		return readTrackPostGap(track, line)
	case "INDEX":
		return readTrackIndex(track, line)
	case "REM":
		return readTrackRem(track, line)
	default:
		return nil
	}
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

func readString(s *string) (string, error) {
	*s = strings.TrimLeft(*s, delims)

	if len(*s) > 0 && isQuoted(*s) {
		v, closed := unquote(*s)
		if !closed {
			return v, ErrorUnclosedQuote
		}
		*s = (*s)[len(v)+2:]
		return v, nil
	}
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == ' ' {
			v := (*s)[0:i]
			*s = (*s)[i+1:]
			return v, nil
		}
	}
	v := *s
	*s = ""
	return v, nil
}

func readUint(s *string) (uint, error) {
	v, err := readString(s)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}
	return uint(n), nil
}

func isQuoted(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] == '"' || s[0] == '\''
}

func unquote(s string) (string, bool) {
	if len(s) == 0 || !isQuoted(s) {
		return s, true
	}

	quote := s[0]
	i := 1
	for ; i < len(s); i++ {
		if s[i] == quote {
			return s[1:i], true
		}
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
	}
	return s[1:], false
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
