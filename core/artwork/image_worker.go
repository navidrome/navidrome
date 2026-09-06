package artwork

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/core/metadataworker"
	"github.com/navidrome/navidrome/core/metadataworker/gen"
)

type imageWorkerRequest struct {
	InputSize    int    `json:"input_size,omitempty"`
	InputSizes   []int  `json:"input_sizes,omitempty"`
	Mosaic       bool   `json:"mosaic,omitempty"`
	Sniff        bool   `json:"sniff,omitempty"`
	Size         int    `json:"size"`
	Square       bool   `json:"square"`
	Fill         bool   `json:"fill,omitempty"` // center-crop fill mode for playlist tiles
	AnimatedGIF  bool   `json:"animated_gif,omitempty"`
	AnimatedWebP bool   `json:"animated_webp,omitempty"`
	AnimatedPNG  bool   `json:"animated_png,omitempty"`
	Quality      int    `json:"quality"`
	Format       string `json:"format,omitempty"`
	Path         string `json:"path,omitempty"`
}

type imageAnimationFlags struct {
	AnimatedGIF  bool
	AnimatedWebP bool
	AnimatedPNG  bool
}

type imageWorkerPool struct{}

var persistentImageWorkers = &imageWorkerPool{}

func (p *imageWorkerPool) resize(ctx context.Context, data []byte, size, quality int, square bool, format string) ([]byte, error) {
	return p.resizeRequest(ctx, [][]byte{data}, imageWorkerRequest{
		InputSize: len(data),
		Size:      size,
		Square:    square,
		Quality:   quality,
		Format:    format,
	})
}

func (p *imageWorkerPool) resizePath(ctx context.Context, path string, size, quality int, square bool, format string) ([]byte, error) {
	return p.resizeRequest(ctx, nil, imageWorkerRequest{
		Path:    path,
		Size:    size,
		Square:  square,
		Quality: quality,
		Format:  format,
	})
}

func (p *imageWorkerPool) resizeAnimatedGIF(ctx context.Context, data []byte, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, [][]byte{data}, imageWorkerRequest{
		InputSize:   len(data),
		Size:        size,
		AnimatedGIF: true,
		Quality:     quality,
	})
}

func (p *imageWorkerPool) resizeAnimatedGIFPath(ctx context.Context, path string, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, nil, imageWorkerRequest{
		Path:        path,
		Size:        size,
		AnimatedGIF: true,
		Quality:     quality,
	})
}

func (p *imageWorkerPool) resizeAnimatedWebP(ctx context.Context, data []byte, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, [][]byte{data}, imageWorkerRequest{
		InputSize:    len(data),
		Size:         size,
		AnimatedWebP: true,
		Quality:      quality,
	})
}

func (p *imageWorkerPool) resizeAnimatedWebPPath(ctx context.Context, path string, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, nil, imageWorkerRequest{
		Path:         path,
		Size:         size,
		AnimatedWebP: true,
		Quality:      quality,
	})
}

func (p *imageWorkerPool) resizeAnimatedPNG(ctx context.Context, data []byte, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, [][]byte{data}, imageWorkerRequest{
		InputSize:   len(data),
		Size:        size,
		AnimatedPNG: true,
		Quality:     quality,
	})
}

func (p *imageWorkerPool) resizeAnimatedPNGPath(ctx context.Context, path string, size, quality int) ([]byte, error) {
	return p.resizeRequest(ctx, nil, imageWorkerRequest{
		Path:        path,
		Size:        size,
		AnimatedPNG: true,
		Quality:     quality,
	})
}

// mosaic fills and stitches 1 or 4 album covers into a playlist mosaic in one Rust round-trip.
func (p *imageWorkerPool) mosaic(ctx context.Context, tiles [][]byte, size, quality int, format string) ([]byte, error) {
	if len(tiles) == 0 || len(tiles) > 4 {
		return nil, fmt.Errorf("mosaic requires 1..=4 tiles, got %d", len(tiles))
	}
	sizes := make([]int, len(tiles))
	for i, tile := range tiles {
		sizes[i] = len(tile)
	}
	return p.resizeRequest(ctx, tiles, imageWorkerRequest{
		InputSizes: sizes,
		Mosaic:     true,
		Size:       size,
		Quality:    quality,
		Format:     format,
	})
}

func (p *imageWorkerPool) sniffAnimation(ctx context.Context, data []byte) (imageAnimationFlags, error) {
	return sniffViaGRPC(ctx, [][]byte{data}, imageWorkerRequest{Sniff: true, InputSize: len(data)})
}

func (p *imageWorkerPool) sniffAnimationPath(ctx context.Context, path string) (imageAnimationFlags, error) {
	return sniffViaGRPC(ctx, nil, imageWorkerRequest{Sniff: true, Path: path})
}

func (p *imageWorkerPool) resizeRequest(ctx context.Context, payloads [][]byte, request imageWorkerRequest) ([]byte, error) {
	return resizeViaGRPC(ctx, payloads, request)
}

func toProtoImageRequest(request imageWorkerRequest, payloads [][]byte) *gen.ImageRequest {
	return &gen.ImageRequest{
		Payloads:     payloads,
		Mosaic:       request.Mosaic,
		Sniff:        request.Sniff,
		Size:         uint32(max(request.Size, 0)),
		Square:       request.Square,
		Fill:         request.Fill,
		AnimatedGif:  request.AnimatedGIF,
		AnimatedWebp: request.AnimatedWebP,
		AnimatedPng:  request.AnimatedPNG,
		Quality:      uint32(max(request.Quality, 0)),
		Format:       request.Format,
		Path:         request.Path,
	}
}

func resizeViaGRPC(ctx context.Context, payloads [][]byte, request imageWorkerRequest) ([]byte, error) {
	resp, err := metadataworker.ProcessImage(ctx, toProtoImageRequest(request, payloads))
	if err != nil {
		return nil, err
	}
	return resp.GetBody(), nil
}

func sniffViaGRPC(ctx context.Context, payloads [][]byte, request imageWorkerRequest) (imageAnimationFlags, error) {
	var flags imageAnimationFlags
	resp, err := metadataworker.ProcessImage(ctx, toProtoImageRequest(request, payloads))
	if err != nil {
		return flags, err
	}
	return imageAnimationFlags{
		AnimatedGIF:  resp.GetAnimatedGif(),
		AnimatedWebP: resp.GetAnimatedWebp(),
		AnimatedPNG:  resp.GetAnimatedPng(),
	}, nil
}

type imageResizeError struct {
	message string
}

func (e *imageResizeError) Error() string {
	return e.message
}
