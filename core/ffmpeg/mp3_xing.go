package ffmpeg

import (
	"encoding/binary"
	"math"
	"slices"

	"github.com/navidrome/navidrome/utils/gg"
)

const (
	mp3HeaderLen = 4
	mp3ID3Len    = 10
	mp3MaxPrefix = 64 << 10 // ffmpeg cannot write an attached picture to a pipe, so the tag stays small
)

// mp3Prefix returns the head of the stream with an Info frame inserted before the first
// audio frame, which ffmpeg omits on a pipe since it only knows the frame count once it
// can rewind. It returns the head unchanged when it cannot make sense of what ffmpeg wrote.
func (h *headerPatcher) mp3Prefix(buf []byte) ([]byte, error) {
	buf, err := h.fill(buf, mp3ID3Len)
	if err != nil || len(buf) < mp3ID3Len {
		return buf, err
	}
	start := 0
	if string(buf[:3]) == "ID3" {
		if start = id3TagLen(buf); start > mp3MaxPrefix {
			return buf, nil
		}
	}
	if buf, err = h.fill(buf, start+mp3HeaderLen); err != nil || len(buf) < start+mp3HeaderLen {
		return buf, err
	}
	frame, ok := parseMP3Header(buf[start:])
	if !ok {
		return buf, nil
	}
	if buf, err = h.fill(buf, start+frame.size); err != nil || len(buf) < start+frame.size {
		return buf, err
	}
	if isXingFrame(buf[start:], frame.tagOffset) {
		return buf, nil
	}
	xing, ok := xingFrame(buf[start:], frame, h.duration)
	if !ok {
		return buf, nil
	}
	return slices.Insert(buf, start, xing...), nil
}

func id3TagLen(header []byte) int {
	size := int(header[6]&0x7F)<<21 | int(header[7]&0x7F)<<14 | int(header[8]&0x7F)<<7 | int(header[9]&0x7F)
	if header[5]&0x10 != 0 { // footer present
		size += mp3ID3Len
	}
	return mp3ID3Len + size
}

type mp3Frame struct {
	sampleRate int
	samples    int // per frame
	size       int // bytes, including the header
	sideInfo   int
	tagOffset  int // where a Xing tag sits in the incoming frame, which may carry a CRC
}

var (
	mp3BitRates = [2][16]int{
		{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}, // MPEG 1
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},     // MPEG 2 and 2.5
	}
	mp3SampleRates = [4][4]int{
		{11025, 12000, 8000, 0},  // MPEG 2.5
		{},                       // reserved
		{22050, 24000, 16000, 0}, // MPEG 2
		{44100, 48000, 32000, 0}, // MPEG 1
	}
)

func parseMP3Header(h []byte) (mp3Frame, bool) {
	version, layer := (h[1]>>3)&0x03, (h[1]>>1)&0x03
	if h[0] != 0xFF || h[1]&0xE0 != 0xE0 || version == 1 || layer != 1 {
		return mp3Frame{}, false // not the sync word of an MPEG Layer III frame
	}
	mpeg1 := version == 3
	sampleRate := mp3SampleRates[version][(h[2]>>2)&0x03]
	bitRate := 1000 * mp3BitRates[gg.If(mpeg1, 0, 1)][h[2]>>4]
	if sampleRate == 0 || bitRate == 0 {
		return mp3Frame{}, false
	}
	f := mp3Frame{sampleRate: sampleRate, samples: 576}
	mono := (h[3]>>6)&0x03 == 3
	switch {
	case mpeg1 && mono:
		f.samples, f.sideInfo = 1152, 17
	case mpeg1:
		f.samples, f.sideInfo = 1152, 32
	case mono:
		f.sideInfo = 9
	default:
		f.sideInfo = 17
	}
	f.size = f.samples/8*bitRate/sampleRate + int((h[2]>>1)&0x01)
	f.tagOffset = mp3HeaderLen + f.sideInfo
	if h[1]&0x01 == 0 { // CRC follows the header
		f.tagOffset += 2
	}
	return f, true
}

func isXingFrame(frame []byte, tagOffset int) bool {
	if len(frame) < tagOffset+4 {
		return false
	}
	tag := string(frame[tagOffset : tagOffset+4])
	return tag == "Xing" || tag == "Info"
}

// xingFrame builds a silent Info frame declaring how many frames follow it. It reuses the
// first frame's header, minus its CRC, so the two describe the same stream.
func xingFrame(first []byte, f mp3Frame, duration float32) ([]byte, bool) {
	frames := math.Round(float64(duration) * float64(f.sampleRate) / float64(f.samples))
	if frames < 1 || frames > math.MaxUint32 {
		return nil, false
	}
	frame := make([]byte, f.size)
	copy(frame, first[:mp3HeaderLen])
	frame[1] |= 0x01 // no CRC, so the tag follows the side info directly
	at := mp3HeaderLen + f.sideInfo
	copy(frame[at:], "Info")                    // a Xing tag without a seek table marks the stream unseekable to some decoders
	binary.BigEndian.PutUint32(frame[at+4:], 1) // only the frame count is present
	binary.BigEndian.PutUint32(frame[at+8:], uint32(frames))
	return frame, true
}
