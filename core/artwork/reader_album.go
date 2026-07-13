package artwork

import (
	"cmp"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/natural"
)

type albumArtworkReader struct {
	cacheKey
	a          *artwork
	provider   external.Provider
	album      model.Album
	updatedAt  *time.Time
	imgFiles   []string // library-relative, forward-slash, no leading slash
	lib        libraryView
	imageIndex int // -1 = use cover-art priority; >=0 = serve the Nth recognized image
}

func newAlbumArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, provider external.Provider) (*albumArtworkReader, error) {
	albumID, imageIndex, err := model.ParseAlbumArtworkID(artID.ID)
	if err != nil {
		return nil, err
	}
	al, err := artwork.ds.Album(ctx).Get(albumID)
	if err != nil {
		return nil, err
	}
	_, imgFiles, imagesUpdateAt, err := loadAlbumFoldersPaths(ctx, artwork.ds, *al)
	if err != nil {
		return nil, err
	}
	if imageIndex >= 0 && imageIndex >= len(recognizedAlbumImages(imgFiles)) {
		return nil, model.ErrNotFound
	}
	lib, err := loadLibraryView(ctx, artwork.ds, al.LibraryID)
	if err != nil {
		return nil, err
	}
	a := &albumArtworkReader{
		a:          artwork,
		provider:   provider,
		album:      *al,
		updatedAt:  imagesUpdateAt,
		imgFiles:   imgFiles,
		lib:        lib,
		imageIndex: imageIndex,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = utils.TimeNewest(al.UpdatedAt, al.ImportedAt)
	if imagesUpdateAt != nil {
		a.cacheKey.lastUpdate = utils.TimeNewest(a.cacheKey.lastUpdate, *imagesUpdateAt)
	}
	return a, nil
}

func (a *albumArtworkReader) Key() string {
	hashInput := conf.Server.CoverArtPriority
	if conf.Server.EnableExternalServices {
		hashInput = conf.Server.Agents + hashInput
	}
	hash := md5.Sum([]byte(hashInput))
	return fmt.Sprintf(
		"%s.%x.%t",
		a.cacheKey.Key(),
		hash,
		conf.Server.EnableExternalServices,
	)
}
func (a *albumArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *albumArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	if a.imageIndex >= 0 {
		return selectImageReader(ctx, a.artID, a.fromImageIndex(ctx, a.imageIndex))
	}
	var ff = a.fromCoverArtPriority(ctx, a.a.ffmpeg, conf.Server.CoverArtPriority)
	return selectImageReader(ctx, a.artID, ff...)
}

// fromImageIndex serves the Nth recognized external image (bypassing priority).
func (a *albumArtworkReader) fromImageIndex(ctx context.Context, index int) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		images := recognizedAlbumImages(a.imgFiles)
		if index < 0 || index >= len(images) {
			return nil, "", fmt.Errorf("album image index %d out of range (%d images): %w", index, len(images), model.ErrNotFound)
		}
		file := images[index].Path
		f, err := a.lib.FS.Open(file)
		if err != nil {
			return nil, "", err
		}
		return f, file, nil
	}
}

func (a *albumArtworkReader) fromCoverArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []sourceFunc {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			embedRel := a.album.EmbedArtPath
			ff = append(ff,
				fromTag(ctx, a.lib.FS, embedRel),
				fromFFmpegTag(ctx, ffmpeg, a.lib.Abs(embedRel)),
			)
		case pattern == "external":
			ff = append(ff, fromAlbumExternalSource(ctx, a.album, a.provider))
		case len(a.imgFiles) > 0:
			ff = append(ff, fromExternalFile(ctx, a.lib.FS, a.imgFiles, pattern))
		}
	}
	return ff
}

