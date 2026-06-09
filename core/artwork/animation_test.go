package artwork

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/png"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Animation detection", func() {
	Describe("isAnimatedGIF", func() {
		It("detects an animated GIF with multiple frames", func() {
			Expect(isAnimatedGIF(createAnimatedGIF(2))).To(BeTrue())
		})

		It("detects an animated GIF with many frames", func() {
			Expect(isAnimatedGIF(createAnimatedGIF(5))).To(BeTrue())
		})

		It("does not flag a static GIF (single frame)", func() {
			Expect(isAnimatedGIF(createAnimatedGIF(1))).To(BeFalse())
		})

		It("returns false for non-GIF data", func() {
			Expect(isAnimatedGIF(nil)).To(BeFalse())
			Expect(isAnimatedGIF([]byte{0xFF, 0xD8})).To(BeFalse())
		})
	})

	Describe("isAnimatedWebP", func() {
		It("detects an animated WebP with ANMF chunk", func() {
			Expect(isAnimatedWebP(createAnimatedWebPBytes())).To(BeTrue())
		})

		It("does not flag a static WebP (no ANMF chunk)", func() {
			Expect(isAnimatedWebP(createStaticWebPBytes())).To(BeFalse())
		})

		It("returns false for non-WebP data", func() {
			Expect(isAnimatedWebP(nil)).To(BeFalse())
			Expect(isAnimatedWebP([]byte{0xFF, 0xD8})).To(BeFalse())
		})
	})

	Describe("isAnimatedPNG", func() {
		It("detects an APNG with acTL chunk", func() {
			Expect(isAnimatedPNG(createAPNGBytes())).To(BeTrue())
		})

		It("does not flag a static PNG (no acTL chunk)", func() {
			Expect(isAnimatedPNG(createStaticPNGBytes())).To(BeFalse())
		})

		It("returns false for non-PNG data", func() {
			Expect(isAnimatedPNG(nil)).To(BeFalse())
			Expect(isAnimatedPNG([]byte{0xFF, 0xD8})).To(BeFalse())
		})
	})
})

// createAnimatedGIF creates a minimal animated GIF with the given number of frames.
func createAnimatedGIF(frames int) []byte {
	g := &gif.GIF{
		LoopCount: 0,
	}
	for range frames {
		img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, 10)
	}
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, g)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// writeUint32LE appends a little-endian uint32 to the buffer.
func writeUint32LE(buf *bytes.Buffer, v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	buf.Write(b)
}

// writeUint32BE appends a big-endian uint32 to the buffer.
func writeUint32BE(buf *bytes.Buffer, v uint32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	buf.Write(b)
}

// createAnimatedWebPBytes creates a minimal RIFF/WEBP container with an ANMF chunk.
func createAnimatedWebPBytes() []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	writeUint32LE(&buf, 100) // file size placeholder
	buf.WriteString("WEBP")
	// VP8X chunk (extended format, required for animation)
	buf.WriteString("VP8X")
	writeUint32LE(&buf, 10)
	buf.Write(make([]byte, 10))
	// ANIM chunk (animation parameters)
	buf.WriteString("ANIM")
	writeUint32LE(&buf, 6)
	buf.Write(make([]byte, 6))
	// ANMF chunk (animation frame)
	buf.WriteString("ANMF")
	writeUint32LE(&buf, 16)
	buf.Write(make([]byte, 16))
	return buf.Bytes()
}

// createStaticWebPBytes creates a minimal RIFF/WEBP container without ANMF chunks.
func createStaticWebPBytes() []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	writeUint32LE(&buf, 20) // file size
	buf.WriteString("WEBP")
	// VP8 chunk (simple lossy format)
	buf.WriteString("VP8 ")
	writeUint32LE(&buf, 4)
	buf.Write(make([]byte, 4))
	return buf.Bytes()
}

// createAPNGBytes creates a minimal PNG with an acTL chunk (making it APNG).
func createAPNGBytes() []byte {
	// Start with a real PNG
	staticPNG := createStaticPNGBytes()

	// Insert an acTL chunk after the IHDR chunk.
	// PNG structure: signature (8) + IHDR chunk (4 len + 4 type + 13 data + 4 crc = 25)
	ihdrEnd := 8 + 25
	var buf bytes.Buffer
	buf.Write(staticPNG[:ihdrEnd])
	// Write acTL chunk: length=8, type="acTL", data=num_frames(4)+num_plays(4), CRC=4
	writeUint32BE(&buf, 8) // chunk data length
	buf.WriteString("acTL")
	writeUint32BE(&buf, 2) // num_frames
	writeUint32BE(&buf, 0) // num_plays (0 = infinite)
	writeUint32BE(&buf, 0) // CRC placeholder
	buf.Write(staticPNG[ihdrEnd:])
	return buf.Bytes()
}

// createStaticPNGBytes creates a minimal valid static PNG.
func createStaticPNGBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
