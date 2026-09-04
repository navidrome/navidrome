package artwork

import (
	"slices"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

// Result states the verdict of the walk. A skipped or failed external tier, or a local candidate
// that would not open, leaves the outcome unknown: nothing observed that there is no artwork.
func Result(source string, steps []TraceStep) string {
	if source != "" {
		for _, s := range steps {
			if s.Outcome == OutcomeHit {
				break
			}
			// An external winner discards the earlier error, so the resolver settles it with no retry.
			if s.Outcome == OutcomeError && strings.HasPrefix(s.Candidate, ExternalPrefix) &&
				!strings.HasPrefix(source, ExternalPrefix) {
				return "resolved from " + source +
					" (indeterminate: a higher-priority external lookup failed; this may resolve differently on a retry)"
			}
		}
		return "resolved from " + source
	}
	for _, s := range steps {
		switch {
		case s.Outcome == OutcomeError && strings.HasPrefix(s.Candidate, ExternalPrefix):
			return "indeterminate (an external lookup failed; the item may resolve on a later attempt)"
		case s.Outcome == OutcomeError, s.Outcome == OutcomeUnreadable:
			return "indeterminate (a candidate was found but could not be processed; the worker retries rather than settling absent)"
		}
	}
	return "not resolved"
}

// ConfigFor names the setting that decides where a kind's artwork comes from, and its value.
func ConfigFor(kind model.Kind) (setting, value string) {
	switch kind {
	case model.KindArtistArtwork:
		return "ArtistArtPriority", conf.Server.ArtistArtPriority
	case model.KindAlbumArtwork:
		return "CoverArtPriority", conf.Server.CoverArtPriority
	case model.KindDiscArtwork:
		return "DiscArtPriority", conf.Server.DiscArtPriority
	case model.KindMediaFileArtwork:
		return "EnableMediaFileCoverArt", strconv.FormatBool(conf.Server.EnableMediaFileCoverArt)
	}
	return "", ""
}

// ImageAgentNames names the agents that can actually supply an image for kind.
func ImageAgentNames(ag *agents.Agents, kind model.Kind) []string {
	if kind == model.KindArtistArtwork {
		return slice.Map(ag.ArtistImageAgents(), func(a agents.ArtistImageAgent) string { return a.Name })
	}
	return slice.Map(ag.AlbumImageAgents(), func(a agents.AlbumImageAgent) string { return a.Name })
}

// FormatAgents accounts for every configured agent: one that cannot be constructed never reaches
// the chain, so the raw list alone overstates it. unavailableNote is appended only if some are.
func FormatAgents(configured string, available []string, unavailableNote string) string {
	if strings.TrimSpace(configured) == "" {
		return "(none)"
	}
	var unavailable bool
	names := slice.Map(strings.Split(configured, ","), func(name string) string {
		name = strings.TrimSpace(name)
		if slices.Contains(available, name) {
			return name
		}
		unavailable = true
		return name + "*"
	})
	line := strings.Join(names, ", ")
	if unavailable {
		line += unavailableNote
	}
	return line
}
