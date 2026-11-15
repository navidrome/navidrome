package stream

import (
	"iter"
	"slices"
	"strconv"
	"strings"
)

// adjustResult represents the outcome of applying a limitation to a transcoded stream value
type adjustResult int

const (
	adjustNone      adjustResult = iota // Value already satisfies the limitation
	adjustAdjusted                      // Value was changed to fit the limitation
	adjustCannotFit                     // Cannot satisfy the limitation (reject this profile)
)

// checkLimitations checks codec profile limitations against source stream details.
// Returns "" if all limitations pass, or a typed reason string for the first failure.
func checkLimitations(src *Details, limitations []Limitation) string {
	for _, lim := range limitations {
		var ok bool
		var reason string

		switch lim.Name {
		case LimitationAudioChannels:
			ok = checkIntLimitation(src.Channels, lim.Comparison, lim.Values)
			reason = "audio channels not supported"
		case LimitationAudioSamplerate:
			ok = checkIntLimitation(src.SampleRate, lim.Comparison, lim.Values)
			reason = "audio samplerate not supported"
		case LimitationAudioBitrate:
			ok = checkIntLimitation(src.Bitrate, lim.Comparison, lim.Values)
			reason = "audio bitrate not supported"
		case LimitationAudioBitdepth:
			ok = checkIntLimitation(src.BitDepth, lim.Comparison, lim.Values)
			reason = "audio bitdepth not supported"
		case LimitationAudioProfile:
			ok = checkStringLimitation(src.Profile, lim.Comparison, lim.Values)
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

func audioCodecProfiles(clientInfo *ClientInfo, codec string) iter.Seq[CodecProfile] {
	return func(yield func(CodecProfile) bool) {
		for _, cp := range clientInfo.CodecProfiles {
			if strings.EqualFold(cp.Type, CodecProfileTypeAudio) && matchesCodec(codec, []string{cp.Name}) {
				if !yield(cp) {
					return
				}
			}
		}
	}
}

func hasUpperBoundLimitation(clientInfo *ClientInfo, codec, name string) bool {
	for cp := range audioCodecProfiles(clientInfo, codec) {
		if slices.ContainsFunc(cp.Limitations, func(lim Limitation) bool {
			return strings.EqualFold(lim.Name, name) && len(upperBounds(lim.Comparison, lim.Values)) > 0
		}) {
			return true
		}
	}
	return false
}

// upperBounds returns the values a limitation can lower a number to, none for >=, != or non-numeric limits.
func upperBounds(comparison string, values []string) []int {
	switch comparison {
	case ComparisonLessThanEqual:
		values = values[:min(len(values), 1)]
	case ComparisonEquals:
	default:
		return nil
	}
	var bounds []int
	for _, s := range values {
		if v, ok := parseInt(s); ok {
			bounds = append(bounds, v)
		}
	}
	return bounds
}

// applyLimitation adjusts a transcoded stream parameter to satisfy the limitation.
// Returns the adjustment result.
func applyLimitation(sourceBitrate int, lim *Limitation, ts *Details) adjustResult {
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
		bounds := upperBounds(comparison, values)
		if len(bounds) == 0 || current <= bounds[0] {
			return adjustNone
		}
		setter(bounds[0])
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
		bounds := upperBounds(comparison, values)
		if slices.Contains(bounds, current) {
			return adjustNone
		}
		// Find the closest allowed value below current (don't upscale)
		var closest int
		found := false
		for _, limit := range bounds {
			if limit < current && (!found || limit > closest) {
				closest = limit
				found = true
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
	return applyIntLimitation(comparison, values, value, func(int) {}) == adjustNone
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
