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

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
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
	// outcomeFoundStale: state was written and is served, but a higher-priority external
	// step failed, so the row must retry (via MarkFailed) to give that source another chance.
	outcomeFoundStale
	outcomeAbsent
	outcomeFailed
)

// thumbnailSize is the max dimension fed to blurhash.
const thumbnailSize = 128

// maxImageBytes caps a resolved image read: a user-editable ExternalImageURL could
// point at an arbitrarily large endpoint, and 20MB is generous for any real cover.
const maxImageBytes = 20 << 20

// maxImagePixels caps declared dimensions: a tiny compressed file can declare a
// huge canvas that image.Decode would expand into gigabytes (decompression bomb).
const maxImagePixels = 64 << 20

// workerDeps are the collaborators processItem needs; gate is set by NewWorker in
// production and nil only in tests, where resolveItem falls back to a plain passthrough.
type workerDeps struct {
	ds     model.DataStore
	store  *ImageStore
	agents *agents.Agents
	ffmpeg ffmpeg.FFmpeg
	gate   gateFunc
}

// processItem resolves one queue item end to end: find an image, hash/decode/
// blurhash it, place its bytes, and persist the resulting state.
func processItem(ctx context.Context, deps *workerDeps, item model.ArtworkQueueItem) outcome {
	repo := deps.ds.Artwork(ctx)

	res, err := resolveItem(ctx, deps.ds, deps.agents, deps.ffmpeg, item, deps.gate)
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

	data, err := readCapped(res.reader)
	if err != nil {
		log.Warn(ctx, "artwork: failed to read resolved image", "kind", item.ItemKind, "id", item.ItemID, "source", res.source, err)
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

	sourcePath, refMtime, err := placeBytes(deps.store, art, res, data)
	if err != nil {
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
		SourcePath:  sourcePath,
		RefMtime:    refMtime,
		AttemptedAt: time.Now(),
	}); err != nil {
		log.Warn(ctx, "artwork: failed to persist item artwork state", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	if res.extError {
		return outcomeFoundStale
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

// readCapped reads r, rejecting anything over maxImageBytes.
func readCapped(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds size cap %d", maxImageBytes)
	}
	return data, nil
}

// decodeCapped rejects declared dimensions over maxImagePixels BEFORE the
// full-decode allocation, then decodes.
func decodeCapped(data []byte) (image.Image, string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image config: %w", err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, "", fmt.Errorf("image dimensions %dx%d exceed pixel cap %d", cfg.Width, cfg.Height, maxImagePixels)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}
	return img, format, nil
}

// decodeArtwork builds a new Artwork row from raw bytes: dimensions, mime and a
// blurhash computed from a downscaled thumbnail.
func decodeArtwork(ctx context.Context, hash string, data []byte) (*model.Artwork, error) {
	img, format, err := decodeCapped(data)
	if err != nil {
		return nil, err
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
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
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

// placeBytes reports the item's backing-file provenance (folder/upload: image, embedded: audio,
// external/generated: none) and writes the bytes into the store for the non-file-backed sources.
func placeBytes(store *ImageStore, art *model.Artwork, res resolution, data []byte) (sourcePath string, refMtime int64, err error) {
	if isFileBacked(res.source) {
		return res.sourcePath, res.refMtime, nil
	}
	if res.source == "embedded" {
		sourcePath, refMtime = res.sourcePath, res.refMtime
	}
	return sourcePath, refMtime, store.Write(art.Hash, art.Mime, bytes.NewReader(data))
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
