package model

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/utils/str"
)

var (
	wordSyncRegex       = regexp.MustCompile(`<([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})([.:][0-9]{1,3})?>`)
	lrcLeadingTimeRegex = regexp.MustCompile(`^\s*\[(?:[0-9]{1,2}:)?[0-9]{1,2}:[0-9]{1,2}(?:[.:][0-9]{1,3})?]`)
	srtTimeLineRegex    = regexp.MustCompile(`^\s*(\d{2,}:\d{2}:\d{2}[,.]\d{1,3})\s*-->\s*(\d{2,}:\d{2}:\d{2}[,.]\d{1,3})`)
)

type parsedLRCSynced struct {
	timestamps []int64
	text       string
}

func parseLRCSyncedLine(line string) (*parsedLRCSynced, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false, nil
	}

	remaining := line
	var timestamps []int64
	for {
		match := lrcLeadingTimeRegex.FindStringIndex(remaining)
		if match == nil || match[0] != 0 {
			break
		}
		token := remaining[:match[1]]
		ts, err := parseTaggedTimestamp(token)
		if err != nil {
			return nil, false, err
		}
		timestamps = append(timestamps, ts)
		remaining = strings.TrimLeft(remaining[match[1]:], " \t")
	}

	wordTimes, cleanedText, err := extractWordTaggedText(remaining)
	if err != nil {
		return nil, false, err
	}
	if len(timestamps) == 0 && len(wordTimes) == 0 {
		return nil, false, nil
	}
	if len(timestamps) == 0 {
		timestamps = append(timestamps, wordTimes[0])
	}

	cleanedText = sanitizeLyricText(cleanedText)
	return &parsedLRCSynced{timestamps: timestamps, text: cleanedText}, true, nil
}

func stripWordSyncTags(line string) string {
	_, cleaned, err := extractWordTaggedText(line)
	if err != nil {
		return sanitizeLyricText(line)
	}
	return sanitizeLyricText(cleaned)
}

func extractWordTaggedText(line string) ([]int64, string, error) {
	matches := wordSyncRegex.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return nil, line, nil
	}

	var out strings.Builder
	var times []int64
	prev := 0
	for _, match := range matches {
		out.WriteString(line[prev:match[0]])
		ts, err := parseTimeToken(line[match[0]+1 : match[1]-1])
		if err != nil {
			return nil, "", err
		}
		times = append(times, ts)
		prev = match[1]
	}
	out.WriteString(line[prev:])
	return times, out.String(), nil
}

func parseTaggedTimestamp(token string) (int64, error) {
	token = strings.TrimSpace(token)
	if len(token) < 2 {
		return 0, fmt.Errorf("invalid lyric timestamp: %q", token)
	}
	return parseTimeToken(token[1 : len(token)-1])
}

func parseTimeToken(token string) (int64, error) {
	parts := strings.Split(token, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("invalid lyric timestamp: %q", token)
	}

	var hours int64
	var minutesPart string
	var secondsPart string
	if len(parts) == 3 {
		h, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		hours = h
		minutesPart = parts[1]
		secondsPart = parts[2]
	} else {
		minutesPart = parts[0]
		secondsPart = parts[1]
	}

	minutes, err := strconv.ParseInt(minutesPart, 10, 64)
	if err != nil {
		return 0, err
	}

	seconds := secondsPart
	fraction := ""
	if idx := strings.IndexAny(secondsPart, ".,"); idx >= 0 {
		seconds = secondsPart[:idx]
		fraction = secondsPart[idx+1:]
	}

	sec, err := strconv.ParseInt(seconds, 10, 64)
	if err != nil {
		return 0, err
	}

	var millis int64
	if fraction != "" {
		if len(fraction) > 3 {
			fraction = fraction[:3]
		}
		value, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
		switch len(fraction) {
		case 1:
			millis = value * 100
		case 2:
			millis = value * 10
		default:
			millis = value
		}
	}

	return (((hours*60+minutes)*60)+sec)*1000 + millis, nil
}

var embeddedTimestampRegex = regexp.MustCompile(`^\[\d{2}:\d{2}([.:]\d{2,3})?\]`)

