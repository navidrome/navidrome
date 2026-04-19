package artwork

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type discArtworkReader struct {
	cacheKey
	a              *artwork
	album          model.Album
	discNumber     int
	imgFiles       []string        // library-relative, forward-slash, no leading slash
	discFoldersRel map[string]bool // library-relative folder paths
	isMultiFolder  bool
	firstTrackRel  string // library-relative; for fromTag / ffmpeg via lib.Abs
	lib            libraryView
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

	lib, err := loadLibraryView(ctx, a.ds, al.LibraryID)
	if err != nil {
		return nil, err
	}

	// Build disc folder set and find first track. mf.Path is already library-relative.
	var firstTrackRel string
	allFolderIDs := make(map[string]bool)
	for _, mf := range mfs {
		allFolderIDs[mf.FolderID] = true
		if firstTrackRel == "" {
			firstTrackRel = filepath.ToSlash(mf.Path)
		}
	}

	// Resolve folder IDs to library-relative paths
	discFoldersRel := make(map[string]bool)
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
			rel := strings.TrimPrefix(path.Join(f.Path, f.Name), "/")
			discFoldersRel[rel] = true
		}
	}

	isMultiFolder := len(al.FolderIDs) > 1

	r := &discArtworkReader{
		a:              a,
		album:          *al,
		discNumber:     discNumber,
		imgFiles:       imgFiles,
		discFoldersRel: discFoldersRel,
		isMultiFolder:  isMultiFolder,
		firstTrackRel:  firstTrackRel,
		lib:            lib,
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
	return d.lastUpdate
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
			name := path.Base(file)
			stem := strings.TrimSuffix(name, path.Ext(name))
			if !strings.EqualFold(stem, subtitle) {
				continue
			}
			f, err := d.lib.FS.Open(file)
			if err != nil {
				log.Warn(ctx, "Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: no image file matching subtitle %q", d.discNumber, subtitle)
	}
}

// globMetaChars holds the substitution metacharacters understood by
// filepath.Match. The '\' escape character is intentionally excluded:
// disc art patterns come from user config and never include escaped
// metachars in practice, and treating '\' as a metachar would misalign
// the literal-prefix extraction in extractDiscNumber.
const globMetaChars = "*?["

// extractDiscNumber parses the disc number from a filename matched by a
// filepath.Match-style glob pattern.
//
// Both pattern and filename must already be lowercased by the caller, which
// is also expected to have verified that filepath.Match(pattern, filename)
// is true before calling this function.
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

// fromExternalFile returns a sourceFunc that matches image files against a glob
// pattern. A numbered filename whose number equals the target disc wins over
// any unnumbered candidate; callers must pass a lowercase pattern.
func (d *discArtworkReader) fromExternalFile(ctx context.Context, pattern string) sourceFunc {
	isLiteral := !strings.ContainsAny(pattern, globMetaChars)
	return func() (io.ReadCloser, string, error) {
		var fallbacks []string
		for _, file := range d.imgFiles {
			name := strings.ToLower(path.Base(file))
			match, err := filepath.Match(pattern, name)
			if err != nil {
				log.Warn(ctx, "Error matching disc art file to pattern", "pattern", pattern, "file", file)
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
						log.Warn(ctx, "Could not open disc art file", "file", file, err)
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
				log.Warn(ctx, "Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: pattern '%s' not matched by files", d.discNumber, pattern)
	}
}
