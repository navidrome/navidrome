package cuesheet

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestFrame_Duration(t *testing.T) {
	tests := []struct {
		input  Frame
		output time.Duration
	}{
		{
			input:  Frame(0),
			output: time.Duration(0),
		},
		{
			input:  Frame(1),
			output: time.Millisecond * 13,
		},
		{
			input:  Frame(74),
			output: time.Millisecond * 986,
		},
		{
			input:  Frame(75),
			output: time.Second,
		},
		{
			input:  Frame(76),
			output: time.Second + time.Millisecond*13,
		},
	}

	for _, test := range tests {
		t.Run(test.input.String(), func(t *testing.T) {
			require.Equal(t, test.output, test.input.Duration())
		})
	}
}

func TestFrameFromStringToString(t *testing.T) {
	tests := []struct {
		input  string
		output Frame
		err    error
	}{
		{
			input:  "14:15:70",
			output: Frame(0xfac3),
		},
		{
			input: "0:0:0",
			err:   ErrorFrameFormat,
		},
		{
			input:  "00:00:00",
			output: Frame(0),
		},
		{
			input:  "00:00:10",
			output: Frame(10),
		},
		{
			input:  "00:01:74",
			output: Frame(0x95),
		},
		{
			input:  "155:01:01",
			output: Frame(0xaa4e8),
		},
		{
			input:  "02:59:01",
			output: Frame(0x3472),
		},
		{
			input:  "01:01:01",
			output: Frame(0x11e0),
		},
		{
			input:  "00:01:75",
			output: Frame(0),
			err:    ErrorFrameFormat,
		},
		{
			input:  "invalid",
			output: Frame(0),
			err:    ErrorFrameFormat,
		},
		{
			input:  "001:60:73",
			output: Frame(0),
			err:    ErrorFrameFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := frameFromString(test.input)
			if test.err != nil {
				require.EqualError(t, err, test.err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, test.output, result)
				require.Equal(t, test.input, result.String())
			}
		})
	}
}

func TestCueReader(t *testing.T) {
	tests := []struct {
		inputFile string
		check     func(output *Cuesheet)
		error     error
	}{
		{
			inputFile: "parser-case-insensitivity.cue",
			check: func(output *Cuesheet) {
				assert.NotEmpty(t, output.Performer)
			},
		},
		{
			inputFile: "parser-comments.cue",
			check: func(output *Cuesheet) {
				cueRem := output.Rem
				assert.NotEmpty(t, cueRem)
				assert.Contains(t, cueRem, "Foo")
				assert.Contains(t, cueRem, "Bar")
				assert.Contains(t, cueRem, "Yay!")
				trackRem := output.File[0].Tracks[0].Rem
				assert.NotEmpty(t, trackRem)
				assert.Contains(t, trackRem, "Doox")
				assert.Contains(t, trackRem, "Goox")
				assert.Contains(t, trackRem, "Loox")
			},
		},
		{
			inputFile: "parser-wrong-track-indent.cue",
			error:     ErrorExpectedTrackIndent,
		},
		{
			inputFile: "parser-duplicate-catalog.cue",
			error:     ErrorDuplicateCatalog,
		},
		{
			inputFile: "parser-duplicate-cdtextfile.cue",
			error:     ErrorDuplicateCdTextFile,
		},
		{
			inputFile: "parser-duplicate-performer.cue",
			error:     ErrorDuplicatePerformer,
		},
		{
			inputFile: "parser-duplicate-songwriter.cue",
			error:     ErrorDuplicateSongwriter,
		},
		{
			inputFile: "parser-duplicate-title.cue",
			error:     ErrorDuplicateTitle,
		},
		{
			inputFile: "parser-duplicate-track-flags.cue",
			error:     ErrorDuplicateTrackFlags,
		},
		{
			inputFile: "parser-duplicate-track-isrc.cue",
			error:     ErrorDuplicateTrackISRC,
		},
		{
			inputFile: "parser-duplicate-track-performer.cue",
			error:     ErrorDuplicateTrackPerformer,
		},
		{
			inputFile: "parser-duplicate-track-postgap.cue",
			error:     ErrorDuplicateTrackPostGap,
		},
		{
			inputFile: "parser-duplicate-track-pregap.cue",
			error:     ErrorDuplicateTrackPreGap,
		},
		{
			inputFile: "parser-duplicate-track-songwriter.cue",
			error:     ErrorDuplicateTrackSongwriter,
		},
		{
			inputFile: "parser-duplicate-track-title.cue",
			error:     ErrorDuplicateTrackTitle,
		},
		{
			inputFile: "parser-index-out-of-order.cue",
			error:     ErrorIndexOutOfOrder,
		},
		{
			inputFile: "parser-invalid-catalog.cue",
			error:     ErrorInvalidCatalog,
		},
		{
			inputFile: "parser-invalid-frames.cue",
			error:     ErrorFrameFormat,
		},
		{
			inputFile: "parser-invalid-isrc.cue",
			error:     ErrorInvalidISRC,
		},
		{
			inputFile: "parser-invalid-seconds.cue",
			error:     ErrorFrameFormat,
		},
		{
			inputFile: "parser-invalid-text.cue",
			error:     ErrorInvalidText,
		},
		{
			inputFile: "parser-missing-file.cue",
			error:     ErrorMissingFile,
		},
		{
			inputFile: "parser-missing-index.cue",
			error:     ErrorMissingIndex,
		},
		{
			inputFile: "parser-missing-track.cue",
			error:     ErrorMissingTrack,
		},
		{
			inputFile: "parser-track-out-of-order.cue",
			error:     ErrorTrackOutOfOrder,
		},
		{
			inputFile: "parser-track-out-of-order-2.cue",
			error:     ErrorTrackOutOfOrder,
		},
		{
			inputFile: "test.cue",
			check: func(output *Cuesheet) {
				assert.Equal(t, "Into The Otherworld", output.Title)
				assert.Equal(t, 1, len(output.File))
				assert.Equal(t, 11, len(output.File[0].Tracks))
			},
		},
		{
			inputFile: "test_1.cue",
			check: func(output *Cuesheet) {
				assert.Equal(t, "When All Is Said: The Best of Edge of Sanity", output.Title)
				assert.Equal(t, 1, len(output.File))
				assert.Equal(t, 99, len(output.File[0].Tracks))
			},
		},
		{
			inputFile: "invalid.binary",
			error:     ErrorParseCUE,
		},
	}

	for _, test := range tests {
		t.Run(test.inputFile, func(t *testing.T) {
			cue, err := ReadFromFile(filepath.Join("testdata", test.inputFile))
			if test.error != nil {
				assert.ErrorContains(t, err, test.error.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cue)
				assert.NotNil(t, test.check)
				test.check(cue)
			}
		})
	}
}
