package core

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

// archiveCoverArtSize is the size of the folder image added to each archive folder.
const archiveCoverArtSize = 500

type Archiver interface {
	ZipAlbum(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipArtist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipShare(ctx context.Context, s *model.Share, w io.Writer) error
	ZipPlaylist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
}

// CoverArtReader is a local interface satisfied by artwork.CoverArtReader.
// Defined here to avoid an import cycle between core/artwork and core.
type CoverArtReader interface {
	// Read returns the image getCoverArt serves for artID, or model.ErrNotFound when the
	// item has no artwork (placeholders are never returned).
	Read(ctx context.Context, artID model.ArtworkID, size int, square bool) (io.ReadCloser, error)
}

func NewArchiver(ms stream.MediaStreamer, ds model.DataStore, shares Share, coverArt CoverArtReader) Archiver {
	return &archiver{ds: ds, ms: ms, shares: shares, coverArt: coverArt}
}

type archiver struct {
	ds       model.DataStore
	ms       stream.MediaStreamer
	shares   Share
	coverArt CoverArtReader
}

func (a *archiver) ZipAlbum(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	return a.zipAlbums(ctx, id, format, bitrate, out, squirrel.Eq{"album_id": id}, model.ArtworkID{})
}

func (a *archiver) ZipArtist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	// Match by album-artist participation, not the deprecated album_artist_id
	// column (first album artist only), so co-album-artists are included too.
	filter := squirrel.And{
		persistence.ParticipantIDFilter("media_file", id, model.RoleAlbumArtist),
		squirrel.Eq{"missing": false},
	}
	return a.zipAlbums(ctx, id, format, bitrate, out, filter, model.Artist{ID: id}.CoverArtID())
}

// zipAlbums puts each album in its own folder, with the album cover in it. rootArt, when set,
// is added to the archive root.
func (a *archiver) zipAlbums(ctx context.Context, id string, format string, bitrate int, out io.Writer, filters squirrel.Sqlizer, rootArt model.ArtworkID) error {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: filters, Sort: "album"})
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from artist", "id", id, err)
		return err
	}

	z := createZipWriter(out, format, bitrate)
	a.addCoverArtToZip(ctx, z, rootArt, "")
	albums := slice.Group(mfs, func(mf model.MediaFile) string {
		return mf.AlbumID
	})
	for _, album := range albums {
		discs := slice.Group(album, func(mf model.MediaFile) int { return mf.DiscNumber })
		isMultiDisc := len(discs) > 1
		log.Debug(ctx, "Zipping album", "name", album[0].Album, "artist", album[0].AlbumArtist,
			"format", format, "bitrate", bitrate, "isMultiDisc", isMultiDisc, "numTracks", len(album))
		a.addCoverArtToZip(ctx, z, album[0].AlbumCoverArtID(), albumFolder(album[0]))
		for _, mf := range album {
			file := a.albumFilename(mf, format, isMultiDisc)
			if addErr := a.addFileToZip(ctx, z, mf, format, bitrate, file); errors.Is(addErr, stream.ErrTooManyTranscodes) {
				// Stop iterating: continuing would just rack up more
				// rejections from the limiter. Close finalises whatever
				// tracks were already written; the rejected one is not
				// present in the archive (addFileToZip aborts before
				// writing its entry header).
				_ = z.Close()
				return addErr
			}
		}
	}
	err = z.Close()
	if err != nil {
		log.Error(ctx, "Error closing zip file", "id", id, err)
	}
	return err
}

func createZipWriter(out io.Writer, format string, bitrate int) *zip.Writer {
	z := zip.NewWriter(out)
	comment := "Downloaded from Navidrome"
	if format != "raw" && format != "" {
		comment = fmt.Sprintf("%s, transcoded to %s %dbps", comment, format, bitrate)
	}
	_ = z.SetComment(comment)
	return z
}

func (a *archiver) albumFilename(mf model.MediaFile, format string, isMultiDisc bool) string {
	_, file := filepath.Split(mf.Path)
	if format != "raw" {
		file = strings.TrimSuffix(file, mf.Suffix) + format
	}
	if isMultiDisc {
		file = fmt.Sprintf("Disc %02d/%s", mf.DiscNumber, file)
	}
	return fmt.Sprintf("%s/%s", albumFolder(mf), file)
}

func albumFolder(mf model.MediaFile) string {
	return str.SanitizeFilename(mf.Album)
}

// ZipShare takes an already-loaded share: Share.Load records a visit, so
// loading it again here would count every download twice.
func (a *archiver) ZipShare(ctx context.Context, s *model.Share, out io.Writer) error {
	if !s.Downloadable {
		return model.ErrNotAuthorized
	}
	log.Debug(ctx, "Zipping share", "name", s.ID, "format", s.Format, "bitrate", s.MaxBitRate, "numTracks", len(s.Tracks))
	return a.zipMediaFiles(ctx, s.ID, s.ID, s.Format, s.MaxBitRate, out, s.Tracks, s.CoverArtID(), false)
}

