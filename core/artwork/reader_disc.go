package artwork

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type discArtworkReader struct {
	cacheKey
	a              *artwork
	album          model.Album
	discNumber     int
	imgFiles       []string
	discFolders    map[string]bool
	isMultiFolder  bool
	firstTrackPath string
	updatedAt      *time.Time
}

func newDiscArtworkReader(ctx context.Context, a *artwork, artID model.ArtworkID) (*discArtworkReader, error) {
	albumID, discNumber, err := model.ParseDiscArtworkID(artID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid disc artwork id '%s': %w", artID.ID, err)
	}

	al, err := a.ds.Album(ctx).Get(albumID)
	if err != nil {
		return nil, err
	}

	_, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, a.ds, *al)
	if err != nil {
		return nil, err
	}

	// Query mediafiles for this album + disc to find folder associations and first track
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Sort:    "track_number",
		Order:   "ASC",
		Filters: squirrel.Eq{"album_id": albumID, "disc_number": discNumber},
	})
	if err != nil {
		return nil, err
	}

	// Build disc folder set and find first track
	discFolders := make(map[string]bool)
	var firstTrackPath string
	allFolderIDs := make(map[string]bool)
	for _, mf := range mfs {
		allFolderIDs[mf.FolderID] = true
		if firstTrackPath == "" {
			firstTrackPath = mf.Path
		}
	}

	// Resolve folder IDs to absolute paths
	if len(allFolderIDs) > 0 {
		folderIDs := make([]string, 0, len(allFolderIDs))
		for id := range allFolderIDs {
			folderIDs = append(folderIDs, id)
		}
		folders, err := a.ds.Folder(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"folder.id": folderIDs},
		})
		if err != nil {
			return nil, err
		}
		for _, f := range folders {
			discFolders[f.AbsolutePath()] = true
		}
	}

	isMultiFolder := len(al.FolderIDs) > 1

	r := &discArtworkReader{
		a:              a,
		album:          *al,
		discNumber:     discNumber,
		imgFiles:       imgFiles,
		discFolders:    discFolders,
		isMultiFolder:  isMultiFolder,
		firstTrackPath: core.AbsolutePath(ctx, a.ds, al.LibraryID, firstTrackPath),
		updatedAt:      imagesUpdatedAt,
	}
	r.cacheKey.artID = artID
	if r.updatedAt != nil && r.updatedAt.After(al.UpdatedAt) {
		r.cacheKey.lastUpdate = *r.updatedAt
	} else {
		r.cacheKey.lastUpdate = al.UpdatedAt
	}
	return r, nil
}

func (d *discArtworkReader) Key() string {
	hash := md5.Sum([]byte(conf.Server.DiscArtPriority))
	return fmt.Sprintf(
		"%s.%x",
		d.cacheKey.Key(),
		hash,
	)
}

func (d *discArtworkReader) LastUpdated() time.Time {
	return d.album.UpdatedAt
}

func (d *discArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff = d.fromDiscArtPriority(ctx, d.a.ffmpeg, conf.Server.DiscArtPriority)
	// Fallback to album cover art
	albumArtID := model.NewArtworkID(model.KindAlbumArtwork, d.album.ID, &d.album.UpdatedAt)
	ff = append(ff, fromAlbum(ctx, d.a, albumArtID))
	return selectImageReader(ctx, d.cacheKey.artID, ff...)
}

func (d *discArtworkReader) fromDiscArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []sourceFunc {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			ff = append(ff, fromTag(ctx, d.firstTrackPath), fromFFmpegTag(ctx, ffmpeg, d.firstTrackPath))
		case pattern == "external":
			// Not supported for disc art, silently ignore
		case pattern == "discsubtitle":
			if subtitle := strings.TrimSpace(d.album.Discs[d.discNumber]); subtitle != "" {
				ff = append(ff, d.fromDiscSubtitle(ctx, subtitle))
			}
		case len(d.imgFiles) > 0:
			ff = append(ff, d.fromExternalFile(ctx, pattern))
		}
	}
	return ff
}

// fromDiscSubtitle returns a sourceFunc that matches image files whose stem
// (filename without extension) equals the disc subtitle (case-insensitive).
func (d *discArtworkReader) fromDiscSubtitle(ctx context.Context, subtitle string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		for _, file := range d.imgFiles {
			_, name := filepath.Split(file)
			stem := strings.TrimSuffix(name, filepath.Ext(name))
			if !strings.EqualFold(stem, subtitle) {
				continue
			}
			f, err := os.Open(file)
			if err != nil {
				log.Warn(ctx, "Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: no image file matching subtitle %q", d.discNumber, subtitle)
	}
}

// extractDiscNumber extracts a disc number from a filename based on a glob pattern.
// It finds the portion of the filename that the wildcard matched and parses leading
// digits as the disc number. Returns (0, false) if the pattern doesn't match or
// no leading digits are found in the wildcard portion.
func extractDiscNumber(pattern, filename string) (int, bool) {
	filename = strings.ToLower(filename)
	pattern = strings.ToLower(pattern)

	matched, err := filepath.Match(pattern, filename)
	if err != nil || !matched {
		return 0, false
	}

	// Find the prefix before the first '*' in the pattern
	starIdx := strings.IndexByte(pattern, '*')
	if starIdx < 0 {
		return 0, false
	}
	prefix := pattern[:starIdx]

	// Strip the prefix from the filename to get the wildcard-matched portion
	if !strings.HasPrefix(filename, prefix) {
		return 0, false
	}
	remainder := filename[len(prefix):]

	// Extract leading ASCII digits from the remainder
	var digits []byte
	for _, r := range remainder {
		if r >= '0' && r <= '9' {
			digits = append(digits, byte(r))
		} else {
			break
		}
	}

	if len(digits) == 0 {
		return 0, false
	}

	num, err := strconv.Atoi(string(digits))
	if err != nil {
		return 0, false
	}
	return num, true
}

// fromExternalFile returns a sourceFunc that matches image files against a glob
// pattern with disc-number-aware filtering.
//
// Matching rules:
//   - If a disc number can be extracted from the filename, the file matches only if
//     the number equals the target disc number.
//   - If no number is found and this is a multi-folder album, the file matches if
//     it's in a folder containing tracks for this disc.
//   - If no number is found and this is a single-folder album, the file is skipped
//     (ambiguous).
func (d *discArtworkReader) fromExternalFile(ctx context.Context, pattern string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		for _, file := range d.imgFiles {
			_, name := filepath.Split(file)
			match, err := filepath.Match(pattern, strings.ToLower(name))
			if err != nil {
				log.Warn(ctx, "Error matching disc art file to pattern", "pattern", pattern, "file", file)
				continue
			}
			if !match {
				continue
			}

			// Try to extract disc number from filename
			num, hasNum := extractDiscNumber(pattern, name)
			if hasNum {
				// File has a disc number — must match target disc
				if num != d.discNumber {
					continue
				}
			} else if d.isMultiFolder {
				// No number, multi-folder: match by folder association
				dir := filepath.Dir(file)
				if !d.discFolders[dir] {
					continue
				}
			} else {
				// No number, single-folder: ambiguous, skip
				continue
			}

			f, err := os.Open(file)
			if err != nil {
				log.Warn(ctx, "Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: pattern '%s' not matched by files", d.discNumber, pattern)
	}
}
