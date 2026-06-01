package model

import (
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/utils/str"
	"gopkg.in/yaml.v3"
)

// ParseLyricsfile parses a LRCLIB Lyricsfile YAML document
// (see https://github.com/tranxuanthang/lrcget/blob/main/LYRICSFILE_CONCEPT.md)
// into a model.LyricList containing a single main Lyrics entry. Returns
// (nil, nil) when the input parses as YAML but does not look like a
// Lyricsfile (no version, no metadata, no lines, no instrumental flag)
//
// When the source contains per-word timing via lines[].words[], each word
// becomes a model.Cue with inclusive UTF-8 byte offsets into Line.Value, and
// overlapping lines are attributed to synthetic voice agents via lowest-free
// voice ID assignment so the OpenSubsonic v2 enhanced response can split
// parallel vocals.
func ParseLyricsfile(text string) (LyricList, error) {
	var doc lyricsfileDocument
	dec := yaml.NewDecoder(strings.NewReader(text))
	dec.KnownFields(false)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("not a valid Lyricsfile YAML: %w", err)
	}

	if doc.Version == "" && doc.Metadata.isEmpty() && len(doc.Lines) == 0 {
		return nil, nil
	}

	lyrics := Lyrics{
		DisplayArtist: str.SanitizeText(doc.Metadata.Artist),
		DisplayTitle:  str.SanitizeText(doc.Metadata.Title),
		Lang:          normalizeLyricsfileLang(doc.Metadata.Language),
		Kind:          lyricsfileKindMain,
	}
	if doc.Metadata.OffsetMs != 0 {
		off := doc.Metadata.OffsetMs
		lyrics.Offset = &off
	}

	if doc.Metadata.Instrumental || len(doc.Lines) == 0 {
		return LyricList{NormalizeLyrics(lyrics)}, nil
	}

	lines, agents := buildLyricsfileLines(doc.Lines)
	lyrics.Line = lines
	lyrics.Agents = agents
	lyrics.Synced = true
	return LyricList{NormalizeLyrics(lyrics)}, nil
}

const lyricsfileKindMain = "main"

type lyricsfileDocument struct {
	Version  string                `yaml:"version"`
	Metadata lyricsfileMetadata    `yaml:"metadata"`
	Lines    []lyricsfileLineEntry `yaml:"lines"`
	Plain    string                `yaml:"plain"`
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

// buildLyricsfileLines converts YAML line entries to model.Line entries with
// per-cue AgentIDs assigned by streaming overlap clustering (lowest-free
// voice ID). The Agents slice is emitted only when at least one cue carries
// attribution AND more than one voice is used; otherwise AgentIDs are
// stripped so the wire shape stays simple per the OpenSubsonic spec rule
// "agents should not be emitted without cueLine data".
func buildLyricsfileLines(entries []lyricsfileLineEntry) ([]Line, []Agent) {
	if len(entries) == 0 {
		return nil, nil
	}

	// Resolved end timestamps per entry: explicit end_ms if present, otherwise
	// the next entry's start. The last entry's end stays nil.
	ends := make([]*int64, len(entries))
	for i := range entries {
		if entries[i].EndMs != nil {
			v := *entries[i].EndMs
			ends[i] = &v
		} else if i+1 < len(entries) {
			v := entries[i+1].StartMs
			ends[i] = &v
		}
	}

	active := map[int]int64{}
	maxVoice := -1
	anyCues := false
	lines := make([]Line, 0, len(entries))

	for i, entry := range entries {
		for vID, vEnd := range active {
			if vEnd <= entry.StartMs {
				delete(active, vID)
			}
		}

		voiceID := 0
		for {
			if _, busy := active[voiceID]; !busy {
				break
			}
			voiceID++
		}
		if voiceID > maxVoice {
			maxVoice = voiceID
		}

		agentID := fmt.Sprintf("voice-%d", voiceID)
		cues, value := wordsToLineCues(entry, agentID)
		if len(cues) > 0 {
			anyCues = true
		}

		startMs := entry.StartMs
		line := Line{
			Start: &startMs,
			End:   ends[i],
			Value: value,
			Cue:   cues,
		}
		lines = append(lines, line)

		var endMs int64
		if ends[i] != nil {
			endMs = *ends[i]
		} else {
			endMs = entry.StartMs
		}
		active[voiceID] = endMs
	}

	// Monophonic source, or attribution that has nowhere to land: emit no
	// agents and strip per-cue AgentIDs to keep the wire shape simple.
	if maxVoice <= 0 || !anyCues {
		for i := range lines {
			for j := range lines[i].Cue {
				lines[i].Cue[j].AgentID = ""
			}
		}
		return lines, nil
	}

	agents := make([]Agent, 0, maxVoice+1)
	for v := 0; v <= maxVoice; v++ {
		role := "voice"
		if v == 0 {
			role = "main"
		}
		agents = append(agents, Agent{
			ID:   fmt.Sprintf("voice-%d", v),
			Role: role,
		})
	}
	return lines, agents
}

// wordsToLineCues converts a Lyricsfile line entry's words[] into model.Cue
// entries with inclusive UTF-8 byte offsets into the reconstructed line
// value. The line value is built from cue text concatenation rather than
// trusting entry.Text, because the Lyricsfile spec only requires word.text
// to "approximate" line.text - byte offsets must always land inside
// Line.Value.
func wordsToLineCues(entry lyricsfileLineEntry, agentID string) ([]Cue, string) {
	if len(entry.Words) == 0 {
		return nil, str.SanitizeText(entry.Text)
	}

	var sb strings.Builder
	for _, w := range entry.Words {
		sb.WriteString(w.Text)
	}
	lineValue := sb.String()

	cues := make([]Cue, len(entry.Words))
	cursor := 0
	for i, w := range entry.Words {
		valueBytes := len(w.Text)
		bs := cursor
		be := bs
		if valueBytes > 0 {
			be = bs + valueBytes - 1
			cursor = be + 1
		}

		s := w.StartMs
		cue := Cue{
			Start:     &s,
			Value:     w.Text,
			ByteStart: bs,
			ByteEnd:   be,
			AgentID:   agentID,
		}
		if w.EndMs != nil {
			e := *w.EndMs
			cue.End = &e
		}
		cues[i] = cue
	}

	for i := 0; i < len(cues)-1; i++ {
		if cues[i].End == nil && cues[i+1].Start != nil {
			v := *cues[i+1].Start
			cues[i].End = &v
		}
	}
	return cues, lineValue
}

func normalizeLyricsfileLang(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		return "xxx"
	}
	return language
}
