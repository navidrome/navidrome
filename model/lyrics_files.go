package model

import (
	"strings"

	"github.com/navidrome/navidrome/conf"
)

// LyricsFileSuffixes returns the external lyrics file suffixes (like ".lrc") listed in LyricsPriority.
func LyricsFileSuffixes() []string {
	var res []string
	for p := range strings.SplitSeq(conf.Server().LyricsPriority, ",") {
		if p = strings.TrimSpace(p); strings.HasPrefix(p, ".") {
			res = append(res, p)
		}
	}
	return res
}
