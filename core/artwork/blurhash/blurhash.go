// Package blurhash implements the blurhash encoding (https://github.com/woltapp/blurhash),
// parameterized to match Jellyfin so clients see equivalent hashes.
package blurhash

import (
	"errors"
	"image"
	"image/draw"
	"math"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~"

// maxInputSize: larger inputs are slower with no visible difference in the result.
const maxInputSize = 128

// components picks x/y component counts targeting ~16 near-square tiles.
func components(width, height int) (int, int) {
	xf := math.Sqrt(16.0 * float64(width) / float64(height))
	yf := xf * float64(height) / float64(width)
	return min(int(xf)+1, 9), min(int(yf)+1, 9)
}

// Encode returns the blurhash of img, deriving the component counts from its aspect ratio.
func Encode(img image.Image) (string, error) {
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return "", errors.New("blurhash: empty image")
	}
	// Pre-downscale: its rounding can flip a component count, and the hash is a client cache key.
	xComp, yComp := components(img.Bounds().Dx(), img.Bounds().Dy())
	src := pixelsOf(downscale(img))
	w, h := src.w, src.h

	cosX := make([][]float64, xComp)
	for i := range cosX {
		cosX[i] = make([]float64, w)
		for x := range cosX[i] {
			cosX[i][x] = math.Cos(math.Pi * float64(i) * float64(x) / float64(w))
		}
	}
	cosY := make([][]float64, yComp)
	for j := range cosY {
		cosY[j] = make([]float64, h)
		for y := range cosY[j] {
			cosY[j][y] = math.Cos(math.Pi * float64(j) * float64(y) / float64(h))
		}
	}

	lin := srgbToLinearTable()
	factors := make([][3]float64, xComp*yComp)
	linR := make([]float64, w)
	linG := make([]float64, w)
	linB := make([]float64, w)
	rowR := make([]float64, xComp)
	rowG := make([]float64, xComp)
	rowB := make([]float64, xComp)
	for y := range h {
		row := src.pix[y*src.stride:]
		for x := range w {
			p := x * 4
			r, g, b := row[p], row[p+1], row[p+2]
			if src.straight {
				r, g, b = premultiply(r, g, b, row[p+3])
			}
			linR[x], linG[x], linB[x] = lin[r], lin[g], lin[b]
		}
		// The basis is separable, so a row costs xComp dot products plus one fold over yComp,
		// rather than xComp*yComp multiply-accumulates per pixel.
		for i := range xComp {
			var sr, sg, sb float64
			for x, c := range cosX[i] {
				sr += c * linR[x]
				sg += c * linG[x]
				sb += c * linB[x]
			}
			rowR[i], rowG[i], rowB[i] = sr, sg, sb
		}
		for j := range yComp {
			cy := cosY[j][y]
			for i := range xComp {
				f := &factors[j*xComp+i]
				f[0] += cy * rowR[i]
				f[1] += cy * rowG[i]
				f[2] += cy * rowB[i]
			}
		}
	}
	for idx := range factors {
		norm := 2.0
		if idx == 0 {
			norm = 1.0
		}
		scale := norm / float64(w*h)
		factors[idx][0] *= scale
		factors[idx][1] *= scale
		factors[idx][2] *= scale
	}

	var sb strings.Builder
	sb.WriteString(encode83((xComp-1)+(yComp-1)*9, 1))

	// Derived counts are at least 1x9, so there is always at least one AC factor.
	ac := factors[1:]
	actualMax := 0.0
	for _, f := range ac {
		actualMax = max(actualMax, math.Abs(f[0]), math.Abs(f[1]), math.Abs(f[2]))
	}
	quantMax := int(max(0, min(82, math.Floor(actualMax*166-0.5))))
	maxVal := float64(quantMax+1) / 166
	sb.WriteString(encode83(quantMax, 1))

	dc := factors[0]
	sb.WriteString(encode83(linearToSRGB(dc[0])<<16|linearToSRGB(dc[1])<<8|linearToSRGB(dc[2]), 4))
	for _, f := range ac {
		sb.WriteString(encode83(quantAC(f[0], maxVal)*19*19+quantAC(f[1], maxVal)*19+quantAC(f[2], maxVal), 2))
	}
	return sb.String(), nil
}

// pixels is direct Pix access for the pixel loop, avoiding a per-pixel allocation via image.At.
type pixels struct {
	pix    []uint8
	stride int
	w, h   int
	// straight marks non-premultiplied alpha, which the loop premultiplies to keep the hash
	// identical to the one an equivalent *image.RGBA produces.
	straight bool
}

// pixelsOf accepts the two types the artwork pipeline produces without copying, and converts
// anything else.
func pixelsOf(img image.Image) pixels {
	b := img.Bounds()
	switch src := img.(type) {
	case *image.RGBA:
		return pixels{pix: src.Pix, stride: src.Stride, w: b.Dx(), h: b.Dy()}
	case *image.NRGBA:
		return pixels{pix: src.Pix, stride: src.Stride, w: b.Dx(), h: b.Dy(), straight: true}
	}
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return pixels{pix: dst.Pix, stride: dst.Stride, w: b.Dx(), h: b.Dy()}
}

func premultiply(r, g, b, a uint8) (uint8, uint8, uint8) {
	if a == 255 {
		return r, g, b
	}
	return uint8(uint32(r) * uint32(a) / 255), uint8(uint32(g) * uint32(a) / 255), uint8(uint32(b) * uint32(a) / 255)
}

var srgbToLinearTable = sync.OnceValue(func() *[256]float64 {
	var t [256]float64
	for i := range t {
		t[i] = srgbToLinear(i)
	}
	return &t
})

func downscale(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxInputSize && h <= maxInputSize {
		return img
	}
	scale := float64(maxInputSize) / float64(max(w, h))
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

func quantAC(v, maxVal float64) int {
	return int(max(0, min(18, math.Floor(signPow(v/maxVal, 0.5)*9+9.5))))
}

func signPow(v, exp float64) float64 {
	return math.Copysign(math.Pow(math.Abs(v), exp), v)
}

func srgbToLinear(v int) float64 {
	f := float64(v) / 255
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) int {
	v = min(max(0, v), 1)
	if v <= 0.0031308 {
		return int(v*12.92*255 + 0.5)
	}
	return int((1.055*math.Pow(v, 1/2.4)-0.055)*255 + 0.5)
}

// encode83 encodes value as a fixed-width, big-endian base83 string of the given length.
func encode83(value, length int) string {
	b := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		b[i] = alphabet[value%83]
		value /= 83
	}
	return string(b)
}
