package blurhash

import (
	"image"
	"image/color"
	"math"

	"github.com/zeebo/xxh3"
)

// A real hash never has 3x3 components: components() always targets ~16 tiles.
const synthComponents = 3

// Synthetic returns a blurhash unique to seed, for artwork whose real hash does not exist
// yet. baseColor ("#rrggbb") sets the hue family; "" derives it from the seed.
func Synthetic(seed, baseColor string) string {
	hue, sat, light := baseTone(baseColor, xxh3.HashStringSeed(seed, 0))

	const n = synthComponents
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i := range n * n {
		// Each cell hashes the seed separately, so one tint shared by many items still
		// yields one value per item.
		bits := xxh3.HashStringSeed(seed, uint64(i)+1)
		dh := (float64(bits&0xFF)/255 - 0.5) * 60
		ds := (float64(bits>>8&0x3F)/63 - 0.5) * 0.16
		dl := (float64(bits>>14&0x3F)/63 - 0.5) * 0.30
		r, g, b := hslToRGB(hue+dh, clamp01(sat+ds), clamp01(light+dl))
		img.SetNRGBA(i%n, i/n, color.NRGBA{R: r, G: g, B: b, A: 255})
	}
	return encodeAt(img, n, n)
}

// baseTone is the hue, saturation and lightness the cells orbit.
func baseTone(baseColor string, hueBits uint64) (hue, sat, light float64) {
	return float64(hueBits&0x1FF) / 512 * 360, 0.22, 0.40
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	h = math.Mod(math.Mod(h, 360)+360, 360)
	c := (1 - math.Abs(2*l-1)) * s
	hp := h / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch int(hp) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return to8(r + m), to8(g + m), to8(b + m)
}

func clamp01(v float64) float64 { return min(max(v, 0), 1) }

func to8(v float64) uint8 { return uint8(math.Round(clamp01(v) * 255)) }
