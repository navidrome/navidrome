package subsonic

import (
	"slices"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

// agentRoleMain is the OpenSubsonic agent role that marks the primary vocal
// layer; its cue line is emitted before other agents sharing the same index.
const agentRoleMain = "main"

func buildLyricsList(mf *model.MediaFile, lyricsList model.LyricList, enhanced bool) *responses.LyricsList {
	filtered := lyricsList
	if !enhanced {
		// Without enhanced, only return main-kind entries (a blank kind is main).
		filtered = nil
		for _, l := range lyricsList {
			if model.LyricKindOrMain(l.Kind) == model.LyricKindMain {
				filtered = append(filtered, l)
			}
		}
	}

	lyricList := make(responses.StructuredLyrics, len(filtered))
	for i, lyrics := range filtered {
		lyricList[i] = buildStructuredLyric(mf, lyrics, enhanced)
	}
	return &responses.LyricsList{StructuredLyrics: lyricList}
}

// mainKindLyric returns the main-kind lyric to surface through the plain-text
// legacy getLyrics endpoint, which has no notion of translation/pronunciation
// tracks. It falls back to the first entry so untyped lyrics still resolve.
func mainKindLyric(lyricsList model.LyricList) (model.Lyrics, bool) {
	if len(lyricsList) == 0 {
		return model.Lyrics{}, false
	}
	for _, l := range lyricsList {
		if model.LyricKindOrMain(l.Kind) == model.LyricKindMain {
			return l, true
		}
	}
	return lyricsList[0], true
}

func buildStructuredLyric(mf *model.MediaFile, lyrics model.Lyrics, enhanced bool) responses.StructuredLyric {
	agents := newLyricAgents(lyrics.Agents)

	lines := make([]responses.Line, len(lyrics.Line))
	var cueLines []responses.CueLine
	for i, line := range lyrics.Line {
		lines[i] = responses.Line{Start: line.Start, Value: line.Value}
		if enhanced && len(line.Cue) > 0 {
			cueLines = append(cueLines, buildCueLines(line, int32(i), agents)...)
		}
	}

	structured := responses.StructuredLyric{
		DisplayArtist: lyrics.DisplayArtist,
		DisplayTitle:  lyrics.DisplayTitle,
		Lang:          lyrics.Lang,
		Line:          lines,
		CueLine:       cueLines,
		Offset:        lyrics.Offset,
		Synced:        lyrics.Synced,
	}

	if enhanced {
		structured.Kind = model.LyricKindOrMain(lyrics.Kind)
		if len(cueLines) > 0 && len(agents.response) > 0 {
			structured.Agents = agents.response
		}
	}

	if structured.DisplayArtist == "" {
		structured.DisplayArtist = mf.Artist
	}
	if structured.DisplayTitle == "" {
		structured.DisplayTitle = mf.Title
	}
	return structured
}

// lyricAgents indexes a lyric's agents by ID so cue lines can be ordered and
// the response agent list reused without rescanning the slice per line.
type lyricAgents struct {
	orderByID map[string]int
	roleByID  map[string]string
	response  []responses.Agent
}

func newLyricAgents(agents []model.Agent) lyricAgents {
	a := lyricAgents{
		orderByID: make(map[string]int, len(agents)),
		roleByID:  make(map[string]string, len(agents)),
		response:  make([]responses.Agent, 0, len(agents)),
	}
	for i, agent := range agents {
		a.orderByID[agent.ID] = i
		a.roleByID[agent.ID] = agent.Role
		a.response = append(a.response, responses.Agent{ID: agent.ID, Role: agent.Role, Name: agent.Name})
	}
	return a
}

// buildCueLines splits a line's cues by agent and emits one CueLine per agent,
// ordered main-role first then by the agent's declared order.
func buildCueLines(line model.Line, index int32, agents lyricAgents) []responses.CueLine {
	agentOrder := make([]string, 0, 2)
	cuesByAgent := make(map[string][]model.Cue)
	for _, cue := range line.Cue {
		if cue.Start == nil {
			continue
		}
		agentID := strings.TrimSpace(cue.AgentID)
		if _, exists := cuesByAgent[agentID]; !exists {
			agentOrder = append(agentOrder, agentID)
		}
		cuesByAgent[agentID] = append(cuesByAgent[agentID], cue)
	}

	sort.SliceStable(agentOrder, func(i, j int) bool {
		return agents.less(agentOrder[i], agentOrder[j], i, j)
	})

	cueLines := make([]responses.CueLine, 0, len(agentOrder))
	for _, agentID := range agentOrder {
		cueLine := responses.CueLine{
			Index: index,
			Start: line.Start,
			End:   line.End,
			Value: line.Value,
			Cue:   buildLyricCues(cuesByAgent[agentID], line.End),
		}
		if agentID != "" {
			cueLine.AgentID = agentID
		}
		cueLines = append(cueLines, cueLine)
	}
	return cueLines
}

// less orders two agent IDs: the main role wins, then the declared agent order,
// then known-before-unknown, then the original encounter order (origI/origJ).
func (a lyricAgents) less(left, right string, origI, origJ int) bool {
	leftMain := a.roleByID[left] == agentRoleMain
	rightMain := a.roleByID[right] == agentRoleMain
	if leftMain != rightMain {
		return leftMain
	}

	leftOrder, leftOK := a.orderByID[left]
	rightOrder, rightOK := a.orderByID[right]
	if leftOK && rightOK && leftOrder != rightOrder {
		return leftOrder < rightOrder
	}
	if leftOK != rightOK {
		return leftOK
	}
	return origI < origJ
}

func buildLyricCues(cues []model.Cue, lineEnd *int64) []responses.LyricCue {
	if len(cues) == 0 {
		return nil
	}

	// Only resolve end times when at least one cue carries one; otherwise the
	// group is start-only and must stay that way.
	hasAnyEnd := slices.ContainsFunc(cues, func(c model.Cue) bool { return c.End != nil })
	if hasAnyEnd {
		cues = model.NormalizeCueEnds(cues, lineEnd)
	}

	out := make([]responses.LyricCue, 0, len(cues))
	for i := range cues {
		if cues[i].Start == nil {
			continue
		}
		cue := responses.LyricCue{
			Start:     *cues[i].Start,
			Value:     cues[i].Value,
			ByteStart: cues[i].ByteStart,
			ByteEnd:   cues[i].ByteEnd,
		}
		if hasAnyEnd {
			cue.End = cues[i].End
		}
		out = append(out, cue)
	}
	return out
}
