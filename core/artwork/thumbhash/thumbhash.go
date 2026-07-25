// Package thumbhash implements the ThumbHash encoding algorithm (https://github.com/evanw/thumbhash).
package thumbhash

import (
	"errors"
	"image"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// maxInputSize is the algorithm's hard limit: larger inputs are rejected by every implementation.
const maxInputSize = 100

// term is one DCT coefficient's frequency pair, in the reference's triangular scan order.
type term struct{ cx, cy int }

// Encode returns the ThumbHash of img: 24 bytes when opaque, 25 with alpha.
func Encode(img image.Image) ([]byte, error) {
	rgba := toNRGBA(downscale(img))
	b := rgba.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, errors.New("thumbhash: empty image")
	}

	avgR, avgG, avgB, avgA := averageColor(rgba, w, h)
	hasAlpha := avgA < float64(w*h)
	if avgA > 0 {
		avgR /= avgA
		avgG /= avgA
		avgB /= avgA
	}

	lLimit := 7.0
	if hasAlpha {
		lLimit = 5.0
	}
	maxWH := float64(max(w, h))
	lx := max(1, int(math.Round(lLimit*float64(w)/maxWH)))
	ly := max(1, int(math.Round(lLimit*float64(h)/maxWH)))

	lTerms := terms(max(3, lx), max(3, ly))
	pTerms := terms(3, 3)
	qTerms := terms(3, 3)
	var aTerms []term
	if hasAlpha {
		aTerms = terms(5, 5)
	}

	nx := maxCX(lTerms, pTerms, qTerms, aTerms) + 1
	cosX := cosTable(nx, w)
	cosY := cosTable(maxCY(lTerms, pTerms, qTerms, aTerms)+1, h)

	lAcc := make([]float64, len(lTerms))
	pAcc := make([]float64, len(pTerms))
	qAcc := make([]float64, len(qTerms))
	aAcc := make([]float64, len(aTerms))
	rowL := make([]float64, nx)
	rowP := make([]float64, nx)
	rowQ := make([]float64, nx)
	rowA := make([]float64, nx)

	for y := range h {
		clear(rowL)
		clear(rowP)
		clear(rowQ)
		clear(rowA)
		row := rgba.Pix[y*rgba.Stride:]
		for x := range w {
			j := x * 4
			alpha := float64(row[j+3]) / 255
			r := avgR*(1-alpha) + alpha/255*float64(row[j])
			g := avgG*(1-alpha) + alpha/255*float64(row[j+1])
			bl := avgB*(1-alpha) + alpha/255*float64(row[j+2])
			lv := (r + g + bl) / 3
			pv := (r+g)/2 - bl
			qv := r - g
			for cx := range nx {
				f := cosX[cx][x]
				rowL[cx] += lv * f
				rowP[cx] += pv * f
				rowQ[cx] += qv * f
			}
			// hasAlpha is loop-invariant, so this costs a predicted branch rather than a
			// quarter of the inner loop on the opaque images that covers almost always are.
			if hasAlpha {
				for cx := range nx {
					rowA[cx] += alpha * cosX[cx][x]
				}
			}
		}
		accumulate(lAcc, lTerms, rowL, cosY, y)
		accumulate(pAcc, pTerms, rowP, cosY, y)
		accumulate(qAcc, qTerms, rowQ, cosY, y)
		accumulate(aAcc, aTerms, rowA, cosY, y)
	}

	n := float64(w * h)
	lDC, lAC, lScale := normalize(lAcc, n)
	pDC, pAC, pScale := normalize(pAcc, n)
	qDC, qAC, qScale := normalize(qAcc, n)
	aDC, aAC, aScale := normalize(aAcc, n)

	return pack(w, h, hasAlpha, lx, ly,
		lDC, pDC, qDC, aDC, lScale, pScale, qScale, aScale, lAC, pAC, qAC, aAC), nil
}

// terms lists the (cx, cy) pairs of the reference's triangular coefficient region, in write order.
func terms(nx, ny int) []term {
	var ts []term
	for cy := range ny {
		for cx := 0; cx*ny < nx*(ny-cy); cx++ {
			ts = append(ts, term{cx, cy})
		}
	}
	return ts
}

func maxCX(groups ...[]term) int {
	m := 0
	for _, g := range groups {
		for _, t := range g {
			m = max(m, t.cx)
		}
	}
	return m
}

func maxCY(groups ...[]term) int {
	m := 0
	for _, g := range groups {
		for _, t := range g {
			m = max(m, t.cy)
		}
	}
	return m
}

