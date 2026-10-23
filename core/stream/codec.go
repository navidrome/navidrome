package stream

import "strings"

// normalizeProbeCodec maps ffprobe codec_name values to the simplified internal
// codec names used throughout Navidrome (matching inferCodecFromSuffix output).
// Most ffprobe names match directly; this handles the exceptions.
func normalizeProbeCodec(codec string) string {
	c := strings.ToLower(codec)
	// DSD variants: dsd_lsbf_planar, dsd_msbf_planar, dsd_lsbf, dsd_msbf
	if strings.HasPrefix(c, "dsd") {
		return "dsd"
	}
	// PCM variants: pcm_s16le, pcm_s24le, pcm_s32be, pcm_f32le, etc.
	if strings.HasPrefix(c, "pcm_") {
		return "pcm"
	}
	return c
}

// isLosslessFormat returns true if the format is a known lossless audio codec/format.
// Detection is based on codec name only, not bit depth — some lossy codecs (e.g. ADPCM)
// report non-zero bits_per_sample in ffprobe, so bit depth alone is not a reliable signal.
//
// Note: core/ffmpeg has a separate isLosslessOutputFormat that covers only formats
// ffmpeg can produce as output (a smaller set).
func isLosslessFormat(format string) bool {
	switch strings.ToLower(format) {
	case "flac", "alac", "wav", "aiff", "ape", "wv", "wavpack", "tta", "tak", "shn", "dsd", "pcm":
		return true
	}
	return false
}

// normalizeSourceSampleRate adjusts a sample rate read from tags for codecs that
// store it differently than PCM. Currently handles DSD (÷8):
// DSD64=2822400→352800, DSD128=5644800→705600, etc.
// ffprobe already reports DSD at this rate, so probe data needs no adjustment.
// For other codecs, returns the rate unchanged.
func normalizeSourceSampleRate(sampleRate int, codec string) int {
	if strings.EqualFold(codec, "dsd") && sampleRate > 0 {
		return sampleRate / 8
	}
	return sampleRate
}

// dsdMaxPCMSampleRate caps DSD transcoded to lossless PCM. It divides every
// 44.1kHz-based DSD rate exactly, DSD carries mostly its own shaped noise above
// it, and many clients cannot play 352.8kHz PCM.
const dsdMaxPCMSampleRate = 88200

// targetSampleRate returns the sample rate for a transcoded stream before codec
// and client limits are applied, with DSD capped when the target is lossless.
func targetSampleRate(srcSampleRate int, srcCodec string, targetIsLossless bool) int {
	if targetIsLossless && strings.EqualFold(srcCodec, "dsd") {
		return min(srcSampleRate, dsdMaxPCMSampleRate)
	}
	return srcSampleRate
}

// targetBitDepth returns the bit depth for a transcoded stream: 0 for lossy
// targets (they have no PCM bit depth), otherwise the source depth, with DSD
// (1 bit in tags, 8 in ffprobe) adjusted to the 24-bit PCM that ffmpeg produces.
func targetBitDepth(srcBitDepth int, srcCodec string, targetIsLossless bool) int {
	if !targetIsLossless {
		return 0
	}
	if strings.EqualFold(srcCodec, "dsd") {
		return 24
	}
	return srcBitDepth
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

// codecMaxChannels returns the hard maximum number of audio channels a codec
// supports. Returns 0 if the codec has no hard limit (or is unknown), in which
// case the source/profile constraints applied upstream are authoritative.
func codecMaxChannels(codec string) int {
	switch strings.ToLower(codec) {
	case "mp3":
		return 2
	case "opus":
		return 8
	}
	return 0
}
