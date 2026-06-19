package model

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/utils/gg"
)

type Cue struct {
	Start     *int64 `structs:"start,omitempty"   json:"start,omitempty"`
	End       *int64 `structs:"end,omitempty"     json:"end,omitempty"`
	Value     string `structs:"value"             json:"value"`
	ByteStart int    `structs:"byteStart"         json:"byteStart"`
	ByteEnd   int    `structs:"byteEnd"           json:"byteEnd"`
	AgentID   string `structs:"agentId,omitempty" json:"agentId,omitempty"`
}

type Agent struct {
	ID   string `structs:"id"             json:"id"`
	Role string `structs:"role"           json:"role"`
	Name string `structs:"name,omitempty" json:"name,omitempty"`
}

type Line struct {
	Start *int64 `structs:"start,omitempty" json:"start,omitempty"`
	End   *int64 `structs:"end,omitempty"   json:"end,omitempty"`
	Value string `structs:"value"           json:"value"`
	Cue   []Cue  `structs:"cue,omitempty"   json:"cue,omitempty"`
}

type Lyrics struct {
	DisplayArtist string  `structs:"displayArtist,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string  `structs:"displayTitle,omitempty"  json:"displayTitle,omitempty"`
	Kind          string  `structs:"kind,omitempty"          json:"kind,omitempty"`
	Lang          string  `structs:"lang"                    json:"lang"`
	Agents        []Agent `structs:"agents,omitempty"       json:"agents,omitempty"`
	Line          []Line  `structs:"line"                    json:"line"`
	Offset        *int64  `structs:"offset,omitempty"        json:"offset,omitempty"`
	Synced        bool    `structs:"synced"                  json:"synced"`
}

// Lyric kinds, as defined by the OpenSubsonic songLyrics v2 contract. These are
// the canonical wire values; keep them in sync with the spec.
const (
	LyricKindMain          = "main"
	LyricKindTranslation   = "translation"
	LyricKindPronunciation = "pronunciation"
)

func (l Lyrics) IsEmpty() bool {
	return len(l.Line) == 0
}

// IsMainKind reports whether the lyric is the main track. A blank kind is an
// untyped (single-track) lyric, which the contract treats as main.
func (l Lyrics) IsMainKind() bool {
	return l.EffectiveKind() == LyricKindMain
}

// EffectiveKind returns the lyric kind, defaulting to LyricKindMain when blank.
// A blank kind means an untyped (single-track) lyric, which the contract treats
// as main.
func (l Lyrics) EffectiveKind() string {
	if strings.TrimSpace(l.Kind) == "" {
		return LyricKindMain
	}
	return l.Kind
}

type LyricList []Lyrics

// MarshalJSON keeps the lyrics column invariant: empty/nil serializes to [], never null.
func (ll LyricList) MarshalJSON() ([]byte, error) {
	if len(ll) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]Lyrics(ll))
}

// Main returns the main-kind lyric, falling back to the first entry so untyped
// lyrics still resolve. The bool is false only when the list is empty. It is
// used to surface a single lyric through the plain-text legacy getLyrics
// endpoint, which has no notion of translation/pronunciation tracks.
func (ll LyricList) Main() (Lyrics, bool) {
	if len(ll) == 0 {
		return Lyrics{}, false
	}
	for _, l := range ll {
		if l.IsMainKind() {
			return l, true
		}
	}
	return ll[0], true
}

func NormalizeLyrics(lyrics Lyrics) Lyrics {
	lyrics.Line = NormalizeCueLines(lyrics.Line)
	if len(lyrics.Agents) == 0 {
		lyrics.Agents = nil
	}
	return lyrics
}

func NormalizeCueLines(lines []Line) []Line {
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

func NormalizeLineTiming(line Line) Line {
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
	line.Cue = NormalizeCueEnds(line.Cue, fallbackEnd)
	return NormalizeLineTiming(line)
}

// NormalizeCueEnds resolves missing cue end times within a single ordered cue
// group: each end is filled from the next cue's start, then from fallbackEnd,
// and is clamped so it never precedes the cue's own start nor overruns the next
// cue. End times are all-or-none — if any cue still lacks an end afterwards, all
// ends in the group are cleared. The input slice is never mutated.
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
