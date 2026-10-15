// Package dominant extracts an image's dominant colour, for use as a flat placeholder while the
// real artwork loads.
package dominant

import (
	"fmt"
	"image"
	"math"
	"sort"
)

const (
	// 4 bits per channel: coarse enough that near-identical pixels land together, fine enough that
	// distinct colours stay apart.
	bits  = 4
	nBins = 1 << (3 * bits)
	// Only the heaviest bins can win, and merging is O(n^2) over whatever survives.
	maxBins = 64
	// Oklab distance below which two bins are the same colour to the eye. Merging matters because a
	// gradient splits across adjacent bins and would otherwise lose to a smaller flat region.
	mergeDist = 0.10
)

type bin struct {
	r, g, b float64
	n       float64
}

// Color returns the dominant colour as "#rrggbb", or "" when the image has no pixels. It reports
// presence, not salience: a mostly white sleeve returns white.
func Color(img image.Image) string {
	var bins [nBins]bin
	total := 0
	eachPixel(img, func(r, g, b uint8) {
		i := int(r>>(8-bits))<<(2*bits) | int(g>>(8-bits))<<bits | int(b>>(8-bits))
		bins[i].r += float64(r)
		bins[i].g += float64(g)
		bins[i].b += float64(b)
		bins[i].n++
		total++
	})
	if total == 0 {
		return ""
	}

	used := make([]bin, 0, 32)
	for i := range bins {
		if bins[i].n > 0 {
			used = append(used, bins[i])
		}
	}
	sort.Slice(used, func(i, j int) bool { return used[i].n > used[j].n })
	if len(used) > maxBins {
		used = used[:maxBins]
	}

	merged := make([]bin, 0, len(used))
	for _, b := range used {
		if i := nearest(merged, b); i >= 0 {
			merged[i].r += b.r
			merged[i].g += b.g
			merged[i].b += b.b
			merged[i].n += b.n
			continue
		}
		merged = append(merged, b)
	}

	best := merged[0]
	for _, m := range merged[1:] {
		if m.n > best.n {
			best = m
		}
	}
	return fmt.Sprintf("#%02x%02x%02x",
		uint8(best.r/best.n+0.5), uint8(best.g/best.n+0.5), uint8(best.b/best.n+0.5))
}

func nearest(merged []bin, b bin) int {
	bl, ba, bb := oklab(b.r/b.n, b.g/b.n, b.b/b.n)
	for i, m := range merged {
		ml, ma, mb := oklab(m.r/m.n, m.g/m.n, m.b/m.n)
		if math.Sqrt((bl-ml)*(bl-ml)+(ba-ma)*(ba-ma)+(bb-mb)*(bb-mb)) < mergeDist {
			return i
		}
	}
	return -1
}

// eachPixel walks the image, taking the NRGBA fast path the artwork pipeline always hits: both hash
// encoders already read the shared thumbnail in that form.
func eachPixel(img image.Image, fn func(r, g, b uint8)) {
	if p, ok := img.(*image.NRGBA); ok {
		for y := range p.Rect.Dy() {
			row := p.Pix[y*p.Stride : y*p.Stride+p.Rect.Dx()*4]
			for x := 0; x < len(row); x += 4 {
				fn(row[x], row[x+1], row[x+2])
			}
		}
		return
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			fn(uint8(r>>8), uint8(g>>8), uint8(bl>>8))
		}
	}
}

func srgbToLinear(v float64) float64 {
	v /= 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func oklab(r, g, b float64) (float64, float64, float64) {
	lr, lg, lb := srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)
	l := math.Cbrt(0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb)
	m := math.Cbrt(0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb)
	s := math.Cbrt(0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}
