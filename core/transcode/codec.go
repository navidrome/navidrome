package transcode

import "strings"

// isLosslessFormat returns true if the format is a lossless audio codec/format.
// Note: core/ffmpeg has a separate isLosslessOutputFormat that covers only formats
// ffmpeg can produce as output (a smaller set). This function covers all known lossless formats
// for transcoding decision purposes.
func isLosslessFormat(format string) bool {
	switch strings.ToLower(format) {
	case "flac", "alac", "wav", "aiff", "ape", "wv", "tta", "tak", "shn", "dsd":
		return true
	}
	return false
}

// normalizeSourceSampleRate adjusts the source sample rate for codecs that store
// it differently than PCM. Currently handles DSD (÷8):
// DSD64=2822400→352800, DSD128=5644800→705600, etc.
// For other codecs, returns the rate unchanged.
func normalizeSourceSampleRate(sampleRate int, codec string) int {
	if strings.EqualFold(codec, "dsd") && sampleRate > 0 {
		return sampleRate / 8
	}
	return sampleRate
}

// normalizeSourceBitDepth adjusts the source bit depth for codecs that use
// non-standard bit depths. Currently handles DSD (1-bit → 24-bit PCM, which is
// what ffmpeg produces). For other codecs, returns the depth unchanged.
func normalizeSourceBitDepth(bitDepth int, codec string) int {
	if strings.EqualFold(codec, "dsd") && bitDepth == 1 {
		return 24
	}
	return bitDepth
}

// codecFixedOutputSampleRate returns the mandatory output sample rate for codecs
// that always resample regardless of input (e.g., Opus always outputs 48000Hz).
// Returns 0 if the codec has no fixed output rate.
func codecFixedOutputSampleRate(codec string) int {
	switch strings.ToLower(codec) {
	case "opus":
		return 48000
	}
	return 0
}

// codecMaxSampleRate returns the hard maximum output sample rate for a codec.
// Returns 0 if the codec has no hard limit.
func codecMaxSampleRate(codec string) int {
	switch strings.ToLower(codec) {
	case "mp3":
		return 48000
	case "aac":
		return 96000
	}
	return 0
}