// cosTable precomputes cos(pi/size * c * (i+0.5)) with the reference's exact expression, so the
// table values are bit-identical to recomputing them per coefficient.
func cosTable(n, size int) [][]float64 {
	t := make([][]float64, n)
	for c := range n {
		t[c] = make([]float64, size)
		for i := range size {
			t[c][i] = math.Cos(math.Pi / float64(size) * float64(c) * (float64(i) + 0.5))
		}
	}
	return t
}

func averageColor(rgba *image.NRGBA, w, h int) (r, g, b, a float64) {
	for y := range h {
		row := rgba.Pix[y*rgba.Stride:]
		for x := range w {
			j := x * 4
			alpha := float64(row[j+3]) / 255
			r += alpha / 255 * float64(row[j])
			g += alpha / 255 * float64(row[j+1])
			b += alpha / 255 * float64(row[j+2])
			a += alpha
		}
	}
	return r, g, b, a
}

// accumulate folds one row's per-cx sums into the term accumulators, so the pixel loop costs
// nx multiplies per pixel instead of one per coefficient.
func accumulate(acc []float64, ts []term, row []float64, cosY [][]float64, y int) {
	for k, t := range ts {
		acc[k] += row[t.cx] * cosY[t.cy][y]
	}
}

// normalize splits the accumulators into DC and scaled AC terms, matching the reference: a constant
// image leaves scale at 0, which skips normalization rather than mapping the terms to the midpoint.
func normalize(acc []float64, n float64) (dc float64, ac []float64, scale float64) {
	if len(acc) == 0 {
		return 0, nil, 0
	}
	dc = acc[0] / n
	ac = make([]float64, len(acc)-1)
	for i, v := range acc[1:] {
		ac[i] = v / n
		scale = math.Max(scale, math.Abs(ac[i]))
	}
	if scale > 0 {
		for i := range ac {
			ac[i] = 0.5 + 0.5/scale*ac[i]
		}
	}
	return dc, ac, scale
}

func pack(w, h int, hasAlpha bool, lx, ly int,
	lDC, pDC, qDC, aDC, lScale, pScale, qScale, aScale float64,
	lAC, pAC, qAC, aAC []float64,
) []byte {
	isLandscape := 0
	if w > h {
		isLandscape = 1
	}
	alphaBit := 0
	if hasAlpha {
		alphaBit = 1
	}
	header24 := int(math.Round(63*lDC)) | int(math.Round(31.5+31.5*pDC))<<6 |
		int(math.Round(31.5+31.5*qDC))<<12 | int(math.Round(31*lScale))<<18 | alphaBit<<23
	lead := lx
	if isLandscape == 1 {
		lead = ly
	}
	header16 := lead | int(math.Round(63*pScale))<<3 | int(math.Round(63*qScale))<<9 | isLandscape<<15

	acs := [][]float64{lAC, pAC, qAC}
	acStart := 5
	if hasAlpha {
		acs = append(acs, aAC)
		acStart = 6
	}
	acCount := 0
	for _, ac := range acs {
		acCount += len(ac)
	}

	hash := make([]byte, acStart+(acCount+1)/2)
	hash[0] = byte(header24 & 255)
	hash[1] = byte((header24 >> 8) & 255)
	hash[2] = byte(header24 >> 16)
	hash[3] = byte(header16 & 255)
	hash[4] = byte(header16 >> 8)
	if hasAlpha {
		hash[5] = byte(int(math.Round(15*aDC)) | int(math.Round(15*aScale))<<4)
	}
	acIndex := 0
	for _, ac := range acs {
		for _, f := range ac {
			hash[acStart+(acIndex>>1)] |= byte(int(math.Round(15*f)) << ((acIndex & 1) << 2))
			acIndex++
		}
	}
	return hash
}

// NRGBA, not RGBA: ThumbHash requires non-premultiplied RGB and the pipeline hands us a
// premultiplied *image.RGBA, which draw.Draw un-premultiplies on the way in.
func toNRGBA(img image.Image) *image.NRGBA {
	// The pixel loops index Pix from its start, so only an origin-anchored image can be used as-is.
	if nrgba, ok := img.(*image.NRGBA); ok && nrgba.Rect.Min == (image.Point{}) {
		return nrgba
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

func downscale(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxInputSize && h <= maxInputSize {
		return img
	}
	scale := float64(maxInputSize) / float64(max(w, h))
	dst := image.NewNRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}
