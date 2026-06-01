package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/utils/str"
	"gopkg.in/yaml.v3"
)

// ParseLyricsfile parses a Lyricsfile YAML document
// (see https://github.com/tranxuanthang/lrcget/blob/main/LYRICSFILE_CONCEPT.md) and produces
// the canonical Lyrics representation. Returns a non-nil error if the input
// is not a valid YAML document or does not appear to be a Lyricsfile
func ParseLyricsfile(text string) (*Lyrics, error) {
	var doc lyricsfileDocument
	dec := yaml.NewDecoder(strings.NewReader(text))
	dec.KnownFields(false)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("not a valid YAML document: %w", err)
	}

	// Shape validation: a Lyricsfile must have at least version + metadata.
	// We accept slightly relaxed inputs (no version) only when both metadata
	// and lines are populated
	if doc.Version == "" && doc.Metadata.isEmpty() && len(doc.Lines) == 0 && !doc.Metadata.Instrumental {
		return nil, errors.New("YAML document does not appear to be a Lyricsfile (missing version, metadata, and lines)")
	}

	lyrics := &Lyrics{
		DisplayArtist: str.SanitizeText(doc.Metadata.Artist),
		DisplayTitle:  str.SanitizeText(doc.Metadata.Title),
		Lang:          str.SanitizeText(doc.Metadata.Language),
		Synced:        false,
	}
	if lyrics.Lang == "" {
		lyrics.Lang = "xxx"
	}
	if doc.Metadata.OffsetMs != 0 {
		off := doc.Metadata.OffsetMs
		lyrics.Offset = &off
	}

	if doc.Metadata.Instrumental {
		// Instrumental tracks are represented as an empty cue list with
		// Synced=false. Clients infer instrumental status elsewhere.
		return lyrics, nil
	}

	if len(doc.Lines) == 0 {
		return lyrics, nil
	}

	cueLines, agents := buildLyricsfileCueLines(doc.Lines)
	lyrics.CueLine = cueLines
	lyrics.Agents = agents
	lyrics.Synced = true
	return lyrics, nil
}

type lyricsfileDocument struct {
	Version  string                `yaml:"version"`
	Metadata lyricsfileMetadata    `yaml:"metadata"`
	Lines    []lyricsfileLineEntry `yaml:"lines"`
	Plain    string                `yaml:"plain"`
	Extra    map[string]yaml.Node  `yaml:",inline"`
}

type lyricsfileMetadata struct {
	Title        string `yaml:"title"`
	Artist       string `yaml:"artist"`
	Album        string `yaml:"album"`
	DurationMs   int64  `yaml:"duration_ms"`
	OffsetMs     int64  `yaml:"offset_ms"`
	Language     string `yaml:"language"`
	Instrumental bool   `yaml:"instrumental"`
}

func (m lyricsfileMetadata) isEmpty() bool {
	return m.Title == "" && m.Artist == "" && m.Album == "" &&
		m.DurationMs == 0 && m.OffsetMs == 0 && m.Language == "" && !m.Instrumental
}

type lyricsfileLineEntry struct {
	Text    string                `yaml:"text"`
	StartMs int64                 `yaml:"start_ms"`
	EndMs   *int64                `yaml:"end_ms"`
	Words   []lyricsfileWordEntry `yaml:"words"`
}

type lyricsfileWordEntry struct {
	Text    string `yaml:"text"`
	StartMs int64  `yaml:"start_ms"`
	EndMs   *int64 `yaml:"end_ms"`
}

