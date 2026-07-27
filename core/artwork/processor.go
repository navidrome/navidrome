package artwork

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // the only artwork format with no other importer in this package
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
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

// workerDeps are the collaborators processItem needs; gate and pruneLock are set by NewWorker
// in production and nil only in tests, where resolveItem falls back to a plain passthrough and
// nothing prunes.
type workerDeps struct {
	ds        model.DataStore
	store     *ImageStore
	agents    *agents.Agents
	ffmpeg    ffmpeg.FFmpeg
	cache     cache.FileCache
	gate      gateFunc
	pruneLock sync.Locker
}

// acquired is what processItem persisted, handed back so the caller can warm the resize
// cache without re-reading the rows and the file it just wrote.
type acquired struct {
	ia   *model.ItemArtwork
	mime string
	data []byte
}

// processItem resolves one queue item end to end: find an image, hash/decode/
// blurhash it, place its bytes, and persist the resulting state.
func processItem(ctx context.Context, deps *workerDeps, item model.ArtworkQueueItem) (outcome, *acquired) {
	repo := deps.ds.Artwork(ctx)

	res, err := newResolver(deps.ds, deps.agents, deps.ffmpeg, deps.gate).resolve(ctx, item)
	if err != nil {
		log.Warn(ctx, "Artwork: Could not resolve item", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil
	}
	if res.reader == nil {
		if res.extError || res.localError {
			// A source errored/timed out rather than answering "no image": never settle on
			// absent, keep serving old state.
			return outcomeFailed, nil
		}
		return writeAbsent(ctx, repo, item), nil
	}
	defer res.reader.Close()

	data, err := readCapped(res.reader)
	if err != nil {
		log.Warn(ctx, "Artwork: Failed to read resolved image", "kind", item.ItemKind, "id", item.ItemID, "source", res.source, err)
		return outcomeFailed, nil
	}
	log.Debug(ctx, "Artwork: Read resolved image", "kind", item.ItemKind, "id", item.ItemID, "source", res.source, "bytes", len(data))

	hash, err := HashImage(bytes.NewReader(data))
	if err != nil {
		log.Warn(ctx, "Artwork: Failed to hash image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil
	}

	art, err := repo.GetImage(hash)
	switch {
	case err == nil:
		// Dedup hit: identical bytes already known, reuse dims/mime/blurhash.
	case errors.Is(err, model.ErrNotFound):
		art, err = decodeArtwork(ctx, hash, data)
		if err != nil {
			log.Warn(ctx, "Artwork: Failed to decode resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
			return outcomeFailed, nil
		}
	default:
		log.Warn(ctx, "Artwork: Failed to look up image hash", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil
	}
	art.SizeBytes = int64(len(data))

	ia, err := persist(deps, repo, item, hash, art, res, data)
	if err != nil {
		log.Warn(ctx, "Artwork: Failed to persist resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil
	}
	got := &acquired{ia: ia, mime: art.Mime, data: data}
	if res.extError {
		return outcomeFoundStale, got
	}
	return outcomeFound, got
}

// persist places the bytes and commits the rows referencing them. Only this window excludes
// Prune, which reclaims store files no row points at; resolution stays outside so a slow fetch
// cannot hold prune off.
func persist(deps *workerDeps, repo model.ArtworkRepository, item model.ArtworkQueueItem, hash string,
	art *model.Artwork, res resolution, data []byte,
) (*model.ItemArtwork, error) {
	if deps.pruneLock != nil {
		deps.pruneLock.Lock()
		defer deps.pruneLock.Unlock()
	}
	sourcePath, refMtime, err := placeBytes(deps.store, art, res, data)
	if err != nil {
		return nil, fmt.Errorf("writing image store: %w", err)
	}
	if err := repo.PutImage(art); err != nil {
		return nil, fmt.Errorf("persisting artwork image: %w", err)
	}
	ia := &model.ItemArtwork{
		ItemKind:    item.ItemKind,
		ItemID:      item.ItemID,
		ImageType:   item.ImageType,
		Hash:        hash,
		Source:      res.source,
		SourcePath:  sourcePath,
		RefMtime:    refMtime,
		AttemptedAt: time.Now(),
	}
	// PutItemArtwork stamps UpdatedAt on ia, so what it holds now matches the persisted row.
	if err := repo.PutItemArtwork(ia); err != nil {
		return nil, fmt.Errorf("persisting item artwork state: %w", err)
	}
	return ia, nil
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
		log.Warn(ctx, "Artwork: Failed to persist absent state", "kind", item.ItemKind, "id", item.ItemID, err)
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
	// Compared by division so the cap holds for any dimensions a decoder might report, without
	// depending on a multiplication staying inside int64.
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImagePixels/cfg.Height {
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
		log.Warn(ctx, "Artwork: Blurhash encoding failed", "hash", hash, err)
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
