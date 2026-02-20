package transcode

import (
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/model"
)

// adjustResult represents the outcome of applying a limitation to a transcoded stream value
type adjustResult int

const (
	adjustNone      adjustResult = iota // Value already satisfies the limitation
	adjustAdjusted                      // Value was changed to fit the limitation
	adjustCannotFit                     // Cannot satisfy the limitation (reject this profile)
)

// checkLimitations checks codec profile limitations against source media.
// Returns "" if all limitations pass, or a typed reason string for the first failure.
func checkLimitations(mf *model.MediaFile, sourceBitrate int, limitations []Limitation) string {
	for _, lim := range limitations {
		var ok bool
		var reason string

		switch lim.Name {
		case LimitationAudioChannels:
			ok = checkIntLimitation(mf.Channels, lim.Comparison, lim.Values)
			reason = "audio channels not supported"
		case LimitationAudioSamplerate:
			ok = checkIntLimitation(mf.SampleRate, lim.Comparison, lim.Values)
			reason = "audio samplerate not supported"
		case LimitationAudioBitrate:
			ok = checkIntLimitation(sourceBitrate, lim.Comparison, lim.Values)
			reason = "audio bitrate not supported"
		case LimitationAudioBitdepth:
			ok = checkIntLimitation(mf.BitDepth, lim.Comparison, lim.Values)
			reason = "audio bitdepth not supported"
		case LimitationAudioProfile:
			// TODO: populate source profile when MediaFile has audio profile info
			ok = checkStringLimitation("", lim.Comparison, lim.Values)
			reason = "audio profile not supported"
		default:
			continue
		}

		if !ok && lim.Required {
			return reason
		}
	}
	return ""
}

// applyLimitation adjusts a transcoded stream parameter to satisfy the limitation.
// Returns the adjustment result.
func applyLimitation(sourceBitrate int, lim *Limitation, ts *StreamDetails) adjustResult {
	switch lim.Name {
	case LimitationAudioChannels:
		return applyIntLimitation(lim.Comparison, lim.Values, ts.Channels, func(v int) { ts.Channels = v })
	case LimitationAudioBitrate:
		current := ts.Bitrate
		if current == 0 {
			current = sourceBitrate
		}
		return applyIntLimitation(lim.Comparison, lim.Values, current, func(v int) { ts.Bitrate = v })
	case LimitationAudioSamplerate:
		return applyIntLimitation(lim.Comparison, lim.Values, ts.SampleRate, func(v int) { ts.SampleRate = v })
	case LimitationAudioBitdepth:
		if ts.BitDepth > 0 {
			return applyIntLimitation(lim.Comparison, lim.Values, ts.BitDepth, func(v int) { ts.BitDepth = v })
		}
	case LimitationAudioProfile:
		// TODO: implement when audio profile data is available
	}
	return adjustNone
}

// applyIntLimitation applies a limitation comparison to a value.
// If the value needs adjusting, calls the setter and returns the result.
func applyIntLimitation(comparison string, values []string, current int, setter func(int)) adjustResult {
	if len(values) == 0 {
		return adjustNone
	}

	switch comparison {
	case ComparisonLessThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return adjustNone
		}
		if current <= limit {
			return adjustNone
		}
		setter(limit)
		return adjustAdjusted
	case ComparisonGreaterThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return adjustNone
		}
		if current >= limit {
			return adjustNone
		}
		// Cannot upscale
		return adjustCannotFit
	case ComparisonEquals:
		// Check if current value matches any allowed value
		for _, v := range values {
			if limit, ok := parseInt(v); ok && current == limit {
				return adjustNone
			}
		}
		// Find the closest allowed value below current (don't upscale)
		var closest int
		found := false
		for _, v := range values {
			if limit, ok := parseInt(v); ok && limit < current {
				if !found || limit > closest {
					closest = limit
					found = true
				}
			}
		}
		if found {
			setter(closest)
			return adjustAdjusted
		}
		return adjustCannotFit
	case ComparisonNotEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && current == limit {
				return adjustCannotFit
			}
		}
		return adjustNone
	}

	return adjustNone
}

func checkIntLimitation(value int, comparison string, values []string) bool {
	if len(values) == 0 {
		return true
	}

	switch comparison {
	case ComparisonLessThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return true
		}
		return value <= limit
	case ComparisonGreaterThanEqual:
		limit, ok := parseInt(values[0])
		if !ok {
			return true
		}
		return value >= limit
	case ComparisonEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && value == limit {
				return true
			}
		}
		return false
	case ComparisonNotEquals:
		for _, v := range values {
			if limit, ok := parseInt(v); ok && value == limit {
				return false
			}
		}
		return true
	}
	return true
}

// checkStringLimitation checks a string value against a limitation.
// Only Equals and NotEquals comparisons are meaningful for strings.
// LessThanEqual/GreaterThanEqual are not applicable and always pass.
func checkStringLimitation(value string, comparison string, values []string) bool {
	switch comparison {
	case ComparisonEquals:
		for _, v := range values {
			if strings.EqualFold(value, v) {
				return true
			}
		}
		return false
	case ComparisonNotEquals:
		for _, v := range values {
			if strings.EqualFold(value, v) {
				return false
			}
		}
		return true
	}
	return true
}

func parseInt(s string) (int, bool) {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}