// buildLyricsfileCueLines runs the streaming overlap-clustering algorithm over
// the parsed Lyricsfile lines and produces:
//   - one CueLine per line, with Index reflecting cluster membership and
//     AgentID assigned via the lowest-free voice rule.
//   - the synthetic Agents slice. When the song has no overlapping vocals
//     (only voice-0 ever used), the Agents slice and per-line AgentID are
//     left empty so the wire format stays simple.
func buildLyricsfileCueLines(entries []lyricsfileLineEntry) ([]CueLine, []Agent) {
	cueLines := make([]CueLine, 0, len(entries))

	// Resolved end timestamps for each entry (handles missing end_ms by
	// inferring from the next line's start; the last line stays open).
	ends := make([]*int64, len(entries))
	for i := range entries {
		if entries[i].EndMs != nil {
			endCopy := *entries[i].EndMs
			ends[i] = &endCopy
		} else if i+1 < len(entries) {
			startCopy := entries[i+1].StartMs
			ends[i] = &startCopy
		}
	}

	// active maps voice ID -> end_ms of the line currently held by that voice.
	active := map[int]int64{}
	currentIndex := -1
	maxVoiceUsed := -1
	hasPriorLine := false

	for i, entry := range entries {
		// Prune voices whose end <= this line's start.
		for v, e := range active {
			if e <= entry.StartMs {
				delete(active, v)
			}
		}

		// New cluster when the active set is empty before adding this line.
		if !hasPriorLine || len(active) == 0 {
			currentIndex++
		}

		// Lowest-free voice ID.
		voiceID := 0
		for {
			if _, busy := active[voiceID]; !busy {
				break
			}
			voiceID++
		}
		if voiceID > maxVoiceUsed {
			maxVoiceUsed = voiceID
		}

		startCopy := entry.StartMs
		cues := wordsToCues(entry.Words)
		// Cue.byteStart/byteEnd are positions in cueLine.value per the
		// OpenSubsonic v2 spec, so when cues are present we reconstruct value
		// from them rather than trust entry.Text (the Lyricsfile spec only
		// requires word.text to "approximate" line.text).
		value := entry.Text
		if len(cues) > 0 {
			var sb strings.Builder
			for _, c := range cues {
				sb.WriteString(c.Value)
			}
			value = sb.String()
		}
		cl := CueLine{
			Index:   currentIndex,
			Start:   &startCopy,
			End:     ends[i],
			Value:   value,
			AgentID: fmt.Sprintf("voice-%d", voiceID),
			Cue:     cues,
		}
		cueLines = append(cueLines, cl)

		// Track this voice as active until its resolved end.
		var endMs int64
		if ends[i] != nil {
			endMs = *ends[i]
		} else {
			// No known end; treat as immediately freed so future lines aren't
			// blocked. The voice continues to render based on its Start alone.
			endMs = entry.StartMs
		}
		active[voiceID] = endMs
		hasPriorLine = true
	}

	// If we never used more than voice-0, the song is monophonic. Strip the
	// AgentID fields and return no Agents slice for a clean wire shape.
	if maxVoiceUsed <= 0 {
		for i := range cueLines {
			cueLines[i].AgentID = ""
		}
		return cueLines, nil
	}

	agents := make([]Agent, 0, maxVoiceUsed+1)
	for v := 0; v <= maxVoiceUsed; v++ {
		a := Agent{ID: fmt.Sprintf("voice-%d", v)}
		if v == 0 {
			a.Role = string(LyricKindMain)
		}
		agents = append(agents, a)
	}
	return cueLines, agents
}

// wordsToCues converts Lyricsfile word entries into model Cues. Cue.End is
// taken from word.end_ms when present; otherwise inferred from the next word's
// start (last word's End stays nil if it has no explicit end_ms).
func wordsToCues(words []lyricsfileWordEntry) []Cue {
	if len(words) == 0 {
		return nil
	}
	cues := make([]Cue, len(words))
	for i, w := range words {
		startCopy := w.StartMs
		cues[i].Start = &startCopy
		cues[i].Value = w.Text
		if w.EndMs != nil {
			endCopy := *w.EndMs
			cues[i].End = &endCopy
		}
	}
	for i := 0; i < len(cues)-1; i++ {
		if cues[i].End == nil && cues[i+1].Start != nil {
			cues[i].End = cues[i+1].Start
		}
	}
	return cues
}
