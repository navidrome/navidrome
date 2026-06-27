package model

import (
	"slices"

	"github.com/navidrome/navidrome/utils/gg"
)

func normalizeLyrics(lyrics Lyrics) Lyrics {
	lyrics.Line = normalizeCueLines(lyrics.Line)
	if len(lyrics.Agents) == 0 {
		lyrics.Agents = nil
	}
	return lyrics
}

func normalizeCueLines(lines []Line) []Line {
	if len(lines) == 0 {
		return lines
	}

	normalized := make([]Line, len(lines))
	copy(normalized, lines)

	for i := range normalized {
		if len(normalized[i].Cue) > 0 {
			normalized[i].Cue = slices.Clone(normalized[i].Cue)
		}

		var fallbackEnd *int64
		if normalized[i].End != nil {
			v := *normalized[i].End
			fallbackEnd = &v
		} else if i+1 < len(normalized) && normalized[i+1].Start != nil {
			v := *normalized[i+1].Start
			fallbackEnd = &v
		}

		normalized[i] = normalizeCueLine(normalized[i], fallbackEnd)
	}

	return normalized
}

func normalizeLineTiming(line Line) Line {
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

func normalizeCueLine(line Line, fallbackEnd *int64) Line {
	if len(line.Cue) == 0 {
		return line
	}
	line.Cue = normalizeCueEndsByAgent(line.Cue, fallbackEnd)
	return normalizeLineTiming(line)
}

// normalizeCueEndsByAgent resolves cue end times independently per agent so that
// background (or other parallel) layers, whose cues interleave with the main
// timeline but are stored together in document order, do not clamp each other's
// ends. Each agent group is normalized in its own document order; results are
// reassembled into the original cue positions.
func normalizeCueEndsByAgent(cues []Cue, fallbackEnd *int64) []Cue {
	groups := make(map[string][]int)
	order := make([]string, 0, 2)
	for i := range cues {
		id := cues[i].AgentID
		if _, ok := groups[id]; !ok {
			order = append(order, id)
		}
		groups[id] = append(groups[id], i)
	}

	// Single agent: the document order already matches the timeline, so the
	// straightforward normalization applies without regrouping.
	if len(order) <= 1 {
		return NormalizeCueEnds(cues, fallbackEnd)
	}

	out := slices.Clone(cues)
	for _, id := range order {
		idxs := groups[id]
		group := make([]Cue, len(idxs))
		for gi, pos := range idxs {
			group[gi] = cues[pos]
		}
		group = NormalizeCueEnds(group, fallbackEnd)
		for gi, pos := range idxs {
			out[pos] = group[gi]
		}
	}
	return out
}

// NormalizeCueEnds resolves missing cue end times within a single ordered cue
// group: each end is filled from the next cue's start, then from fallbackEnd,
// and is clamped so it never precedes the cue's own start nor overruns the next
// cue. End times are all-or-none — if any cue still lacks an end afterwards, all
// ends in the group are cleared. The input slice is never mutated.
//
// Exported because the Subsonic enhanced-lyrics serializer resolves cue ends
// per agent group while building the response; all other normalization is
// package-internal.
func NormalizeCueEnds(cues []Cue, fallbackEnd *int64) []Cue {
	if len(cues) == 0 {
		return cues
	}

	out := slices.Clone(cues)
	for i := range out {
		end := out[i].End
		if end == nil {
			if i+1 < len(out) && out[i+1].Start != nil {
				end = out[i+1].Start
			} else {
				end = fallbackEnd
			}
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
		if out[i].End == nil {
			for j := range out {
				out[j].End = nil
			}
			break
		}
	}
	return out
}
