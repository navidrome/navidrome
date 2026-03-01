package lyrics

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

const (
	defaultTTMLFrameRate    = 30.0
	defaultTTMLSubFrameRate = 1.0
	defaultTTMLTickRate     = 1.0

	ttmlLyricKindMain          = "main"
	ttmlLyricKindTranslation   = "translation"
	ttmlLyricKindPronunciation = "pronunciation"
)

var offsetTimeRegex = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)(h|m|s|ms|f|t)$`)
var xmlEncodingRegex = regexp.MustCompile(`(?i)<\?xml([^>]*?)encoding\s*=\s*["'][^"']+["']([^>]*)\?>`)

type ttmlTimeKind int

const (
	ttmlTimeAbsolute ttmlTimeKind = iota
	ttmlTimeOffset
	ttmlTimeAmbiguous
)

type ttmlTimingParams struct {
	frameRate    float64
	subFrameRate float64
	tickRate     float64
}

type ttmlTimingContext struct {
	lang     string
	role     string
	begin    int64
	hasBegin bool
	end      int64
	hasEnd   bool
	invalid  bool
}

type ttmlLineRef struct {
	order int
	line  model.Line
}

type ttmlMetadataEntry struct {
	key  string
	line model.Line
	seq  int
}

type ttmlResolvedMetadataLine struct {
	order int
	seq   int
	line  model.Line
}

type ttmlParser struct {
	decoder *xml.Decoder
	params  ttmlTimingParams

	mainLangOrder   []string
	mainLinesByLang map[string][]model.Line

	mainLineRefsByKey map[string]ttmlLineRef
	mainLineOrder     int

	translationLangOrder   []string
	translationEntriesByLg map[string][]ttmlMetadataEntry

	pronunciationLangOrder   []string
	pronunciationEntriesByLg map[string][]ttmlMetadataEntry

	metadataSeq int
}

func parseTTML(contents []byte) (model.LyricList, error) {
	contents = xmlEncodingRegex.ReplaceAll(contents, []byte(`<?xml$1encoding="UTF-8"$2?>`))

	p := ttmlParser{
		decoder: xml.NewDecoder(bytes.NewReader(contents)),
		params: ttmlTimingParams{
			frameRate:    defaultTTMLFrameRate,
			subFrameRate: defaultTTMLSubFrameRate,
			tickRate:     defaultTTMLTickRate,
		},
		mainLinesByLang:          make(map[string][]model.Line),
		mainLineRefsByKey:        make(map[string]ttmlLineRef),
		translationEntriesByLg:   make(map[string][]ttmlMetadataEntry),
		pronunciationEntriesByLg: make(map[string][]ttmlMetadataEntry),
	}

	root := ttmlTimingContext{lang: "xxx"}

	for {
		token, err := p.decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if err := p.parseElement(start, root); err != nil {
			return nil, err
		}
	}

	return p.toLyricList(), nil
}

func (p *ttmlParser) parseElement(start xml.StartElement, parent ttmlTimingContext) error {
	local := strings.ToLower(start.Name.Local)
	if local == "tt" {
		p.updateTimingParams(start.Attr)
	}

	switch local {
	case "translation":
		return p.parseMetadataTrack(start, parent, ttmlLyricKindTranslation)
	case "transliteration":
		return p.parseMetadataTrack(start, parent, ttmlLyricKindPronunciation)
	}

	ctx := p.childContext(start.Attr, parent)
	if local == "p" {
		lineText, tokens, err := p.parseParagraph(ctx)
		if err != nil {
			return err
		}
		if ctx.invalid || lineText == "" {
			return nil
		}

		parsedLine := model.Line{Value: lineText}
		if ctx.hasBegin {
			startMs := ctx.begin
			parsedLine.Start = &startMs
		}
		if ctx.hasEnd {
			endMs := ctx.end
			parsedLine.End = &endMs
		}
		if len(tokens) > 0 {
			parsedLine.Cue = tokens
		}
		parsedLine = hydrateLineTimingFromTokens(parsedLine)

		lineKey, _ := attrValue(start.Attr, "key")
		p.addMainLine(ctx.lang, lineKey, parsedLine)
		return nil
	}

	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			nextParent := ctx
			if ctx.invalid {
				// Best effort: ignore invalid timing in container elements, and
				// continue traversing descendants with parent context.
				nextParent = parent
			}
			if err := p.parseElement(t, nextParent); err != nil {
				return err
			}
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, start.Name.Local) {
				return nil
			}
		}
	}
}

func (p *ttmlParser) parseMetadataTrack(start xml.StartElement, parent ttmlTimingContext, kind string) error {
	ctx := p.childContext(start.Attr, parent)
	lang := normalizeTTMLLang(ctx.lang)

	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if strings.EqualFold(t.Name.Local, "text") {
				entry, ok, err := p.parseMetadataText(t, ctx)
				if err != nil {
					return err
				}
				if ok {
					p.addMetadataEntry(kind, lang, entry)
				}
				continue
			}

			nextParent := ctx
			if ctx.invalid {
				nextParent = parent
			}
			if err := p.parseElement(t, nextParent); err != nil {
				return err
			}
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, start.Name.Local) {
				return nil
			}
		}
	}
}

func (p *ttmlParser) parseMetadataText(start xml.StartElement, parent ttmlTimingContext) (ttmlMetadataEntry, bool, error) {
	forKey, hasFor := attrValue(start.Attr, "for")
	forKey = strings.TrimSpace(forKey)

	value, tokens, err := p.parseInlineElement(start, parent)
	if err != nil {
		return ttmlMetadataEntry{}, false, err
	}
	if !hasFor || forKey == "" {
		return ttmlMetadataEntry{}, false, nil
	}

	ctx := p.childContext(start.Attr, parent)
	if ctx.invalid {
		return ttmlMetadataEntry{}, false, nil
	}

	line := model.Line{Value: sanitizeTTMLText(value)}
	if ctx.hasBegin {
		startMs := ctx.begin
		line.Start = &startMs
	}
	if ctx.hasEnd {
		endMs := ctx.end
		line.End = &endMs
	}
	if len(tokens) > 0 {
		line.Cue = tokens
	}
	line = hydrateLineTimingFromTokens(line)

	if line.Value == "" && len(line.Cue) == 0 {
		return ttmlMetadataEntry{}, false, nil
	}

	return ttmlMetadataEntry{key: forKey, line: line}, true, nil
}

func (p *ttmlParser) parseParagraph(parent ttmlTimingContext) (string, []model.Cue, error) {
	var text strings.Builder
	var tokens []model.Cue

	for {
		token, err := p.decoder.Token()
		if err != nil {
			return "", nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			value, inlineTokens, err := p.parseInlineElement(t, parent)
			if err != nil {
				return "", nil, err
			}
			text.WriteString(value)
			tokens = append(tokens, inlineTokens...)
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, "p") {
				return sanitizeTTMLText(text.String()), tokens, nil
			}
		case xml.CharData:
			text.WriteString(string(t))
		}
	}
}

func (p *ttmlParser) parseInlineElement(start xml.StartElement, parent ttmlTimingContext) (string, []model.Cue, error) {
	local := strings.ToLower(start.Name.Local)
	if local == "br" {
		return "\n", nil, nil
	}

	ctx := p.childContext(start.Attr, parent)
	_, hasBegin := attrValue(start.Attr, "begin")
	_, hasEnd := attrValue(start.Attr, "end")
	_, hasDur := attrValue(start.Attr, "dur")
	hasOwnTiming := hasBegin || hasEnd || hasDur

	var text strings.Builder
	var tokens []model.Cue

	for {
		token, err := p.decoder.Token()
		if err != nil {
			return "", nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			value, inlineTokens, err := p.parseInlineElement(t, ctx)
			if err != nil {
				return "", nil, err
			}
			text.WriteString(value)
			tokens = append(tokens, inlineTokens...)
		case xml.EndElement:
			if !strings.EqualFold(t.Name.Local, start.Name.Local) {
				continue
			}

			value := text.String()
			tokenText := sanitizeTTMLText(value)
			if local == "span" && hasOwnTiming && !ctx.invalid && tokenText != "" && len(tokens) == 0 {
				parsedToken := model.Cue{
					Value: tokenText,
					Role:  ctx.role,
				}
				if ctx.hasBegin {
					startMs := ctx.begin
					parsedToken.Start = &startMs
				}
				if ctx.hasEnd {
					endMs := ctx.end
					parsedToken.End = &endMs
				}
				tokens = append(tokens, parsedToken)
			}

			return value, tokens, nil
		case xml.CharData:
			text.WriteString(string(t))
		}
	}
}

func (p *ttmlParser) toLyricList() model.LyricList {
	res := make(model.LyricList, 0, len(p.mainLangOrder)+len(p.translationLangOrder)+len(p.pronunciationLangOrder))
	for _, lang := range p.mainLangOrder {
		lines := p.mainLinesByLang[lang]
		if len(lines) == 0 {
			continue
		}
		res = append(res, model.Lyrics{
			Kind:   ttmlLyricKindMain,
			Lang:   lang,
			Line:   lines,
			Synced: linesAreSynced(lines),
		})
	}

	res = append(res, p.buildMetadataLyrics(ttmlLyricKindTranslation, p.translationLangOrder, p.translationEntriesByLg)...)
	res = append(res, p.buildMetadataLyrics(ttmlLyricKindPronunciation, p.pronunciationLangOrder, p.pronunciationEntriesByLg)...)
	return res
}

func (p *ttmlParser) buildMetadataLyrics(kind string, langOrder []string, entriesByLang map[string][]ttmlMetadataEntry) model.LyricList {
	res := make(model.LyricList, 0, len(langOrder))

	for _, lang := range langOrder {
		entries := entriesByLang[lang]
		if len(entries) == 0 {
			continue
		}

		seenKeys := make(map[string]struct{}, len(entries))
		resolved := make([]ttmlResolvedMetadataLine, 0, len(entries))
		for _, entry := range entries {
			if _, exists := seenKeys[entry.key]; exists {
				continue
			}
			seenKeys[entry.key] = struct{}{}

			ref, ok := p.mainLineRefsByKey[entry.key]
			if !ok {
				log.Warn("Skipping TTML metadata line without matching key", "kind", kind, "lang", lang, "key", entry.key)
				continue
			}

			line := entry.line
			if line.Start == nil && ref.line.Start != nil {
				startMs := *ref.line.Start
				line.Start = &startMs
			}
			if line.End == nil && ref.line.End != nil {
				endMs := *ref.line.End
				line.End = &endMs
			}
			line = hydrateLineTimingFromTokens(line)

			if line.Value == "" && len(line.Cue) == 0 {
				continue
			}

			resolved = append(resolved, ttmlResolvedMetadataLine{
				order: ref.order,
				seq:   entry.seq,
				line:  line,
			})
		}

		if len(resolved) == 0 {
			continue
		}

		sort.SliceStable(resolved, func(i, j int) bool {
			if resolved[i].order != resolved[j].order {
				return resolved[i].order < resolved[j].order
			}
			return resolved[i].seq < resolved[j].seq
		})

		lines := make([]model.Line, len(resolved))
		for i := range resolved {
			lines[i] = resolved[i].line
		}

		res = append(res, model.Lyrics{
			Kind:   kind,
			Lang:   lang,
			Line:   lines,
			Synced: linesAreSynced(lines),
		})
	}

	return res
}

func (p *ttmlParser) addMainLine(lang string, lineKey string, line model.Line) {
	lang = normalizeTTMLLang(lang)
	if _, ok := p.mainLinesByLang[lang]; !ok {
		p.mainLangOrder = append(p.mainLangOrder, lang)
	}
	p.mainLinesByLang[lang] = append(p.mainLinesByLang[lang], line)

	lineKey = strings.TrimSpace(lineKey)
	if lineKey != "" {
		if _, exists := p.mainLineRefsByKey[lineKey]; !exists {
			p.mainLineRefsByKey[lineKey] = ttmlLineRef{
				order: p.mainLineOrder,
				line:  line,
			}
		}
	}
	p.mainLineOrder++
}

func (p *ttmlParser) addMetadataEntry(kind string, lang string, entry ttmlMetadataEntry) {
	lang = normalizeTTMLLang(lang)
	entry.seq = p.metadataSeq
	p.metadataSeq++

	switch kind {
	case ttmlLyricKindTranslation:
		if _, ok := p.translationEntriesByLg[lang]; !ok {
			p.translationLangOrder = append(p.translationLangOrder, lang)
		}
		p.translationEntriesByLg[lang] = append(p.translationEntriesByLg[lang], entry)
	case ttmlLyricKindPronunciation:
		if _, ok := p.pronunciationEntriesByLg[lang]; !ok {
			p.pronunciationLangOrder = append(p.pronunciationLangOrder, lang)
		}
		p.pronunciationEntriesByLg[lang] = append(p.pronunciationEntriesByLg[lang], entry)
	}
}

func (p *ttmlParser) childContext(attrs []xml.Attr, parent ttmlTimingContext) ttmlTimingContext {
	ctx := parent

	if lang, ok := attrValue(attrs, "lang"); ok {
		ctx.lang = normalizeTTMLLang(lang)
	}
	if role, ok := attrValue(attrs, "role"); ok {
		role = strings.TrimSpace(role)
		if role != "" {
			if ctx.role == "" {
				ctx.role = role
			} else if !strings.Contains(ctx.role, role) {
				ctx.role = ctx.role + " " + role
			}
		}
	}

	beginExpr, hasBegin := attrValue(attrs, "begin")
	endExpr, hasEnd := attrValue(attrs, "end")
	durExpr, hasDur := attrValue(attrs, "dur")

	if hasBegin {
		begin, kind, ok := parseTTMLTimeExpression(beginExpr, p.params)
		if !ok {
			ctx.invalid = true
			return ctx
		}

		base := int64(0)
		if parent.hasBegin {
			base = parent.begin
		}
		ctx.begin = resolveTTMLTime(begin, kind, base, parent)
		ctx.hasBegin = true
	} else {
		ctx.begin = parent.begin
		ctx.hasBegin = parent.hasBegin
	}

	var calculatedEnd int64
	calculatedHasEnd := false

	if hasEnd {
		end, kind, ok := parseTTMLTimeExpression(endExpr, p.params)
		if !ok {
			ctx.invalid = true
			return ctx
		}

		base := ctx.begin
		if !ctx.hasBegin {
			base = parent.begin
		}
		calculatedEnd = resolveTTMLTime(end, kind, base, parent)
		calculatedHasEnd = true
	}

	if hasDur {
		dur, ok := parseTTMLDurationExpression(durExpr, p.params)
		if !ok {
			ctx.invalid = true
			return ctx
		}
		if ctx.hasBegin {
			durEnd := ctx.begin + dur
			if !calculatedHasEnd || durEnd < calculatedEnd {
				calculatedEnd = durEnd
				calculatedHasEnd = true
			}
		}
	}

	if !calculatedHasEnd && parent.hasEnd {
		calculatedEnd = parent.end
		calculatedHasEnd = true
	}

	ctx.end = calculatedEnd
	ctx.hasEnd = calculatedHasEnd
	return ctx
}

func (p *ttmlParser) updateTimingParams(attrs []xml.Attr) {
	frameRate := p.params.frameRate
	if value, ok := attrValue(attrs, "frameRate"); ok {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			frameRate = parsed
		}
	}

	if value, ok := attrValue(attrs, "frameRateMultiplier"); ok {
		parts := strings.Fields(value)
		if len(parts) == 2 {
			numerator, errA := strconv.ParseFloat(parts[0], 64)
			denominator, errB := strconv.ParseFloat(parts[1], 64)
			if errA == nil && errB == nil && denominator > 0 {
				frameRate = frameRate * (numerator / denominator)
			}
		}
	}

	subFrameRate := p.params.subFrameRate
	if value, ok := attrValue(attrs, "subFrameRate"); ok {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			subFrameRate = parsed
		}
	}

	tickRate := p.params.tickRate
	if value, ok := attrValue(attrs, "tickRate"); ok {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			tickRate = parsed
		}
	}

	p.params.frameRate = max(frameRate, defaultTTMLFrameRate)
	p.params.subFrameRate = max(subFrameRate, defaultTTMLSubFrameRate)
	p.params.tickRate = max(tickRate, defaultTTMLTickRate)
}

func parseTTMLDurationExpression(expr string, params ttmlTimingParams) (int64, bool) {
	value, _, ok := parseTTMLTimeExpression(expr, params)
	return value, ok
}

func resolveTTMLTime(value int64, kind ttmlTimeKind, base int64, parent ttmlTimingContext) int64 {
	switch kind {
	case ttmlTimeAbsolute:
		return value
	case ttmlTimeOffset:
		return base + value
	case ttmlTimeAmbiguous:
		absolute := value
		offset := base + value

		// No parent timing context → no reference frame for offsets.
		// Prefer absolute when offset differs (i.e., base > 0).
		if !parent.hasBegin && !parent.hasEnd && base != 0 {
			return absolute
		}

		if parent.hasBegin && parent.hasEnd {
			absoluteInParent := absolute >= parent.begin && absolute <= parent.end
			offsetInParent := offset >= parent.begin && offset <= parent.end
			if absoluteInParent && !offsetInParent {
				return absolute
			}
			if offsetInParent && !absoluteInParent {
				return offset
			}
		}

		if parent.hasBegin {
			if absolute < parent.begin && offset >= parent.begin {
				return offset
			}
			if absolute >= parent.begin && offset > absolute {
				return absolute
			}
		}
		return offset
	default:
		return base + value
	}
}

func parseTTMLTimeExpression(expr string, params ttmlTimingParams) (int64, ttmlTimeKind, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, ttmlTimeOffset, false
	}

	lower := strings.ToLower(expr)
	if strings.Contains(lower, "wallclock(") ||
		strings.Contains(lower, ".begin") ||
		strings.Contains(lower, ".end") {
		log.Warn("Unsupported TTML time expression", "value", expr)
		return 0, ttmlTimeOffset, false
	}

	// Best-effort support for non-standard TTML seen in the wild where a
	// bare decimal value is used (implicitly seconds), e.g. "0.170".
	if value, err := strconv.ParseFloat(lower, 64); err == nil && value >= 0 {
		return int64(math.Round(value * 1000)), ttmlTimeAmbiguous, true
	}

	if matches := offsetTimeRegex.FindStringSubmatch(lower); len(matches) == 3 {
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, ttmlTimeOffset, false
		}

		unit := matches[2]
		seconds := 0.0
		switch unit {
		case "h":
			seconds = value * 60 * 60
		case "m":
			seconds = value * 60
		case "s":
			seconds = value
		case "ms":
			seconds = value / 1000
		case "f":
			seconds = value / params.frameRate
		case "t":
			seconds = value / params.tickRate
		default:
			return 0, ttmlTimeOffset, false
		}

		return int64(math.Round(seconds * 1000)), ttmlTimeOffset, true
	}

	colonCount := strings.Count(expr, ":")
	switch colonCount {
	case 1, 2:
		clockMs, ok := parseTTMLClockTime(expr)
		if !ok {
			return 0, ttmlTimeAbsolute, false
		}
		return clockMs, ttmlTimeAbsolute, true
	case 3:
		framesMs, ok := parseTTMLFrameTime(expr, params)
		if !ok {
			return 0, ttmlTimeAbsolute, false
		}
		return framesMs, ttmlTimeAbsolute, true
	default:
		log.Warn("Unsupported TTML time expression", "value", expr)
		return 0, ttmlTimeOffset, false
	}
}

func parseTTMLClockTime(value string) (int64, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, false
	}

	hours := int64(0)
	minutesIdx := 0
	if len(parts) == 3 {
		h, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, false
		}
		hours = h
		minutesIdx = 1
	}

	minutes, err := strconv.ParseInt(parts[minutesIdx], 10, 64)
	if err != nil {
		return 0, false
	}

	seconds, err := strconv.ParseFloat(parts[minutesIdx+1], 64)
	if err != nil {
		return 0, false
	}

	totalSeconds := float64(hours*60*60+minutes*60) + seconds
	return int64(math.Round(totalSeconds * 1000)), true
}

func parseTTMLFrameTime(value string, params ttmlTimingParams) (int64, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 4 {
		return 0, false
	}

	hours, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}

	minutes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}

	seconds, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, false
	}

	frameParts := strings.SplitN(parts[3], ".", 2)
	frames, err := strconv.ParseFloat(frameParts[0], 64)
	if err != nil {
		return 0, false
	}

	subFrames := 0.0
	if len(frameParts) == 2 {
		subFrames, err = strconv.ParseFloat(frameParts[1], 64)
		if err != nil {
			return 0, false
		}
	}

	totalSeconds := float64(hours*60*60 + minutes*60 + seconds)
	totalSeconds += frames / params.frameRate
	totalSeconds += subFrames / (params.subFrameRate * params.frameRate)

	return int64(math.Round(totalSeconds * 1000)), true
}

func attrValue(attrs []xml.Attr, key string) (string, bool) {
	for _, attr := range attrs {
		if strings.EqualFold(attr.Name.Local, key) {
			return strings.TrimSpace(attr.Value), true
		}
	}
	return "", false
}

func normalizeTTMLLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "xxx"
	}
	return lang
}

func sanitizeTTMLText(raw string) string {
	raw = str.SanitizeText(raw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func linesAreSynced(lines []model.Line) bool {
	for i := range lines {
		if lines[i].Start != nil {
			return true
		}
		for j := range lines[i].Cue {
			if lines[i].Cue[j].Start != nil {
				return true
			}
		}
	}
	return false
}

func hydrateLineTimingFromTokens(line model.Line) model.Line {
	if len(line.Cue) == 0 {
		return line
	}

	var earliestStart *int64
	var latestEnd *int64
	for i := range line.Cue {
		token := line.Cue[i]
		if token.Start != nil {
			if earliestStart == nil || *token.Start < *earliestStart {
				v := *token.Start
				earliestStart = &v
			}
		}

		candidateEnd := token.End
		if candidateEnd == nil {
			candidateEnd = token.Start
		}
		if candidateEnd != nil {
			if latestEnd == nil || *candidateEnd > *latestEnd {
				v := *candidateEnd
				latestEnd = &v
			}
		}
	}

	if line.Start == nil && earliestStart != nil {
		v := *earliestStart
		line.Start = &v
	}
	if line.End == nil && latestEnd != nil {
		v := *latestEnd
		line.End = &v
	}
	return line
}

func max(v float64, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	return v
}
