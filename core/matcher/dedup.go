package matcher

import (
	"math"
	"slices"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

// fuzzyDedupTitleThreshold and fuzzyDedupDurationToleranceSec are deliberately stricter than the
// external-song matching threshold in findBestMatch - here we're clustering tracks we already
// know are correct library entries against each other, not resolving an uncertain external
// description, so false positives (collapsing two genuinely different songs) are worse than
// false negatives (missing a duplicate).
const (
	fuzzyDedupTitleThreshold       = 0.92
	fuzzyDedupDurationToleranceSec = 5.0
)

// DeduplicateMediaFiles collapses likely-same-recording tracks (e.g. the same song appearing on
// both a studio album and a "Best Of" compilation, or a live + studio version sharing metadata)
// down to one representative per recording (the highest-bitrate copy), preserving the input's
// relative order. Clustering priority mirrors Matcher's external-song matching: exact MusicBrainz
// Recording ID, then exact ISRC, then fuzzy title+artist+duration for anything left unmatched by
// an exact identifier.
//
// Unlike Matcher.MatchSongs/MatchSongsIndexed (which resolve external agents.Song descriptions
// against the DB), this operates directly on an already-fetched model.MediaFiles slice - no DB
// round-trip, no external input shape to adapt.
func DeduplicateMediaFiles(files model.MediaFiles) model.MediaFiles {
	if len(files) <= 1 {
		return files
	}

	assigned := make([]bool, len(files))
	var clusters [][]int // each entry is a list of indices into files

	// Tier 1: exact MBID.
	byMBID := map[string][]int{}
	for i, f := range files {
		if id := f.MbzRecordingID; id != "" {
			byMBID[id] = append(byMBID[id], i)
		}
	}
	for _, idxs := range byMBID {
		clusters = append(clusters, idxs)
		for _, i := range idxs {
			assigned[i] = true
		}
	}

	// Tier 2: exact ISRC, among tracks not already clustered by MBID.
	byISRC := map[string][]int{}
	for i, f := range files {
		if assigned[i] {
			continue
		}
		for _, isrc := range f.Tags.Values(model.TagISRC) {
			byISRC[isrc] = append(byISRC[isrc], i)
		}
	}
	seenByISRC := map[int]bool{}
	for _, idxs := range byISRC {
		var cluster []int
		for _, i := range idxs {
			if !seenByISRC[i] {
				cluster = append(cluster, i)
				seenByISRC[i] = true
				assigned[i] = true
			}
		}
		if len(cluster) > 0 {
			clusters = append(clusters, cluster)
		}
	}

	// Tier 3: fuzzy title+artist+duration, among whatever's left. Each remaining track either
	// joins the first existing fuzzy cluster it's close enough to, or starts a new one.
	type fuzzyEntry struct {
		idx      int
		title    string
		artist   string
		duration float32
	}
	var fuzzyClusters [][]fuzzyEntry
	for i, f := range files {
		if assigned[i] {
			continue
		}
		entry := fuzzyEntry{
			idx:      i,
			title:    str.SanitizeFieldForSorting(f.Title),
			artist:   str.SanitizeFieldForSortingNoArticle(f.Artist),
			duration: f.Duration,
		}
		placed := false
		for c, cluster := range fuzzyClusters {
			rep := cluster[0]
			if entry.artist == rep.artist &&
				similarityRatio(entry.title, rep.title) >= fuzzyDedupTitleThreshold &&
				math.Abs(float64(entry.duration)-float64(rep.duration)) <= fuzzyDedupDurationToleranceSec {
				fuzzyClusters[c] = append(cluster, entry)
				placed = true
				break
			}
		}
		if !placed {
			fuzzyClusters = append(fuzzyClusters, []fuzzyEntry{entry})
		}
	}
	for _, cluster := range fuzzyClusters {
		idxs := make([]int, len(cluster))
		for j, e := range cluster {
			idxs[j] = e.idx
		}
		clusters = append(clusters, idxs)
	}

	// Pick one representative per cluster (highest bitrate), then order the result by each
	// cluster's earliest-appearing member so it stays close to the input's original order.
	type repEntry struct {
		firstIdx int
		mf       model.MediaFile
	}
	reps := make([]repEntry, 0, len(clusters))
	for _, idxs := range clusters {
		best := idxs[0]
		first := idxs[0]
		for _, i := range idxs {
			if files[i].BitRate > files[best].BitRate {
				best = i
			}
			if i < first {
				first = i
			}
		}
		reps = append(reps, repEntry{firstIdx: first, mf: files[best]})
	}
	slices.SortFunc(reps, func(a, b repEntry) int { return a.firstIdx - b.firstIdx })

	result := make(model.MediaFiles, len(reps))
	for i, r := range reps {
		result[i] = r.mf
	}
	return result
}
