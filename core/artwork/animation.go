package artwork

import (
	"bytes"
	"encoding/binary"
)

// isAnimatedGIF checks for multiple image descriptor blocks (0x2C) in a GIF file.
// Animated GIFs use GIF89a and contain multiple image blocks.
func isAnimatedGIF(data []byte) bool {
	// GIF header: "GIF87a" or "GIF89a"
	if !bytes.HasPrefix(data, []byte("GIF")) {
		return false
	}

	// Skip header (6 bytes) + logical screen descriptor (7 bytes)
	pos := 13
	if pos >= len(data) {
		return false
	}

	// Skip Global Color Table if present (bit 7 of packed byte at offset 10)
	if len(data) > 10 && data[10]&0x80 != 0 {
		// GCT size = 3 * 2^(N+1) where N = bits 0-2 of packed byte
		gctSize := 3 * (1 << ((data[10] & 0x07) + 1))
		pos += gctSize
	}

	frameCount := 0
	for pos < len(data) {
		switch data[pos] {
		case 0x2C: // Image Descriptor - marks a frame
			frameCount++
			if frameCount > 1 {
				return true
			}
			pos++ // skip introducer
			if pos+8 >= len(data) {
				return false
			}
			pos += 8 // skip x, y, w, h (each 2 bytes)
			packed := data[pos]
			pos++ // skip packed byte
			// Skip Local Color Table if present
			if packed&0x80 != 0 {
				lctSize := 3 * (1 << ((packed & 0x07) + 1))
				pos += lctSize
			}
			// Skip LZW minimum code size
			pos++
			// Skip sub-blocks
			pos = skipGIFSubBlocks(data, pos)
		case 0x21: // Extension block
			pos++ // skip introducer
			if pos >= len(data) {
				return false
			}
			pos++ // skip extension label
			// Skip sub-blocks
			pos = skipGIFSubBlocks(data, pos)
		case 0x3B: // Trailer
			return false
		default:
			// Unknown block, bail
			return false
		}
	}
	return false
}

// skipGIFSubBlocks advances past a sequence of GIF sub-blocks (terminated by a zero-length block).
func skipGIFSubBlocks(data []byte, pos int) int {
	for pos < len(data) {
		blockSize := int(data[pos])
		pos++ // skip size byte
		if blockSize == 0 {
			break
		}
		pos += blockSize
	}
	return pos
}

// isAnimatedWebP checks for ANMF (animation frame) chunks in a WebP RIFF container.
func isAnimatedWebP(data []byte) bool {
	// WebP header: "RIFF" + 4 bytes size + "WEBP"
	if !bytes.HasPrefix(data, []byte("RIFF")) || len(data) < 12 {
		return false
	}
	if !bytes.Equal(data[8:12], []byte("WEBP")) {
		return false
	}
	// Scan for ANMF chunk identifier
	return bytes.Contains(data[12:], []byte("ANMF"))
}

// isAnimatedPNG checks for the acTL (animation control) chunk in a PNG file.
// APNG files contain an acTL chunk that is not present in static PNGs.
func isAnimatedPNG(data []byte) bool {
	// PNG signature: 8 bytes
	pngSig := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(data, pngSig) {
		return false
	}

	// Scan chunks for "acTL" (animation control)
	pos := uint64(8)
	dataLen := uint64(len(data))
	for pos+8 <= dataLen {
		chunkLen := uint64(binary.BigEndian.Uint32(data[pos : pos+4]))
		chunkType := string(data[pos+4 : pos+8])

		if chunkType == "acTL" {
			return true
		}
		// Move to next chunk: 4 (length) + 4 (type) + chunkLen (data) + 4 (CRC)
		pos += 12 + chunkLen
	}
	return false
}
