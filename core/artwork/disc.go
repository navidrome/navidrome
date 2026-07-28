package artwork

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
)

// discArtworkReader resolves disc-level artwork from a library's folder images
// and embedded tags. It is used by the serving path's provisional disc read-through.
type discArtworkReader struct {
	album          model.Album
	discNumber     int
	imgFiles       []string        // library-relative, forward-slash, no leading slash
	discFoldersRel map[string]bool // library-relative folder paths
	isMultiFolder  bool
	firstTrackRel  string // library-relative; for fromTag / ffmpeg via lib.Abs
	lib            libraryView
	// Newest ImagesUpdatedAt across the album's and this disc's folders: an image can be
	// replaced without the album row changing, so this is what makes a cache key notice it.
	imagesUpdatedAt time.Time
}

// cacheTime is the disc image's validity stamp: any of these moving means the selection may
// have changed.
func (d *discArtworkReader) cacheTime() time.Time {
	return utils.TimeNewest(d.album.UpdatedAt, d.album.ImportedAt, d.imagesUpdatedAt)
}

func newDiscArtworkReader(ctx context.Context, ds model.DataStore, artID model.ArtworkID) (*discArtworkReader, error) {
	albumID, discNumber, err := model.ParseDiscArtworkID(artID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid disc artwork id '%s': %w", artID.ID, err)
	}

	al, err := ds.Album(ctx).Get(albumID)
	if err != nil {
		return nil, err
	}

	_, imgFiles, albumImagesAt, err := loadAlbumFoldersPaths(ctx, ds, *al)
	if err != nil {
		return nil, err
	}

	var imagesUpdatedAt time.Time
	if albumImagesAt != nil {
		imagesUpdatedAt = *albumImagesAt
	}

	// Query mediafiles for this album + disc to find folder associations and first track
	mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Sort:    "track_number",
		Order:   "ASC",
		Filters: squirrel.Eq{"album_id": albumID, "disc_number": discNumber},
	})
	if err != nil {
		return nil, err
	}

	lib, err := loadLibraryView(ctx, ds, al.LibraryID)
	if err != nil {
		return nil, err
	}

	// Build disc folder set and find first track. mf.Path is already library-relative.
	var firstTrackRel string
	for _, mf := range mfs {
		if mf.Path != "" {
			firstTrackRel = filepath.ToSlash(mf.Path)
			break
		}
	}
	folderIDs := slice.Unique(slice.Map(mfs, func(mf model.MediaFile) string { return mf.FolderID }))

	// Resolve folder IDs to library-relative paths
	discFoldersRel := make(map[string]bool)
	if len(folderIDs) > 0 {
		folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"folder.id": folderIDs},
		})
		if err != nil {
			return nil, err
		}
		for _, f := range folders {
			rel := strings.TrimPrefix(path.Join(f.Path, f.Name), "/")
			discFoldersRel[rel] = true
			imagesUpdatedAt = utils.TimeNewest(imagesUpdatedAt, f.ImagesUpdatedAt)
		}
	}

	return &discArtworkReader{
		album:           *al,
		discNumber:      discNumber,
		imgFiles:        imgFiles,
		discFoldersRel:  discFoldersRel,
		isMultiFolder:   len(al.FolderIDs) > 1,
		firstTrackRel:   firstTrackRel,
		lib:             lib,
		imagesUpdatedAt: imagesUpdatedAt,
	}, nil
}

func (d *discArtworkReader) fromDiscArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []sourceFunc {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			ff = append(ff,
				fromTag(ctx, d.lib.FS, d.firstTrackRel),
				fromFFmpegTag(ctx, ffmpeg, d.lib.Abs(d.firstTrackRel)),
			)
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
			stem := utils.BaseName(file)
			if !strings.EqualFold(stem, subtitle) {
				continue
			}
			f, err := d.lib.FS.Open(file)
			if err != nil {
				log.Warn(ctx, "Artwork: Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: no image file matching subtitle %q", d.discNumber, subtitle)
	}
}

// filepath.Match's '\' escape is excluded on purpose: treating it as a metachar
// would misalign the literal-prefix extraction in extractDiscNumber.
const globMetaChars = "*?["

// extractDiscNumber parses the disc number from a filename matched by a filepath.Match-style
// glob. Caller must lowercase both args and have already verified the match.
func extractDiscNumber(pattern, filename string) (int, bool) {
	metaIdx := strings.IndexAny(pattern, globMetaChars)
	if metaIdx < 0 {
		return 0, false
	}
	prefix := pattern[:metaIdx]
	if !strings.HasPrefix(filename, prefix) {
		return 0, false
	}

	start := len(prefix)
	end := start
	for end < len(filename) && filename[end] >= '0' && filename[end] <= '9' {
		end++
	}
	if end == start {
		return 0, false
	}
	num, err := strconv.Atoi(filename[start:end])
	if err != nil {
		return 0, false
	}
	return num, true
}

// fromExternalFile matches image files against a (lowercase) glob pattern. A numbered
// filename whose number equals the target disc wins over any unnumbered candidate.
func (d *discArtworkReader) fromExternalFile(ctx context.Context, pattern string) sourceFunc {
	isLiteral := !strings.ContainsAny(pattern, globMetaChars)
	return func() (io.ReadCloser, string, error) {
		var fallbacks []string
		for _, file := range d.imgFiles {
			name := strings.ToLower(path.Base(file))
			match, err := filepath.Match(pattern, name)
			if err != nil {
				log.Warn(ctx, "Artwork: Error matching disc art file to pattern", "pattern", pattern, "file", file)
				continue
			}
			if !match {
				continue
			}

			if !isLiteral {
				if num, hasNum := extractDiscNumber(pattern, name); hasNum {
					if num != d.discNumber {
						continue
					}
					f, err := d.lib.FS.Open(file)
					if err != nil {
						log.Warn(ctx, "Artwork: Could not open disc art file", "file", file, err)
						continue
					}
					return f, file, nil
				}
			}

			if d.isMultiFolder && !d.discFoldersRel[path.Dir(file)] {
				continue
			}
			fallbacks = append(fallbacks, file)
		}

		for _, file := range fallbacks {
			f, err := d.lib.FS.Open(file)
			if err != nil {
				log.Warn(ctx, "Artwork: Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: pattern '%s' not matched by files", d.discNumber, pattern)
	}
}