func loadAlbumFoldersPaths(ctx context.Context, ds model.DataStore, albums ...model.Album) ([]string, []string, *time.Time, error) {
	var folderIDs []string
	for _, album := range albums {
		folderIDs = append(folderIDs, album.FolderIDs...)
	}
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderIDs, "missing": false}})
	if err != nil {
		return nil, nil, nil, err
	}

	parent, err := albumRootParent(ctx, ds, folders, folderIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	if parent != nil {
		folders = append(folders, *parent)
	}

	var paths []string
	var imgFiles []string
	var updatedAt time.Time
	for _, f := range folders {
		paths = append(paths, f.AbsolutePath())
		if f.ImagesUpdatedAt.After(updatedAt) {
			updatedAt = f.ImagesUpdatedAt
		}
		rel := strings.TrimPrefix(path.Join(f.Path, f.Name), "/")
		for _, img := range f.ImageFiles {
			imgFiles = append(imgFiles, path.Join(rel, img))
		}
	}

	// Sort image files to ensure consistent selection of cover art
	// This prioritizes files without numeric suffixes (e.g., cover.jpg over cover.1.jpg)
	// by comparing base filenames without extensions
	slices.SortFunc(imgFiles, compareImageFiles)

	return paths, imgFiles, &updatedAt, nil
}

// albumRootParent returns the common parent of the album's folders when it
// qualifies as the album's root folder (e.g. "Artist/Album" above disc
// subfolders), or nil when there is no such parent. This finds cover art in
// the album root folder when tracks live in disc subfolders, like
// "Artist/Album/cover.jpg" with tracks in "Artist/Album/CD1/" and
// "Artist/Album/CD2/". The parent must look like an album root, not an
// artist-level folder — it qualifies only when it holds no audio belonging to
// other albums — so artist images are never served as album art.
func albumRootParent(ctx context.Context, ds model.DataStore, folders []model.Folder, folderIDs []string) (*model.Folder, error) {
	folderIDSet := make(map[string]bool, len(folderIDs))
	for _, id := range folderIDs {
		folderIDSet[id] = true
	}
	commonParentID := commonParentFolder(folders, folderIDSet)
	if commonParentID == "" {
		return nil, nil
	}
	// Single-folder albums only use the parent when the folder has no images
	// of its own (indicating a disc subfolder needing parent artwork).
	if len(folders) < 2 && anyFolderHasImages(folders) {
		return nil, nil
	}
	parent, err := ds.Folder(ctx).Get(commonParentID)
	if errors.Is(err, model.ErrNotFound) {
		log.Warn(ctx, "Parent folder not found for album cover art lookup", "parentID", commonParentID)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parent.ParentID == "" {
		// The library root can never be an album root
		return nil, nil
	}
	hasOtherAudio, err := ds.Folder(ctx).HasAudioOutsideFolders(*parent, folderIDs)
	if err != nil {
		return nil, err
	}
	if hasOtherAudio {
		return nil, nil
	}
	return parent, nil
}

func anyFolderHasImages(folders []model.Folder) bool {
	for _, f := range folders {
		if len(f.ImageFiles) > 0 {
			return true
		}
	}
	return false
}

// commonParentFolder returns the shared parent folder ID when all folders have the
// same parent and that parent is not already in folderIDSet. Returns "" otherwise.
func commonParentFolder(folders []model.Folder, folderIDSet map[string]bool) string {
	if len(folders) == 0 {
		return ""
	}
	parentID := folders[0].ParentID
	if parentID == "" || folderIDSet[parentID] {
		return ""
	}
	for _, f := range folders[1:] {
		if f.ParentID != parentID {
			return ""
		}
	}
	return parentID
}

// albumImage is a recognized external image file with its inferred type.
type albumImage struct {
	Path string // library-relative, forward-slash
	Name string // base filename
	Type string // official MusicBrainz CAA type (Front, Back, Booklet, Medium, ...)
}

// albumImageTypes maps filename stems to the official MusicBrainz CAA type
// (https://musicbrainz.org/doc/Cover_Art/Types). Slice order is the gallery
// display order; non-Front types match before Front so "back cover" → Back.
var albumImageTypes = []struct {
	Type  string
	stems []string
}{
	{"Front", []string{"front", "cover", "folder", "album", "albumart", "art"}},
	{"Back", []string{"back"}},
	{"Booklet", []string{"booklet", "leaflet", "inlay", "inside"}},
	{"Medium", []string{"medium", "media", "disc", "discart", "disque", "cd", "cdart"}},
	{"Tray", []string{"tray"}},
	{"Obi", []string{"obi"}},
	{"Spine", []string{"spine"}},
	{"Track", []string{"track"}},
	{"Liner", []string{"liner"}},
	{"Sticker", []string{"sticker"}},
	{"Poster", []string{"poster"}},
	{"Matrix/Runout", []string{"matrix", "runout"}},
	{"Top", []string{"top"}},
	{"Bottom", []string{"bottom"}},
	{"Panel", []string{"panel", "gatefold"}},
	{"Watermark", []string{"watermark"}},
	{"Raw/Unedited", []string{"raw", "unedited"}},
	{"Other", []string{"other"}},
}

// imageTypeRank maps each type to its display order, derived from albumImageTypes.
var imageTypeRank = func() map[string]int {
	m := make(map[string]int, len(albumImageTypes))
	for i, t := range albumImageTypes {
		m[t.Type] = i
	}
	return m
}()

// imageTypeFromName infers the CAA type from a filename (numeric suffixes
// tolerated); returns "" for unrecognized names.
func imageTypeFromName(name string) string {
	stem := strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
	fields := strings.FieldsFunc(stem, func(r rune) bool {
		return r == ' ' || r == '.' || r == '-' || r == '_'
	})
	for i, f := range fields {
		fields[i] = strings.TrimRight(f, "0123456789")
	}
	matches := func(stems []string) bool {
		for _, f := range fields {
			if slices.Contains(stems, f) {
				return true
			}
		}
		return false
	}
	// A specific (non-Front) type wins over a generic front token.
	for _, t := range albumImageTypes {
		if t.Type == "Front" {
			continue
		}
		if matches(t.stems) {
			return t.Type
		}
	}
	if matches(albumImageTypes[0].stems) { // Front
		return "Front"
	}
	return ""
}

// resolveCoverFile returns the external file the primary cover (al-<id>) resolves
// to, or "" if it comes from a non-file source. Lets AlbumImages skip that file.
func resolveCoverFile(imgFiles []string, priority string) string {
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || pattern == "embedded" || pattern == "external" {
			continue
		}
		for _, f := range imgFiles {
			if ok, _ := path.Match(pattern, strings.ToLower(path.Base(f))); ok {
				return f
			}
		}
	}
	return ""
}

