package model_test

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/model"
)

// This is also the output asserted by the UI exporter's background-voice
// fixture. Re-import must preserve its lanes and terminal word timings.
func TestExportedELRCRetainsBackgroundVoiceOnImport(t *testing.T) {
	const exported = "[00:52.00]<00:52.00>Main <00:52.50>line<00:53.00>\n[bg:<00:53.00>Okay<00:53.50>]\n[00:55.00]Next"
	lyrics, err := model.ParseLyrics(context.Background(), ".elrc", "eng", []byte(exported))
	if err != nil || len(lyrics) != 1 || len(lyrics[0].Line) != 2 {
		t.Fatalf("export could not be re-imported: lyrics=%v error=%v", lyrics, err)
	}
	line := lyrics[0].Line[0]
	if line.Value != "Main line Okay" || len(line.Cue) != 3 || len(lyrics[0].Agents) != 2 {
		t.Fatalf("background lane was lost: line=%#v agents=%#v", line, lyrics[0].Agents)
	}
	for index, end := range []int64{52500, 53000, 53500} {
		if line.Cue[index].End == nil || *line.Cue[index].End != end {
			t.Fatalf("cue %d end=%v, want %d", index, line.Cue[index].End, end)
		}
	}
	if line.Cue[2].AgentID == line.Cue[0].AgentID || lyrics[0].Agents[1].Role != "bg" {
		t.Fatal("background voice must retain a separate background agent")
	}
}

// Matches the UI export fixture with line-timed voices and no word cues.
func TestExportedTTMLRetainsSeparateVoiceTimingOnImport(t *testing.T) {
	const exported = `<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata"><head><metadata><ttm:agent xml:id="v1" type="person"/><ttm:agent xml:id="v2" type="person"/></metadata></head><body><div><p begin="00:00:01.000" end="00:00:08.000"><span begin="00:00:01.000" end="00:00:03.000" ttm:agent="v1">Lead</span> <span begin="00:00:06.000" end="00:00:08.000" ttm:agent="v2" ttm:role="x-bg">Echo</span></p></div></body></tt>`
	lyrics, err := model.ParseLyrics(context.Background(), ".ttml", "eng", []byte(exported))
	if err != nil || len(lyrics) != 1 || len(lyrics[0].Line) != 1 {
		t.Fatalf("export could not be re-imported: lyrics=%v error=%v", lyrics, err)
	}
	cues := lyrics[0].Line[0].Cue
	if len(cues) != 2 {
		t.Fatalf("voice cues = %#v, want two timed voices", cues)
	}
	for index, window := range [][2]int64{{1000, 3000}, {6000, 8000}} {
		cue := cues[index]
		if cue.Start == nil || cue.End == nil || *cue.Start != window[0] || *cue.End != window[1] || cue.AgentID == "" {
			t.Fatalf("voice %d = %#v, want attributed timing %v", index, cue, window)
		}
	}
}
