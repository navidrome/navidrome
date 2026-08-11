package blurhash

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/zeebo/xxh3"
)

// A real hash never has 3x3 components: components() always targets ~16 tiles.
const synthComponents = 3

// Synthetic returns a blurhash unique to seed, for artwork whose real hash does not exist
// yet. baseColor ("#rrggbb") sets the hue family; "" derives it from the seed.
func Synthetic(seed, baseColor string) string {
	hue, sat, light := baseTone(baseColor, xxh3.HashStringSeed(seed, 0))

	img := image.NewNRGBA(image.Rect(0, 0, synthComponents, synthComponents))
	for i := range synthComponents * synthComponents {
		// Each cell hashes the seed separately, so one tint shared by many items still
		// yields one value per item.
		bits := xxh3.HashStringSeed(seed, uint64(i)+1)
		dh := (byteFrac(bits, 0) - 0.5) * 60
		ds := (byteFrac(bits, 8) - 0.5) * 0.16
		dl := (byteFrac(bits, 16) - 0.5) * 0.30
		r, g, b := hslToRGB(hue+dh, clamp01(sat+ds), clamp01(light+dl))
		img.SetNRGBA(i%synthComponents, i/synthComponents, color.NRGBA{R: r, G: g, B: b, A: 255})
	}
	return encodeAt(img, synthComponents, synthComponents)
}

func byteFrac(bits uint64, shift int) float64 {
	return float64(bits>>shift&0xFF) / 255
}

// baseTone clamps saturation and lightness away from the extremes, so a placeholder
// never reads as neon or as solid black.
func baseTone(baseColor string, hueBits uint64) (hue, sat, light float64) {
	if r, g, b, ok := parseHex(baseColor); ok {
		h, s, l := rgbToHSL(r, g, b)
		return h, min(max(s, 0.10), 0.35), min(max(l, 0.15), 0.75)
	}
	return float64(hueBits&0x1FF) / 512 * 360, 0.22, 0.40
}

func parseHex(s string) (r, g, b uint8, ok bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), true
}

func rgbToHSL(r, g, b uint8) (h, s, l float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	mx, mn := max(rf, gf, bf), min(rf, gf, bf)
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	s = d / (1 - math.Abs(2*l-1))
	switch mx {
	case rf:
		h = math.Mod((gf-bf)/d, 6)
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
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