func sanitizeLyricText(text string) string {
	text = str.SanitizeText(text)
	text = strings.TrimSpace(text)

	for {
		lower := strings.ToLower(text)
		switch {
		case strings.HasPrefix(lower, "[bg:") && strings.HasSuffix(text, "]"):
			text = strings.TrimSpace(text[len("[bg:") : len(text)-1])
		case embeddedTimestampRegex.MatchString(lower):
			text = strings.TrimSpace(embeddedTimestampRegex.ReplaceAllString(text, ""))
		case strings.HasPrefix(lower, "bg:"):
			text = strings.TrimSpace(text[len("bg:"):])
		case strings.HasPrefix(lower, "v1:"):
			text = strings.TrimSpace(text[len("v1:"):])
		case strings.HasPrefix(lower, "v2:"):
			text = strings.TrimSpace(text[len("v2:"):])
		case strings.HasPrefix(lower, "v3:"):
			text = strings.TrimSpace(text[len("v3:"):])
		default:
			return text
		}
	}
}

func parseSRTLyrics(language, text string) (*Lyrics, bool, error) {
	text = strings.TrimLeft(text, "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "1" {
		return nil, false, nil
	}

	var structured []Line
	i := 0
	for i < len(lines) {
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) {
			break
		}

		if _, err := strconv.Atoi(strings.TrimSpace(lines[i])); err != nil {
			return nil, false, nil
		}
		i++
		if i >= len(lines) {
			return nil, false, fmt.Errorf("invalid srt: missing timing line")
		}

		match := srtTimeLineRegex.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if match == nil {
			return nil, false, nil
		}
		start, err := parseSRTTimestamp(match[1])
		if err != nil {
			return nil, true, err
		}
		i++

		var textLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textLines = append(textLines, str.SanitizeText(lines[i]))
			i++
		}

		structured = append(structured, Line{
			Start: &start,
			Value: strings.TrimSpace(strings.Join(textLines, "\n")),
		})
	}

	return &Lyrics{Lang: language, Line: structured, Synced: len(structured) > 0}, len(structured) > 0, nil
}

func parseSRTTimestamp(value string) (int64, error) {
	value = strings.ReplaceAll(value, ",", ".")
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid srt timestamp: %q", value)
	}
	hours, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}
	secParts := strings.SplitN(parts[2], ".", 2)
	seconds, err := strconv.ParseInt(secParts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	var millis int64
	if len(secParts) == 2 {
		fraction := secParts[1]
		if len(fraction) > 3 {
			fraction = fraction[:3]
		}
		value, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
		switch len(fraction) {
		case 1:
			millis = value * 100
		case 2:
			millis = value * 10
		default:
			millis = value
		}
	}
	return (((hours*60+minutes)*60)+seconds)*1000 + millis, nil
}

type ttmlNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*ttmlNode
	Text     string
	IsText   bool
}

type ttmlParagraph struct {
	Start *int64
	End   *int64
	Text  string
}

