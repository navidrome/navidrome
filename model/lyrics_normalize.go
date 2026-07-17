package model

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/navidrome/navidrome/utils/gg"
)

// NormalizeLyrics returns a canonical, independent copy of lyrics. It keeps
// source line order and overlapping vocal timelines intact while repairing
// timing only when the source contains enough information to do so.
func NormalizeLyrics(lyrics Lyrics) Lyrics {
	out := cloneLyrics(lyrics)
	out.Line = normalizeCueLines(out.Line, out.Agents)
	out.Agents = pruneLyricAgents(out.Line, out.Agents)
	return out
}

func cloneLyrics(lyrics Lyrics) Lyrics {
	out := lyrics
	out.Offset = gg.Clone(lyrics.Offset)
	out.Agents = slices.Clone(lyrics.Agents)
	out.Line = make([]Line, len(lyrics.Line))
	for i := range lyrics.Line {
		out.Line[i] = cloneLyricLine(lyrics.Line[i])
	}
	return out
}

func cloneLyricLine(line Line) Line {
	out := line
	out.Start = gg.Clone(line.Start)
	out.End = gg.Clone(line.End)
	out.Cue = make([]Cue, len(line.Cue))
	for i := range line.Cue {
		out.Cue[i] = line.Cue[i]
		out.Cue[i].Start = gg.Clone(line.Cue[i].Start)
		out.Cue[i].End = gg.Clone(line.Cue[i].End)
	}
	if len(out.Cue) == 0 {
		out.Cue = nil
	}
	return out
}

func normalizeCueLines(lines []Line, agents []Agent) []Line {
	if len(lines) == 0 {
		return lines
	}

	knownAgents := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		id := strings.TrimSpace(agent.ID)
		if id != "" {
			knownAgents[id] = struct{}{}
		}
	}

	for i := range lines {
		line := lines[i]
		if line.Start != nil && *line.Start < 0 {
			line.Start = nil
			line.End = nil
		}
		if line.Start != nil && line.End != nil && *line.End < *line.Start {
			line.End = nil
		}

		if len(line.Cue) == 0 {
			if line.Start == nil {
				line.End = nil
			}
			lines[i] = line
			continue
		}

		cues, valid := validateCueGeometry(line.Value, line.Cue, knownAgents)
		if !valid {
			line.Cue = nil
			lines[i] = line
			continue
		}

		var nextTimedStart *int64
		for j := i + 1; j < len(lines); j++ {
			if lines[j].Start != nil {
				nextTimedStart = gg.Clone(lines[j].Start)
				break
			}
		}

		line.Cue = normalizeCueEndsByAgent(cues, line.End, nextTimedStart)
		line = widenLineTiming(line, cues, nextTimedStart)
		lines[i] = line
	}

	return lines
}

func validateCueGeometry(lineValue string, cues []Cue, knownAgents map[string]struct{}) ([]Cue, bool) {
	if !utf8.ValidString(lineValue) {
		return nil, false
	}

	out := make([]Cue, len(cues))
	lastByteEndByAgent := make(map[string]int)
	lastStartByAgent := make(map[string]int64)
	seenAgent := make(map[string]bool)
	for i, cue := range cues {
		out[i] = cue
		out[i].Start = gg.Clone(cue.Start)
		out[i].End = gg.Clone(cue.End)

		if cue.Start == nil || *cue.Start < 0 || !utf8.ValidString(cue.Value) {
			return nil, false
		}
		if cue.ByteStart < 0 || cue.ByteEnd < cue.ByteStart || cue.ByteEnd >= len(lineValue) {
			return nil, false
		}
		agentID := strings.TrimSpace(cue.AgentID)
		out[i].AgentID = agentID
		if agentID != "" {
			if _, ok := knownAgents[agentID]; !ok {
				return nil, false
			}
		}
		if lastByteEnd, seen := lastByteEndByAgent[agentID]; seen && cue.ByteStart <= lastByteEnd {
			return nil, false
		}
		if !isRuneBoundary(lineValue, cue.ByteStart) || !isRuneBoundary(lineValue, cue.ByteEnd+1) {
			return nil, false
		}
		if lineValue[cue.ByteStart:cue.ByteEnd+1] != cue.Value {
			return nil, false
		}
		if seenAgent[agentID] && *cue.Start < lastStartByAgent[agentID] {
			return nil, false
		}
		seenAgent[agentID] = true
		lastStartByAgent[agentID] = *cue.Start
		lastByteEndByAgent[agentID] = cue.ByteEnd

		if cue.End != nil && *cue.End < *cue.Start {
			out[i].End = nil
		}
	}
	return out, true
}

func isRuneBoundary(value string, offset int) bool {
	return offset == 0 || offset == len(value) || utf8.RuneStart(value[offset])
}