// recognizedAlbumImages returns the album's recognized-type images, ordered by
// type then filename. Single source of truth for the indexed fetch and listing.
func recognizedAlbumImages(imgFiles []string) []albumImage {
	var images []albumImage
	for _, f := range imgFiles {
		name := path.Base(f)
		if t := imageTypeFromName(name); t != "" {
			images = append(images, albumImage{Path: f, Name: name, Type: t})
		}
	}
	slices.SortStableFunc(images, func(a, b albumImage) int {
		return cmp.Or(
			cmp.Compare(imageTypeRank[a.Type], imageTypeRank[b.Type]),
			compareImageFiles(a.Path, b.Path),
		)
	})
	return images
}

// compareImageFiles sorts image paths by: base filename (natural order),
// then path depth (shallower first), then full path (stable tiebreaker).
func compareImageFiles(a, b string) int {
	// Case-insensitive comparison
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Extract base filenames without extensions
	baseA := strings.TrimSuffix(path.Base(a), path.Ext(a))
	baseB := strings.TrimSuffix(path.Base(b), path.Ext(b))

	// Compare base names first, then prefer shallower paths, then full path as tiebreaker
	return cmp.Or(
		natural.Compare(baseA, baseB),
		cmp.Compare(strings.Count(a, "/"), strings.Count(b, "/")),
		natural.Compare(a, b),
	)
}