func parseTTMLLyrics(language, text string) (*Lyrics, bool, error) {
	raw := strings.TrimSpace(strings.TrimLeft(text, "\ufeff"))
	if raw == "" || !strings.HasPrefix(raw, "<") {
		return nil, false, nil
	}
	raw = escapeBrokenXMLEntities(raw)

	root, err := parseXMLTree(raw)
	if err != nil {
		var syntaxErr *xml.SyntaxError
		if strings.Contains(strings.ToLower(err.Error()), "xml") || errors.As(err, &syntaxErr) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if root == nil || root.Name.Local != "tt" {
		return nil, false, nil
	}

	params := newTTMLTiming(root)
	var paragraphs []ttmlParagraph
	var walk func(node *ttmlNode, parent *ttmlRange)
	walk = func(node *ttmlNode, parent *ttmlRange) {
		current := params.childRange(node, parent)
		if node.Name.Local == "p" {
			text := strings.TrimSpace(renderTTMLText(node))
			if text != "" {
				paragraphs = append(paragraphs, ttmlParagraph{
					Start: current.start,
					End:   current.end,
					Text:  str.SanitizeText(text),
				})
			}
			return
		}
		seqCursor := current.start
		for _, child := range node.Children {
			childParent := current
			if strings.EqualFold(attr(node, "timeContainer"), "seq") && seqCursor != nil {
				childParent.start = seqCursor
			}
			walk(child, childParent)
			if strings.EqualFold(attr(node, "timeContainer"), "seq") && len(paragraphs) > 0 {
				last := paragraphs[len(paragraphs)-1]
				if last.End != nil {
					seqCursor = last.End
				}
			}
		}
	}

	body := findTTMLBody(root)
	if body == nil {
		return nil, true, nil
	}
	walk(body, &ttmlRange{})

	if len(paragraphs) == 0 {
		return &Lyrics{Lang: language, Synced: false}, true, nil
	}

	synced := false
	lines := make([]Line, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		start := paragraph.Start
		if start != nil {
			synced = true
		}
		lines = append(lines, Line{Start: start, Value: paragraph.Text})
	}
	return &Lyrics{Lang: language, Line: lines, Synced: synced}, true, nil
}

func parseXMLTree(input string) (*ttmlNode, error) {
	decoder := xml.NewDecoder(strings.NewReader(input))
	var stack []*ttmlNode
	var root *ttmlNode
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok := token.(type) {
		case xml.StartElement:
			node := &ttmlNode{Name: tok.Name, Attrs: tok.Attr}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("malformed xml")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, &ttmlNode{
					Text:   string(tok),
					IsText: true,
				})
			}
		}
	}
	return root, nil
}

func renderTTMLText(node *ttmlNode) string {
	if node == nil {
		return ""
	}
	if node.IsText {
		return node.Text
	}
	if node.Name.Local == "br" {
		return "\n"
	}
	var out strings.Builder
	for _, child := range node.Children {
		out.WriteString(renderTTMLText(child))
	}
	return out.String()
}

func findTTMLBody(node *ttmlNode) *ttmlNode {
	if node == nil {
		return nil
	}
	if node.Name.Local == "body" {
		return node
	}
	for _, child := range node.Children {
		if found := findTTMLBody(child); found != nil {
			return found
		}
	}
	return nil
}

type ttmlTiming struct {
	frameRate    float64
	subFrameRate float64
	tickRate     float64
}

type ttmlRange struct {
	start *int64
	end   *int64
}

func newTTMLTiming(root *ttmlNode) ttmlTiming {
	frameRate := parseFloatDefault(attr(root, "frameRate"), 30)
	if mult := strings.Fields(attr(root, "frameRateMultiplier")); len(mult) == 2 {
		num := parseFloatDefault(mult[0], 1)
		den := parseFloatDefault(mult[1], 1)
		if den != 0 {
			frameRate *= num / den
		}
	}
	return ttmlTiming{
		frameRate:    frameRate,
		subFrameRate: parseFloatDefault(attr(root, "subFrameRate"), 1),
		tickRate:     parseFloatDefault(attr(root, "tickRate"), 1),
	}
}

func (t ttmlTiming) childRange(node *ttmlNode, parent *ttmlRange) *ttmlRange {
	if node == nil {
		return &ttmlRange{}
	}
	baseStart := int64(0)
	if parent != nil && parent.start != nil {
		baseStart = *parent.start
	}

	begin, _ := t.parseTimestamp(attr(node, "begin"), baseStart)
	end, _ := t.parseTimestamp(attr(node, "end"), baseStart)
	dur, _ := t.parseDuration(attr(node, "dur"))

	if begin == nil && parent != nil && parent.start != nil {
		begin = parent.start
	}
	if end == nil && dur != nil && begin != nil {
		value := *begin + *dur
		end = &value
	}
	if end == nil && parent != nil && parent.end != nil {
		end = parent.end
	}
	if begin == nil && dur != nil && end != nil {
		value := *end - *dur
		begin = &value
	}
	return &ttmlRange{start: begin, end: end}
}

