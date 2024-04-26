package ffmpeg

import (
	"bufio"
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/scanner/metadata"
)

const ExtractorID = "ffmpeg"

type Extractor struct {
	ffmpeg ffmpeg.FFmpeg
}

func (e *Extractor) Parse(files ...string) (map[string]metadata.ParsedTags, error) {
	output, err := e.ffmpeg.Probe(context.TODO(), files)
	if err != nil {
		log.Error("Cannot use ffmpeg to extract tags. Aborting", err)
		return nil, err
	}
	fileTags := map[string]metadata.ParsedTags{}
	if len(output) == 0 {
		return fileTags, errors.New("error extracting metadata files")
	}
	infos := e.parseOutput(output)
	for file, info := range infos {
		tags, err := e.extractMetadata(file, info)
		// Skip files with errors
		if err == nil {
			fileTags[file] = tags
		}
	}
	return fileTags, nil
}

func (e *Extractor) CustomMappings() metadata.ParsedTags {
	return metadata.ParsedTags{
		"disc":         {"tpa"},
		"has_picture":  {"metadata_block_picture"},
		"originaldate": {"tdor"},
	}
}

func (e *Extractor) Version() string {
	return e.ffmpeg.Version()
}

func (e *Extractor) extractMetadata(filePath, info string) (metadata.ParsedTags, error) {
	tags := e.parseInfo(info)
	if len(tags) == 0 {
		log.Trace("Not a media file. Skipping", "filePath", filePath)
		return nil, errors.New("not a media file")
	}

	return tags, nil
}

var (
	// Input #0, mp3, from 'groovin.mp3':
	inputRegex = regexp.MustCompile(`(?m)^Input #\d+,.*,\sfrom\s'(.*)'`)

	//    TITLE           : Back In Black
	tagsRx = regexp.MustCompile(`(?i)^\s{4,6}([\w\s-]+)\s*:(.*)`)

	//                    : Second comment line
	continuationRx = regexp.MustCompile(`(?i)^\s+:(.*)`)

	//  Duration: 00:04:16.00, start: 0.000000, bitrate: 995 kb/s`
	durationRx = regexp.MustCompile(`^\s\sDuration: ([\d.:]+).*bitrate: (\d+)`)

	//    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s
	bitRateRx = regexp.MustCompile(`^\s{2,4}Stream #\d+:\d+: Audio:.*, (\d+) kb/s`)

	//    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 192 kb/s
	//    Stream #0:0: Audio: flac, 44100 Hz, stereo, s16
	audioStreamRx = regexp.MustCompile(`^\s{2,4}Stream #\d+:\d+.*: Audio: (.*), (.* Hz), ([\w.]+),*(.*.,)*`)

	//    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 600x600 [SAR 1:1 DAR 1:1], 90k tbr, 90k tbn, 90k tbc`
	coverRx = regexp.MustCompile(`^\s{2,4}Stream #\d+:.+: (Video):.*`)
)

func (e *Extractor) parseOutput(output string) map[string]string {
	outputs := map[string]string{}
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
		outputs[file] = info
	}
	return outputs
}

func (e *Extractor) parseInfo(info string) map[string][]string {
	tags := map[string][]string{}

	reader := strings.NewReader(info)
	scanner := bufio.NewScanner(reader)
	lastTag := ""
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		match := tagsRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tagName := strings.TrimSpace(strings.ToLower(match[1]))
			if tagName != "" {
				tagValue := strings.TrimSpace(match[2])
				tags[tagName] = append(tags[tagName], tagValue)
				lastTag = tagName
				continue
			}
		}

		if lastTag != "" {
			match = continuationRx.FindStringSubmatch(line)
			if len(match) > 0 {
				if tags[lastTag] == nil {
					tags[lastTag] = []string{""}
				}
				tagValue := tags[lastTag][0]
				tags[lastTag][0] = tagValue + "\n" + strings.TrimSpace(match[1])
				continue
			}
		}

		lastTag = ""
		match = coverRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tags["has_picture"] = []string{"true"}
			continue
		}

		match = durationRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tags["duration"] = []string{e.parseDuration(match[1])}
			if len(match) > 1 {
				tags["bitrate"] = []string{match[2]}
			}
			continue
		}

		match = bitRateRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tags["bitrate"] = []string{match[1]}
		}

		match = audioStreamRx.FindStringSubmatch(line)
		if len(match) > 0 {
			tags["channels"] = []string{e.parseChannels(match[3])}
		}
	}

	comment := tags["comment"]
	if len(comment) > 0 && comment[0] == "Cover (front)" {
		delete(tags, "comment")
	}

	return tags
}

var zeroTime = time.Date(0000, time.January, 1, 0, 0, 0, 0, time.UTC)

func (e *Extractor) parseDuration(tag string) string {
	d, err := time.Parse("15:04:05", tag)
	if err != nil {
		return "0"
	}
	return strconv.FormatFloat(d.Sub(zeroTime).Seconds(), 'f', 2, 32)
}

func (e *Extractor) parseChannels(tag string) string {
	switch tag {
	case "mono":
		return "1"
	case "stereo":
		return "2"
	case "5.1":
		return "6"
	case "7.1":
		return "8"
	default:
		return "0"
	}
}

// Inputs will always be absolute paths
func init() {
	metadata.RegisterExtractor(ExtractorID, &Extractor{ffmpeg: ffmpeg.New()})
}
