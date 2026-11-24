package model

import (
	"slices"
	"strings"

	"github.com/navidrome/navidrome/conf"
)

// LyricsFileSuffixes returns the set of external lyrics file suffixes (like ".lrc") listed in
// LyricsPriority, lowercased and sorted, so it does not depend on their configured order or case.
func LyricsFileSuffixes() []string {
	var res []string
	for p := range strings.SplitSeq(conf.Server().LyricsPriority, ",") {
		if p = strings.TrimSpace(p); strings.HasPrefix(p, ".") {
			res = append(res, strings.ToLower(p))
		}
	}
	slices.Sort(res)
	return slices.Compact(res)
}
