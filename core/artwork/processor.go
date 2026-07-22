package artwork

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"time"

	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	xdraw "golang.org/x/image/draw"
)

// outcome tells the worker what to do with the queue row: found/absent
// delete it, failed reschedules it via MarkFailed.
type outcome int

const (
	outcomeFound outcome = iota
	outcomeAbsent
	outcomeFailed
)

// thumbnailSize is the max dimension fed to blurhash.
const thumbnailSize = 128

// workerDeps are the collaborators processItem needs; extGate is set by NewWorker in
// production and nil only in tests, where resolveItem falls back to a plain passthrough.
type workerDeps struct {
	ds      model.DataStore
	store   *ImageStore
	prov    external.Provider
	ffmpeg  ffmpeg.FFmpeg
	extGate func(func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error)
}

// processItem resolves one queue item end to end: find an image, hash/decode/
// blurhash it, place its bytes, and persist the resulting state.
func processItem(ctx context.Context, deps *workerDeps, item model.ArtworkQueueItem) outcome {
	repo := deps.ds.Artwork(ctx)

	res, err := resolveItem(ctx, deps.ds, deps.prov, deps.ffmpeg, item, deps.extGate)
	if err != nil {
		log.Warn(ctx, "artwork: could not resolve item", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	if res.reader == nil {
		if res.extError {
			// An external source errored/timed out: never settle on absent, keep serving old state.
			return outcomeFailed
		}
		return writeAbsent(ctx, repo, item)
	}
	defer res.reader.Close()

	data, err := io.ReadAll(res.reader)
	if err != nil {
		log.Warn(ctx, "artwork: failed to read resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	log.Debug(ctx, "artwork: read resolved image", "kind", item.ItemKind, "id", item.ItemID, "source", res.source, "bytes", len(data))

	hash, err := HashImage(bytes.NewReader(data))
	if err != nil {
		log.Warn(ctx, "artwork: failed to hash image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}

	art, err := repo.GetImage(hash)
	switch {
	case err == nil:
		// Dedup hit: identical bytes already known, reuse dims/mime/blurhash.
	case errors.Is(err, model.ErrNotFound):
		art, err = decodeArtwork(ctx, hash, data)
		if err != nil {
			log.Warn(ctx, "artwork: failed to decode resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
			return outcomeFailed
		}
	default:
		log.Warn(ctx, "artwork: failed to look up image hash", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	art.SizeBytes = int64(len(data))

	if err := placeBytes(deps.store, art, res, data); err != nil {
		log.Warn(ctx, "artwork: failed to write image store", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	if err := repo.PutImage(art); err != nil {
		log.Warn(ctx, "artwork: failed to persist artwork image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	if err := repo.PutItemArtwork(&model.ItemArtwork{
		ItemKind:    item.ItemKind,
		ItemID:      item.ItemID,
		ImageType:   item.ImageType,
		Hash:        hash,
		Source:      res.source,
		AttemptedAt: time.Now(),
	}); err != nil {
		log.Warn(ctx, "artwork: failed to persist item artwork state", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	return outcomeFound
}

// writeAbsent records a known-absent state: every local/external source answered definitively "no".
func writeAbsent(ctx context.Context, repo model.ArtworkRepository, item model.ArtworkQueueItem) outcome {
	err := repo.PutItemArtwork(&model.ItemArtwork{
		ItemKind:    item.ItemKind,
		ItemID:      item.ItemID,
		ImageType:   item.ImageType,
		AttemptedAt: time.Now(),
	})
	if err != nil {
		log.Warn(ctx, "artwork: failed to persist absent state", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	return outcomeAbsent
}

// decodeArtwork builds a new Artwork row from raw bytes: dimensions, mime and a
// blurhash computed from a downscaled thumbnail.
func decodeArtwork(ctx context.Context, hash string, data []byte) (*model.Artwork, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image config: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	thumb := makeThumbnail(img, thumbnailSize)
	xComp, yComp := blurhash.Components(thumb.Bounds().Dx(), thumb.Bounds().Dy())
	bh, err := blurhash.Encode(thumb, xComp, yComp)
	if err != nil {
		log.Warn(ctx, "artwork: blurhash encoding failed", "hash", hash, err)
		bh = ""
	}

	return &model.Artwork{
		Hash:     hash,
		Mime:     mimeForFormat(format),
		Width:    cfg.Width,
		Height:   cfg.Height,
		BlurHash: bh,
	}, nil
}

// makeThumbnail downscales img to fit within maxSize on its longest side.
// Images within bounds are returned as-is (no upscaling).
func makeThumbnail(img image.Image, maxSize int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSize && h <= maxSize {
		return toFastScaleType(img)
	}
	scale := float64(maxSize) / float64(max(w, h))
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), toFastScaleType(img), b, draw.Src, nil)
	return dst
}

// isFileBacked reports whether a resolution's bytes already live in a library/upload
// file, so the acquisition must not duplicate them into the content-addressed store.
func isFileBacked(source string) bool {
	return source == "folder" || source == "upload"
}

// placeBytes fills in art's SourcePath/RefMtime and, for sources with no library file
// backing them, writes the bytes into the store (embedded keeps its audio file provenance).
func placeBytes(store *ImageStore, art *model.Artwork, res resolution, data []byte) error {
	if isFileBacked(res.source) {
		art.SourcePath = res.sourcePath
		art.RefMtime = res.refMtime
		return nil
	}
	art.SourcePath = ""
	art.RefMtime = 0
	if res.source == "embedded" {
		art.SourcePath = res.sourcePath
		art.RefMtime = res.refMtime
	}
	return store.Write(art.Hash, art.Mime, bytes.NewReader(data))
}

// mimeForFormat maps an image.Decode format name to its MIME type; extForMime
// in image_store.go performs the inverse for content-addressed file paths.
func mimeForFormat(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	}
	return "application/octet-stream"
}
