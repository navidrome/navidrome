package metadata

import (
	"bufio"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

type ffmpegExtractor struct{}

func (e *ffmpegExtractor) Extract(files ...string) (map[string]*Tags, error) {
	args := e.createProbeCommand(files)

	log.Trace("Executing command", "args", args)
	cmd := exec.Command(args[0], args[1:]...) // #nosec
	output, _ := cmd.CombinedOutput()
	fileTags := map[string]*Tags{}
	if len(output) == 0 {
		return fileTags, errors.New("error extracting metadata files")
	}
	infos := e.parseOutput(string(output))
	for file, info := range infos {
		tags, err := e.extractMetadata(file, info)
		// Skip files with errors
		if err == nil {
			fileTags[file] = tags
		}
	}
	return fileTags, nil
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
	bitRateRx = regexp.MustCompile(`^\s{2,4}Stream #\d+:\d+: (Audio):.*, (\d+) kb/s`)

	//    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 600x600 [SAR 1:1 DAR 1:1], 90k tbr, 90k tbn, 90k tbc`
	coverRx = regexp.MustCompile(`^\s{2,4}Stream #\d+:\d+: (Video):.*`)
)

func (e *ffmpegExtractor) parseOutput(output string) map[string]string {
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

func (e *ffmpegExtractor) extractMetadata(filePath, info string) (*Tags, error) {
	parsedTags := e.parseInfo(info)
	if len(parsedTags) == 0 {
		log.Trace("Not a media file. Skipping", "filePath", filePath)
		return nil, errors.New("not a media file")
	}

	tags := NewTag(filePath, parsedTags, map[string][]string{
		"disc":        {"tpa"},
		"has_picture": {"metadata_block_picture"},
	})
	return tags, nil
}

func (e *ffmpegExtractor) parseInfo(info string) map[string][]string {
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
			tags["bitrate"] = []string{match[2]}
		}
	}

	comment := tags["comment"]
	if len(comment) > 0 && comment[0] == "Cover (front)" {
		delete(tags, "comment")
	}

	return tags
}

var zeroTime = time.Date(0000, time.January, 1, 0, 0, 0, 0, time.UTC)

func (e *ffmpegExtractor) parseDuration(tag string) string {
	d, err := time.Parse("15:04:05", tag)
	if err != nil {
		return "0"
	}
	return strconv.FormatFloat(d.Sub(zeroTime).Seconds(), 'f', 2, 32)
}

// Inputs will always be absolute paths
func (e *ffmpegExtractor) createProbeCommand(inputs []string) []string {
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