func normalizeCueEndsByAgent(cues []Cue, lineEnd, nextTimedStart *int64) []Cue {
	groups := make(map[string][]int)
	order := make([]string, 0, 2)
	for i := range cues {
		id := cues[i].AgentID
		if _, ok := groups[id]; !ok {
			order = append(order, id)
		}
		groups[id] = append(groups[id], i)
	}

	out := slices.Clone(cues)
	for _, id := range order {
		idxs := groups[id]
		group := make([]Cue, len(idxs))
		for i, pos := range idxs {
			group[i] = cues[pos]
		}
		group = normalizePartialCueEnds(group, lineEnd, nextTimedStart)
		for i, pos := range idxs {
			out[pos] = group[i]
		}
	}
	return out
}

func normalizePartialCueEnds(cues []Cue, lineEnd, nextTimedStart *int64) []Cue {
	out := slices.Clone(cues)
	hasEnd := false
	for _, cue := range out {
		if cue.End != nil {
			hasEnd = true
			break
		}
	}
	if !hasEnd {
		return out
	}

	complete := true
	for i := range out {
		end := out[i].End
		if end == nil && i+1 < len(out) {
			end = out[i+1].Start
		}
		if end == nil {
			end = lineEnd
		}
		if end == nil {
			end = nextTimedStart
		}
		if end != nil && i+1 < len(out) && *end > *out[i+1].Start {
			end = out[i+1].Start
		}
		if end == nil || *end < *out[i].Start {
			complete = false
			break
		}
		out[i].End = gg.Clone(end)
	}

	if !complete {
		for i := range out {
			out[i].End = nil
		}
	}
	return out
}

func widenLineTiming(line Line, sourceCues []Cue, nextTimedStart *int64) Line {
	var earliestStart *int64
	for _, cue := range line.Cue {
		if cue.Start == nil {
			continue
		}
		if earliestStart == nil || *cue.Start < *earliestStart {
			earliestStart = cue.Start
		}
	}
	if earliestStart != nil && (line.Start == nil || *earliestStart < *line.Start) {
		line.Start = gg.Clone(earliestStart)
	}

	// Only an explicit terminal cue end is exact enough to become Line.End.
	groups := make(map[string][]Cue)
	for _, cue := range sourceCues {
		groups[cue.AgentID] = append(groups[cue.AgentID], cue)
	}
	var latestExactEnd *int64
	for _, group := range groups {
		terminal := group[len(group)-1]
		if terminal.Start == nil {
			continue
		}
		isNextLineFallback := line.End == nil && nextTimedStart != nil && terminal.End != nil && *terminal.End == *nextTimedStart
		if terminal.End != nil && !isNextLineFallback && *terminal.End >= *terminal.Start && (latestExactEnd == nil || *terminal.End > *latestExactEnd) {
			latestExactEnd = terminal.End
		}
	}
	if latestExactEnd != nil && (line.End == nil || *latestExactEnd > *line.End) {
		line.End = gg.Clone(latestExactEnd)
	}
	if line.Start != nil && line.End != nil && *line.End < *line.Start {
		line.End = nil
	}
	if line.Start == nil {
		line.End = nil
	}
	return line
}

// normalizeLineTiming is used while TTML is still assembling agent metadata.
// Full cue validation and agent pruning happen when the complete Lyrics value
// is passed through NormalizeLyrics.
func normalizeLineTiming(line Line) Line {
	out := cloneLyricLine(line)
	if out.Start != nil && out.End != nil && *out.End < *out.Start {
		out.End = nil
	}
	return widenLineTiming(out, out.Cue, nil)
}

func pruneLyricAgents(lines []Line, agents []Agent) []Agent {
	used := make(map[string]struct{})
	for _, line := range lines {
		for _, cue := range line.Cue {
			if cue.AgentID != "" {
				used[cue.AgentID] = struct{}{}
			}
		}
	}
	if len(used) == 0 {
		return nil
	}

	out := make([]Agent, 0, len(used))
	seen := make(map[string]struct{}, len(used))
	for _, agent := range agents {
		id := strings.TrimSpace(agent.ID)
		if _, ok := used[id]; !ok {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		agent.ID = id
		out = append(out, agent)
		seen[id] = struct{}{}
	}
	return out
}

// NormalizeCueEnds retains the v2 response helper contract: resolve missing
// ends from the next cue or the supplied exact/fallback end without mutating
// the caller. Model parsing should use NormalizeLyrics instead.
func NormalizeCueEnds(cues []Cue, fallbackEnd *int64) []Cue {
	if len(cues) == 0 {
		return cues
	}

	out := slices.Clone(cues)
	for i := range out {
		end := out[i].End
		if end == nil && i+1 < len(out) && out[i+1].Start != nil {
			end = out[i+1].Start
		}
		if end == nil {
			end = fallbackEnd
		}
		if end != nil && i+1 < len(out) && out[i+1].Start != nil && *end > *out[i+1].Start {
			end = out[i+1].Start
		}
		if end != nil && out[i].Start != nil && *end < *out[i].Start {
			end = out[i].Start
		}
		out[i].End = gg.Clone(end)
	}

	for i := range out {
		if out[i].End != nil {
			continue
		}
		for j := range out {
			out[j].End = nil
		}
		break
	}
	return out
}
