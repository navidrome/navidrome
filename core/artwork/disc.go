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

// discCandidate is one DiscArtPriority entry. skip is set when the entry maps to no source at
// all, so a chain walk can say why instead of leaving a configured entry unaccounted for.
type discCandidate struct {
	pattern string
	sources []sourceFunc
	skip    string
}

func (d *discArtworkReader) discCandidates(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []discCandidate {
	var cc []discCandidate
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		c := discCandidate{pattern: pattern}
		switch {
		case pattern == "embedded":
			c.sources = []sourceFunc{
				fromTag(ctx, d.lib.FS, d.firstTrackRel),
				fromFFmpegTag(ctx, ffmpeg, d.lib.Abs(d.firstTrackRel)),
			}
		case pattern == externalCandidate:
			c.skip = "external sources are not supported for disc artwork"
		case pattern == "discsubtitle":
			subtitle := strings.TrimSpace(d.album.Discs[d.discNumber])
			if subtitle == "" {
				c.skip = "disc has no subtitle"
			} else {
				c.sources = []sourceFunc{d.fromDiscSubtitle(ctx, subtitle)}
			}
		case len(d.imgFiles) == 0:
			c.skip = "no images in album folder"
		default:
			c.sources = []sourceFunc{d.fromExternalFile(ctx, pattern)}
		}
		cc = append(cc, c)
	}
	return cc
}

// selectImage walks the DiscArtPriority entries and returns the first that yields an image.
// chain records the walk; the serving path passes an untraced one and pays nothing for it.
func (d *discArtworkReader) selectImage(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string,
	chain *chainState) (resolution, error) {
	for _, c := range d.discCandidates(ctx, ffmpeg, priority) {
		if err := ctx.Err(); err != nil {
			return resolution{}, err
		}
		if c.skip != "" {
			chain.record(c.pattern, OutcomeSkipped, c.skip)
			continue
		}
		res, ok := d.openCandidate(ctx, c)
		if res, ok = chain.try(c.pattern, res, ok); ok {
			return res, nil
		}
	}
	return chain.exhausted(), nil
}

func (d *discArtworkReader) openCandidate(ctx context.Context, c discCandidate) (resolution, bool) {
	source := "folder"
	if c.pattern == "embedded" {
		source = "embedded"
	}
	for _, sf := range c.sources {
		start := time.Now()
		rd, path, err := sf()
		if rd == nil {
			log.Trace(ctx, "Artwork: Failed trying to extract disc artwork", "albumID", d.album.ID,
				"disc", d.discNumber, "pattern", c.pattern, "source", sf, "elapsed", time.Since(start), err)
			continue
		}
		// The disc sources disagree on this: only ffmpeg hands back an absolute path.
		if !filepath.IsAbs(path) {
			path = d.lib.Abs(path)
		}
		log.Debug(ctx, "Artwork: Found disc artwork", "albumID", d.album.ID, "disc", d.discNumber,
			"pattern", c.pattern, "path", path, "elapsed", time.Since(start))
		return resolution{reader: rd, source: source, sourcePath: path}, true
	}
	return resolution{}, false
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
