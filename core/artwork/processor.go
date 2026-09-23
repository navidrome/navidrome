package artwork

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // the only artwork format with no other importer in this package
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/core/artwork/dominant"
	"github.com/navidrome/navidrome/core/artwork/thumbhash"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	xdraw "golang.org/x/image/draw"
)

// outcome tells the worker whether to delete the queue row (found/absent) or reschedule it.
type outcome int

const (
	outcomeFound outcome = iota
	// outcomeFoundStale: state was written and is served, but a higher-priority external
	// source failed, so the row must retry to give it another chance.
	outcomeFoundStale
	outcomeAbsent
	outcomeFailed
)

func (o outcome) String() string {
	switch o {
	case outcomeFound:
		return "found"
	case outcomeFoundStale:
		return "foundStale"
	case outcomeAbsent:
		return "absent"
	default:
		return "failed"
	}
}

// thumbnailSize is the max dimension fed to both hash encoders; thumbhash rejects anything larger.
const thumbnailSize = 100

// maxImageBytes caps a resolved image read: a user-editable ExternalImageURL could point at
// an arbitrarily large endpoint.
func maxImageBytes() int64 {
	return parseSize(conf.Server.MaxImageSize, consts.DefaultMaxImageSize)
}

// maxImagePixels guards against decompression bombs: a tiny file can declare a canvas that
// image.Decode would expand into gigabytes.
const maxImagePixels = 64 << 20

// acquired is what a successful acquire persisted, handed back so the caller can warm the resize
// cache without re-reading the rows and the file it just wrote.
type acquired struct {
	ia   *model.ItemArtwork
	mime string
	data []byte
}

// processor turns one queue item into stored artwork; settling the queue row is the Worker's job.
type processor struct {
	ds        model.DataStore
	store     *ImageStore
	resolver  *resolver
	pruneLock sync.Locker
}

