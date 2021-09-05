package ffmpeg

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

type Parser struct{}

type parsedTags = map[string][]string

type ffprobeStream struct{
	Index				uint				`json:"index"`
	CodecName			string				`json:"codec_name"`
	CodecLongName		string				`json:"codec_long_name"`
	CodecType			string				`json:"codec_type"`
	CodecTagString		string				`json:"codec_tag_string"`
	CodecTag			string				`json:"codec_tag"`

	/* Audio-specific */
	SampleFmt			string				`json:"sample_fmt"`
	SampleRate			string				`json:"sample_rate"`
	Channels			uint				`json:"channels"`
	ChannelLayout		string				`json:"channel_layout"`
	BitsPerSample		uint				`json:"bits_per_sample"`

	/* Video-specific */
	Width				uint				`json:"width"`
	Height				uint				`json:"height"`
	CodedWidth			uint				`json:"coded_width"`
	CodedHeight			uint				`json:"coded_height"`
	ClosedCaptions		uint				`json:"closed_captions"`
	HasBFrames			uint				`json:"has_b_frames"`
	SampleAspectRatio	string				`json:"sample_aspect_ratio"`
	DisplayAspectRatio	string				`json:"display_aspect_ratio"`
	PixFmt				string				`json:"pix_fmt"`
	Level				int					`json:"level"`
	ColourRange			string				`json:"color_range"`
	Refs				uint				`json:"refs"`

	/* Common */
	BitRate				string				`json:"bit_rate"`
	RFrameRate			string				`json:"r_frame_rate"`
	AvgFrameRate		string				`json:"avg_Frame_rate"`
	TimeBase			string				`json:"time_base"`
	StartPTS			int64				`json:"start_pts"`
	StartTime			string				`json:"start_time"`
	DurationTS			uint				`json:"duration_ts"`
	Duration			string				`json:"duration"`
	BitsPerRawSample	string				`json:"bits_per_raw_sample"`

	Disposition struct {
		Default			uint	`json:"default"`
		Dub				uint	`json:"du"`
		Original		uint	`json:"original"`
		Comment			uint	`json:"comment"`
		Lyrics			uint	`json:"lyrics"`
		Karaoke			uint	`json:"karaoke"`
		Forced			uint	`json:"forced"`
		HearingImpared	uint	`json:"hearing_impaired"`
		VisualImpared	uint	`json:"visual_impaired"`
		CleanEffects	uint	`json:"clean_effects"`
		AttachedPic		uint	`json:"attached_pic"`
		TimedThumbnails	uint	`json:"timed_thumbnails"`
	} `json:"disposition"`
	Tags *map[string]string `json:"tags,omitempty"`
}

type ffprobeFormat struct {
	Filename		string				`json:"filename"`
	NumStreams		uint				`json:"nb_streams"`
	NumPrograms		uint				`json:"nb_programs"`
	FormatName		string				`json:"format_name"`
	FormatLongName	string				`json:"format_long_name"`
	StartTime		string				`json:"start_time"`
	Duration		string				`json:"duration"`
	Size			string				`json:"size"`
	BitRate			string				`json:"bit_rate"`
	ProbeScore		int 				`json:"probe_score"`
	Tags			map[string]string	`json:"tags"`
}

type ffprobePayload struct {
	Streams []ffprobeStream	`json:"streams"`
	Format ffprobeFormat	`json:"format"`
}

func (e *Parser) Parse(files ...string) (map[string]parsedTags, error) {
	fileTags := map[string]parsedTags{}

	for _, file := range files {
		args := e.createProbeCommand(file)

		cmd := exec.Command(args[0], args[1:]...) // #nosec
		output, err := cmd.Output()
		if err != nil {
			log.Warn("error running probe", err, "file", file, "command", args)
			continue
		}

		tags, err := e.extractMetadata(file, output)
		// Skip files with errors
		if err == nil {
			fileTags[file] = tags
		}
	}

	return fileTags, nil
}

func (e *Parser) extractMetadata(filePath string, info []byte) (parsedTags, error) {
	payload := ffprobePayload{}

	err := json.Unmarshal(info, &payload)
	if err != nil {
		log.Warn("unable to parse probe payload", err, "file", filePath)
		return nil, err
	}

	tags := map[string][]string{}

	var audioStream *ffprobeStream = nil

	// Pass 1: Ensure we have an audio stream
	for idx, stream := range payload.Streams {
		if stream.CodecType == "audio" {
			audioStream = &payload.Streams[idx]
			break
		}
	}

	if audioStream == nil {
		log.Warn("no audio streams, bailing...", "file", filePath)
		return nil, errors.New("no audio stream")
	}

	// Pass 2: Check for an attached picture, i.e. cover image
	for _, stream := range payload.Streams {
		if stream.Disposition.AttachedPic != 0 {
			tags["has_picture"] = []string{"true"}
		}
	}

	// Duration, given as a string, make sure it's actually a decimal
	duration := audioStream.Duration
	_, err = strconv.ParseFloat(duration, 64)
	if err != nil {
		duration = payload.Format.Duration
		_, err = strconv.ParseFloat(duration, 64)
	}

	if err == nil {
		tags["duration"] = []string{duration}
	}

	// Bit rate, given in b, convert to kb
	bitRate, err := strconv.ParseUint(audioStream.BitRate, 10, 64)
	if err != nil {
		bitRate, err = strconv.ParseUint(payload.Format.BitRate, 10, 64)
	}

	if err == nil {
		tags["bitrate"] = []string{strconv.FormatUint(bitRate / 1000, 10)}
	}

	// Add top-level tags first
	for tag, val := range payload.Format.Tags {
		ltag := strings.ToLower(tag)
		tags[ltag] = append(tags[ltag], val)
	}

	// Add stream tags. Not sure if this is correct behaviour with
	// multiple audio streams.
	for _, stream := range payload.Streams {
		if stream.CodecType != "audio" && stream.Tags == nil {
			continue
		}

		if stream.Tags == nil {
			continue
		}

		for tag, val := range *stream.Tags {
			ltag := strings.ToLower(tag)
			if ltag == "comment" && val == "Cover (front)" {
				continue
			}
			tags[ltag] = append(tags[ltag], val)
		}
	}

	alternativeTags := map[string][]string{
		"disc":        {"tpa"},
		"has_picture": {"metadata_block_picture"},
	}
	for tagName, alternatives := range alternativeTags {
		for _, altName := range alternatives {
			if altValue, ok := tags[altName]; ok {
				tags[tagName] = append(tags[tagName], altValue...)
			}
		}
	}
	return tags, nil
}

// Inputs will always be absolute paths
func (e *Parser) createProbeCommand(input string) []string {
	split := strings.Split(conf.Server.ProbeCommand, " ")
	args := make([]string, 0)

	for _, s := range split {
		if s == "%s" {
			args = append(args, "-i", input)
		} else {
			args = append(args, s)
		}
	}
	return args
}
