// Package blurhash implements the blurhash encoding algorithm (https://github.com/woltapp/blurhash),
// matching Jellyfin's parameters so clients tuned against Jellyfin see equivalent hashes.
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

// maxInputSize matches Jellyfin: larger inputs are slower with no visually discernible difference.
const maxInputSize = 128

// Components picks x/y component counts for an image, targeting ~16 near-square tiles (Jellyfin's formula).
func Components(width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	xf := math.Sqrt(16.0 * float64(width) / float64(height))
	yf := xf * float64(height) / float64(width)
	return min(int(xf)+1, 9), min(int(yf)+1, 9)
}

// Encode returns the blurhash of img using xComp x yComp components.
func Encode(img image.Image, xComp, yComp int) (string, error) {
	if xComp < 1 || xComp > 9 || yComp < 1 || yComp > 9 {
		return "", errors.New("blurhash: components must be between 1 and 9")
	}
	rgba := toRGBA(downscale(img))
	bounds := rgba.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return "", errors.New("blurhash: empty image")
	}

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
	for y := 0; y < h; y++ {
		row := rgba.Pix[y*rgba.Stride:]
		for x := 0; x < w; x++ {
			p := x * 4
			lr, lg, lb := lin[row[p]], lin[row[p+1]], lin[row[p+2]]
			for j := 0; j < yComp; j++ {
				for i := 0; i < xComp; i++ {
					basis := cosX[i][x] * cosY[j][y]
					f := &factors[j*xComp+i]
					f[0] += basis * lr
					f[1] += basis * lg
					f[2] += basis * lb
				}
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

	ac := factors[1:]
	maxVal := 1.0
	if len(ac) > 0 {
		actualMax := 0.0
		for _, f := range ac {
			actualMax = max(actualMax, math.Abs(f[0]), math.Abs(f[1]), math.Abs(f[2]))
		}
		quantMax := int(math.Max(0, math.Min(82, math.Floor(actualMax*166-0.5))))
		maxVal = float64(quantMax+1) / 166
		sb.WriteString(encode83(quantMax, 1))
	} else {
		sb.WriteString(encode83(0, 1))
	}

	dc := factors[0]
	sb.WriteString(encode83(linearToSRGB(dc[0])<<16|linearToSRGB(dc[1])<<8|linearToSRGB(dc[2]), 4))
	for _, f := range ac {
		sb.WriteString(encode83(quantAC(f[0], maxVal)*19*19+quantAC(f[1], maxVal)*19+quantAC(f[2], maxVal), 2))
	}
	return sb.String(), nil
}

// toRGBA gives the pixel loop direct Pix access, avoiding a per-pixel allocation through the
// image.At interface (~16k allocs per encode).
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
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
	return int(math.Max(0, math.Min(18, math.Floor(signPow(v/maxVal, 0.5)*9+9.5))))
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
	v = math.Min(math.Max(0, v), 1)
	if v <= 0.0031308 {
		return int(v*12.92*255 + 0.5)
	}
	return int((1.055*math.Pow(v, 1/2.4)-0.055)*255 + 0.5)
}

func encode83(value, length int) string {
	b := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		b[i] = alphabet[value%83]
		value /= 83
	}
	return string(b)
}
