package ffmpeg

import (
	"context"

	"github.com/navidrome/navidrome/core/metadataworker"
)

type pictureWorkerPool struct{}

var persistentPictureWorkers = &pictureWorkerPool{}

func (p *pictureWorkerPool) extract(ctx context.Context, binary, path string, maxBytes int64) ([]byte, error) {
	_ = binary
	data, err := metadataworker.ExtractPicture(ctx, path, maxBytes)
	if err != nil {
		return nil, &pictureExtractionError{message: err.Error()}
	}
	return data, nil
}

type pictureExtractionError struct {
	message string
}

func (e *pictureExtractionError) Error() string {
	return e.message
}

func (p *pictureWorkerPool) closeIdle() {}