func (t ttmlTiming) parseTimestamp(value string, offset int64) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if ms, err := t.parseClockTime(value); err == nil {
		ms += offset
		return &ms, nil
	}
	if ms, err := t.parseOffsetTime(value); err == nil {
		ms += offset
		return &ms, nil
	}
	if ms, err := t.parseAppleTime(value); err == nil {
		return &ms, nil
	}
	return nil, fmt.Errorf("invalid ttml timestamp: %q", value)
}

func (t ttmlTiming) parseDuration(value string) (*int64, error) {
	ms, err := t.parseOffsetTime(strings.TrimSpace(value))
	if err == nil {
		return &ms, nil
	}
	ms, err = t.parseClockTime(strings.TrimSpace(value))
	if err == nil {
		return &ms, nil
	}
	return nil, err
}

func (t ttmlTiming) parseAppleTime(value string) (int64, error) {
	match := regexp.MustCompile(`^(?:(\d+):)?(?:(\d+):)?(\d+(?:\.\d+)?)$`).FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("not apple time")
	}
	hours := 0.0
	minutes := 0.0
	if match[2] != "" {
		hours, _ = strconv.ParseFloat(match[1], 64)
		minutes, _ = strconv.ParseFloat(match[2], 64)
	} else if match[1] != "" {
		minutes, _ = strconv.ParseFloat(match[1], 64)
	}
	seconds, _ := strconv.ParseFloat(match[3], 64)
	return int64(hours*3600000 + minutes*60000 + seconds*1000), nil
}

func (t ttmlTiming) parseClockTime(value string) (int64, error) {
	match := regexp.MustCompile(`^(\d{2,}):(\d{2}):(\d{2})(?:(\.\d+)|:(\d{2})(?:\.(\d+))?)?$`).FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("not clock time")
	}
	hours, _ := strconv.ParseFloat(match[1], 64)
	minutes, _ := strconv.ParseFloat(match[2], 64)
	seconds, _ := strconv.ParseFloat(match[3]+match[4], 64)
	frameSecs := 0.0
	if match[5] != "" {
		frames, _ := strconv.ParseFloat(match[5], 64)
		frameSecs = frames / t.frameRate
	}
	subFrameSecs := 0.0
	if match[6] != "" {
		subFrames, _ := strconv.ParseFloat(match[6], 64)
		subFrameSecs = subFrames / t.subFrameRate / t.frameRate
	}
	return int64(hours*3600000 + minutes*60000 + (seconds+frameSecs+subFrameSecs)*1000), nil
}

func (t ttmlTiming) parseOffsetTime(value string) (int64, error) {
	match := regexp.MustCompile(`^(\d+(?:\.\d+)?)(h|m|s|ms|f|t)$`).FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("not offset time")
	}
	number, _ := strconv.ParseFloat(match[1], 64)
	switch match[2] {
	case "h":
		number *= 3600000
	case "m":
		number *= 60000
	case "s":
		number *= 1000
	case "ms":
	case "f":
		number = number / t.frameRate * 1000
	case "t":
		number = number / t.tickRate * 1000
	}
	return int64(number), nil
}

func attr(node *ttmlNode, local string) string {
	for _, a := range node.Attrs {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func parseFloatDefault(value string, fallback float64) float64 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func escapeBrokenXMLEntities(input string) string {
	var out strings.Builder
	out.Grow(len(input))

	for i := 0; i < len(input); i++ {
		if input[i] != '&' {
			out.WriteByte(input[i])
			continue
		}

		semi := strings.IndexByte(input[i+1:], ';')
		if semi >= 0 {
			entity := input[i+1 : i+1+semi]
			if entity != "" {
				if entity[0] == '#' {
					if len(entity) > 1 && (entity[1] == 'x' || entity[1] == 'X') {
						if isAlphaNumeric(entity[2:]) {
							out.WriteByte('&')
							continue
						}
					} else if isDigits(entity[1:]) {
						out.WriteByte('&')
						continue
					}
				} else if isAlphaNumeric(entity) {
					out.WriteByte('&')
					continue
				}
			}
		}

		out.WriteString("&amp;")
	}

	return out.String()
}

func isAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