func (a *archiver) ZipPlaylist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	pls, err := a.ds.Playlist(ctx).GetWithTracks(id, true, false)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from playlist", "id", id, err)
		return err
	}
	mfs := pls.MediaFiles()
	log.Debug(ctx, "Zipping playlist", "name", pls.Name, "format", format, "bitrate", bitrate, "numTracks", len(mfs))
	return a.zipMediaFiles(ctx, id, pls.Name, format, bitrate, out, mfs, pls.CoverArtID(), true)
}

func (a *archiver) zipMediaFiles(ctx context.Context, id, name string, format string, bitrate int, out io.Writer, mfs model.MediaFiles, coverArt model.ArtworkID, addM3U bool) error {
	z := createZipWriter(out, format, bitrate)
	a.addCoverArtToZip(ctx, z, coverArt, "")

	zippedMfs := make(model.MediaFiles, len(mfs))
	for idx, mf := range mfs {
		file := a.playlistFilename(mf, format, idx)
		if addErr := a.addFileToZip(ctx, z, mf, format, bitrate, file); errors.Is(addErr, stream.ErrTooManyTranscodes) {
			// Abort the whole archive: continuing would silently emit
			// empty zip entries since the headers are already written.
			_ = z.Close()
			return addErr
		}
		mf.Path = file
		zippedMfs[idx] = mf
	}

	// Add M3U file if requested
	if addM3U && len(zippedMfs) > 0 {
		plsName := str.SanitizeFilename(name)
		w, err := z.CreateHeader(&zip.FileHeader{
			Name:     plsName + ".m3u",
			Modified: mfs[0].UpdatedAt,
			Method:   zip.Store,
		})
		if err != nil {
			log.Error(ctx, "Error creating playlist zip entry", err)
			return err
		}

		_, err = w.Write([]byte(zippedMfs.ToM3U8(plsName, false)))
		if err != nil {
			log.Error(ctx, "Error writing m3u in zip", err)
			return err
		}
	}

	err := z.Close()
	if err != nil {
		log.Error(ctx, "Error closing zip file", "id", id, err)
	}
	return err
}

func (a *archiver) playlistFilename(mf model.MediaFile, format string, idx int) string {
	ext := mf.Suffix
	if format != "" && format != "raw" {
		ext = format
	}
	return fmt.Sprintf("%02d - %s - %s.%s", idx+1, str.SanitizeFilename(mf.Artist), str.SanitizeFilename(mf.Title), ext)
}

func (a *archiver) addFileToZip(ctx context.Context, z *zip.Writer, mf model.MediaFile, format string, bitrate int, filename string) error {
	path := mf.AbsolutePath()

	// Open the source before writing the zip entry header so a rejection
	// (limiter, missing file, etc.) does not leave an empty entry in the
	// archive.
	var r io.ReadCloser
	var err error
	if format != "raw" && format != "" {
		r, err = a.ms.NewStream(ctx, &mf, stream.Request{Format: format, BitRate: bitrate})
	} else {
		r, err = os.Open(path)
	}
	if err != nil {
		log.Error(ctx, "Error opening file for zipping", "file", path, "format", format, err)
		return err
	}
	defer func() {
		if err := r.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error(ctx, "Error closing stream", "id", mf.ID, "file", path, err)
		}
	}()

	w, err := z.CreateHeader(&zip.FileHeader{
		Name:     filename,
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	})
	if err != nil {
		log.Error(ctx, "Error creating zip entry", "file", path, err)
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		log.Error(ctx, "Error zipping file", "file", path, err)
		return err
	}

	return nil
}

// addCoverArtToZip writes the item's cover as folder.<ext> in dir, the image most players and
// car stereos show for the files next to it. A missing or failing cover never fails the archive.
func (a *archiver) addCoverArtToZip(ctx context.Context, z *zip.Writer, artID model.ArtworkID, dir string) {
	if artID.ID == "" {
		return
	}
	// Read the whole image before writing the entry header, so a failure leaves no empty entry.
	data, err := a.readCoverArt(ctx, artID)
	if errors.Is(err, model.ErrNotFound) {
		log.Debug(ctx, "No cover art to add to zip", "artID", artID)
		return
	}
	if err != nil {
		log.Warn(ctx, "Error reading cover art for zipping", "artID", artID, err)
		return
	}
	ext := coverArtExtension(data)
	if ext == "" {
		log.Warn(ctx, "Unknown cover art image type, not adding it to zip", "artID", artID)
		return
	}
	w, err := z.CreateHeader(&zip.FileHeader{
		Name:     path.Join(dir, "folder."+ext),
		Modified: time.Now(),
		Method:   zip.Store,
	})
	if err != nil {
		log.Warn(ctx, "Error creating cover art zip entry", "artID", artID, err)
		return
	}
	if _, err = w.Write(data); err != nil {
		log.Warn(ctx, "Error zipping cover art", "artID", artID, err)
	}
}

func (a *archiver) readCoverArt(ctx context.Context, artID model.ArtworkID) ([]byte, error) {
	r, err := a.coverArt.Read(ctx, artID, archiveCoverArtSize, false)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// coverArtExtension names the image by its content: resizing can re-encode it (e.g. to WebP),
// so the source file's extension is not reliable.
func coverArtExtension(data []byte) string {
	switch http.DetectContentType(data) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	}
	return ""
}