// acquire resolves one queue item end to end: find an image, hash/decode/
// blurhash it, place its bytes, and persist the resulting state.
func (p *processor) acquire(ctx context.Context, item model.ArtworkQueueItem) (out outcome, got *acquired, retryIn time.Duration) {
	repo := p.ds.Artwork(ctx)
	start := time.Now()
	defer func() {
		log.Debug(ctx, "Artwork: Acquisition finished", "kind", item.ItemKind, "id", item.ItemID,
			"outcome", out, "elapsed", time.Since(start))
	}()

	res, err := p.resolver.resolve(ctx, item)
	if err != nil {
		traceStage(ctx, "resolve", err)
		log.Warn(ctx, "Artwork: Could not resolve item", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil, 0
	}
	if retry, ok := errors.AsType[*agents.RetryLaterError](res.extErr); ok {
		retryIn = retry.RetryIn
	}
	if res.reader == nil {
		if res.extErr != nil || res.localError {
			// A fault is not a definitive "no image": never settle absent, keep serving old state.
			// A chainless resolver (playlist/radio) records no step, so leave a fallback or explain is blank.
			if t := traceFrom(ctx); len(t.Steps()) == 0 {
				outcome := OutcomeError
				if res.localError {
					outcome = OutcomeUnreadable
				}
				t.add(TraceStep{Candidate: cmp.Or(res.source, "source"), Outcome: outcome})
			}
			log.Debug(ctx, "Artwork: No image, but a source faulted; keeping previous state",
				"kind", item.ItemKind, "id", item.ItemID, "extErr", res.extErr, "localError", res.localError)
			return outcomeFailed, nil, retryIn
		}
		return writeAbsent(ctx, repo, item), nil, 0
	}
	defer res.reader.Close()

	readStart := time.Now()
	data, err := readCapped(res.reader)
	if err != nil {
		traceStage(ctx, "read", err)
		log.Warn(ctx, "Artwork: Failed to read resolved image", "kind", item.ItemKind, "id", item.ItemID, "source", res.source, err)
		return outcomeFailed, nil, retryIn
	}
	log.Debug(ctx, "Artwork: Read resolved image", "kind", item.ItemKind, "id", item.ItemID,
		"source", res.source, "bytes", len(data), "elapsed", time.Since(readStart))

	hashStart := time.Now()
	hash, err := hashImage(bytes.NewReader(data))
	if err != nil {
		traceStage(ctx, "hash", err)
		log.Warn(ctx, "Artwork: Failed to hash image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil, retryIn
	}
	log.Trace(ctx, "Artwork: Hashed image", "kind", item.ItemKind, "id", item.ItemID,
		"hash", hash, "bytes", len(data), "elapsed", time.Since(hashStart))

	art, err := repo.GetImage(hash)
	switch {
	case err == nil && art.Width > 0:
		log.Debug(ctx, "Artwork: Reusing a known image, skipping decode", "kind", item.ItemKind,
			"id", item.ItemID, "hash", hash)
	// A row with no dimensions was stored when no decoder matched; retry in case one exists now.
	case err == nil, errors.Is(err, model.ErrNotFound):
		decodeStart := time.Now()
		art, err = decodeArtwork(ctx, hash, data)
		// Extension-matched local bytes we cannot decode are most likely a codec we lack; an
		// external body carries no such guarantee, and empty bytes are no image at all.
		if errors.Is(err, image.ErrFormat) && len(data) > 0 && isLocalSource(res.source) {
			log.Debug(ctx, "Artwork: No decoder for this image format, storing it without placeholders",
				"kind", item.ItemKind, "id", item.ItemID, "source", res.source, "bytes", len(data))
			art, err = undecodedArtwork(hash), nil
		}
		if err != nil {
			traceStage(ctx, "decode", err)
			log.Warn(ctx, "Artwork: Failed to decode resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
			return outcomeFailed, nil, retryIn
		}
		log.Debug(ctx, "Artwork: Decoded new image", "kind", item.ItemKind, "id", item.ItemID, "hash", hash,
			"width", art.Width, "height", art.Height, "mime", art.Mime, "elapsed", time.Since(decodeStart))
	default:
		traceStage(ctx, "lookup", err)
		log.Warn(ctx, "Artwork: Failed to look up image hash", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil, retryIn
	}
	art.SizeBytes = int64(len(data))

	ia, err := p.persist(ctx, repo, item, art, res, data)
	if err != nil {
		traceStage(ctx, "store", err)
		log.Warn(ctx, "Artwork: Failed to persist resolved image", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed, nil, retryIn
	}
	got = &acquired{ia: ia, mime: art.Mime, data: data}
	if res.extErr != nil {
		log.Debug(ctx, "Artwork: Serving a lower-priority source after an external failure",
			"kind", item.ItemKind, "id", item.ItemID, "source", res.source)
		return outcomeFoundStale, got, retryIn
	}
	return outcomeFound, got, retryIn
}

// persist places the bytes and commits the rows referencing them, excluding Prune for that
// window only so a slow resolution can never hold it off.
func (p *processor) persist(ctx context.Context, repo model.ArtworkRepository, item model.ArtworkQueueItem,
	art *model.Artwork, res resolution, data []byte,
) (*model.ItemArtwork, error) {
	if p.pruneLock != nil {
		p.pruneLock.Lock()
		defer p.pruneLock.Unlock()
	}
	sourcePath, refMtime, err := placeBytes(p.store, art, res, data)
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
		Hash:        art.Hash,
		Source:      res.source,
		SourcePath:  sourcePath,
		RefMtime:    refMtime,
		AttemptedAt: time.Now(),
		Trace:       traceFrom(ctx).encode(sourcePath),
	}
	// PutItemArtwork stamps UpdatedAt on ia, so the returned struct matches the persisted row.
	if err := repo.PutItemArtwork(ia); err != nil {
		return nil, fmt.Errorf("persisting item artwork state: %w", err)
	}
	return ia, nil
}

// writeAbsent records a known-absent state: every source answered definitively "no".
func writeAbsent(ctx context.Context, repo model.ArtworkRepository, item model.ArtworkQueueItem) outcome {
	err := repo.PutItemArtwork(&model.ItemArtwork{
		ItemKind:    item.ItemKind,
		ItemID:      item.ItemID,
		ImageType:   item.ImageType,
		AttemptedAt: time.Now(),
		Trace:       traceFrom(ctx).encode(""),
	})
	if err != nil {
		log.Warn(ctx, "Artwork: Failed to persist absent state", "kind", item.ItemKind, "id", item.ItemID, err)
		return outcomeFailed
	}
	log.Debug(ctx, "Artwork: Settled absent, every source answered definitively",
		"kind", item.ItemKind, "id", item.ItemID)
	return outcomeAbsent
}

func readCapped(r io.Reader) ([]byte, error) {
	limit := maxImageBytes()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("image exceeds size cap %d", limit)
	}
	return data, nil
}

// decodeCapped rejects declared dimensions over maxImagePixels before the full-decode allocation.
func decodeCapped(data []byte) (image.Image, string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image config: %w", err)
	}
	// Compared by division so the cap cannot be defeated by an int64 overflow.
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImagePixels/cfg.Height {
		return nil, "", fmt.Errorf("image dimensions %dx%d exceed pixel cap %d", cfg.Width, cfg.Height, maxImagePixels)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}
	return img, format, nil
}

// undecodedArtwork is the row for bytes no decoder matched: servable, but with no dimensions
// and none of the placeholders a decode would have produced.
func undecodedArtwork(hash string) *model.Artwork {
	return &model.Artwork{Hash: hash, Mime: mimeForFormat("")}
}

// decodeArtwork builds a new Artwork row from raw bytes: dimensions, mime and the two
// placeholder hashes, both encoded from one shared downscaled thumbnail.
func decodeArtwork(ctx context.Context, hash string, data []byte) (*model.Artwork, error) {
	img, format, err := decodeCapped(data)
	if err != nil {
		return nil, err
	}

	thumb := makeThumbnail(img, thumbnailSize)
	bh, err := blurhash.Encode(thumb)
	if err != nil {
		log.Warn(ctx, "Artwork: Blurhash encoding failed", "hash", hash, err)
		bh = ""
	}

	var th string
	if raw, err := thumbhash.Encode(thumb); err != nil {
		log.Warn(ctx, "Artwork: Thumbhash encoding failed", "hash", hash, err)
	} else {
		th = base64.StdEncoding.EncodeToString(raw)
	}

	return &model.Artwork{
		Hash:          hash,
		Mime:          mimeForFormat(format),
		Width:         img.Bounds().Dx(),
		Height:        img.Bounds().Dy(),
		BlurHash:      bh,
		ThumbHash:     th,
		DominantColor: dominant.Color(thumb),
	}, nil
}

// makeThumbnail downscales img to fit within maxSize on its longest side; it never upscales.
func makeThumbnail(img image.Image, maxSize int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSize && h <= maxSize {
		return toFastScaleType(img)
	}
	scale := float64(maxSize) / float64(max(w, h))
	// NRGBA, not RGBA: thumbhash requires straight alpha, and blurhash reads this type without
	// converting, so neither encoder allocates a second copy of the thumbnail.
	dst := image.NewNRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), toFastScaleType(img), b, draw.Src, nil)
	return dst
}

// isFileBacked reports whether the bytes already live in a library/upload file, so the
// content-addressed store must not duplicate them.
func isFileBacked(source string) bool {
	return source == "folder" || source == "upload"
}

// isLocalSource reports whether the bytes came off disk rather than off the network.
func isLocalSource(source string) bool {
	return isFileBacked(source) || source == "embedded"
}

// placeBytes reports the item's backing-file provenance and writes the bytes into the store
// for the sources that have none.
func placeBytes(store *ImageStore, art *model.Artwork, res resolution, data []byte) (sourcePath string, refMtime int64, err error) {
	if isFileBacked(res.source) {
		return res.sourcePath, res.refMtime, nil
	}
	if res.source == "embedded" {
		sourcePath, refMtime = res.sourcePath, res.refMtime
	}
	return sourcePath, refMtime, store.Write(art.Hash, art.Mime, bytes.NewReader(data))
}

// mimeForFormat maps an image.Decode format name to its MIME type; extForMime is the inverse.
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
