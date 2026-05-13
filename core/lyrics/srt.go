package lyrics

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

var srtTimeRegex = regexp.MustCompile(`^\s*(\d{1,2}):(\d{2}):(\d{2})[,.](\d{1,3})\s*$`)

func parseSRT(contents []byte) (model.LyricList, error) {
	raw := strings.ReplaceAll(string(contents), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	blocks := splitSRTBlocks(raw)
	lines := make([]model.Line, 0, len(blocks))

	for _, block := range blocks {
		line, ok, err := parseSRTBlock(block)
		if err != nil {
			return nil, err
		}
		if ok {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return nil, nil
	}

	lyrics := model.NormalizeLyrics(model.Lyrics{
		Lang:   "xxx",
		Line:   lines,
		Synced: true,
	})
	return model.LyricList{lyrics}, nil
}

func splitSRTBlocks(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, "\n\n")
	blocks := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			blocks = append(blocks, part)
		}
	}
	return blocks
}

func parseSRTBlock(block string) (model.Line, bool, error) {
	scanner := bytes.Split([]byte(block), []byte("\n"))
	if len(scanner) == 0 {
		return model.Line{}, false, nil
	}

	lines := make([]string, 0, len(scanner))
	for _, line := range scanner {
		lines = append(lines, strings.TrimSpace(string(line)))
	}

	if len(lines) == 0 {
		return model.Line{}, false, nil
	}

	startIdx := 0
	if digitsOnly(lines[0]) {
		startIdx = 1
	}
	if startIdx >= len(lines) {
		return model.Line{}, false, nil
	}

	timing := strings.Split(lines[startIdx], "-->")
	if len(timing) != 2 {
		return model.Line{}, false, nil
	}

	startMs, err := parseSRTTime(timing[0])
	if err != nil {
		return model.Line{}, false, err
	}
	endMs, err := parseSRTTime(timing[1])
	if err != nil {
		return model.Line{}, false, err
	}

	textLines := make([]string, 0, len(lines)-startIdx-1)
	for _, line := range lines[startIdx+1:] {
		if line == "" {
			continue
		}
		textLines = append(textLines, line)
	}

	value := str.SanitizeText(strings.Join(textLines, "\n"))
	if value == "" {
		return model.Line{}, false, nil
	}

	return model.Line{
		Start: &startMs,
		End:   &endMs,
		Value: value,
	}, true, nil
}

func parseSRTTime(value string) (int64, error) {
	match := srtTimeRegex.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return 0, strconv.ErrSyntax
	}

	hours, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseInt(match[3], 10, 64)
	if err != nil {
		return 0, err
	}
	millis, err := strconv.ParseInt(match[4], 10, 64)
	if err != nil {
		return 0, err
	}

	switch len(match[4]) {
	case 1:
		millis *= 100
	case 2:
		millis *= 10
	}

	return (((hours*60)+minutes)*60+seconds)*1000 + millis, nil
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
