package core

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type Archiver interface {
	ZipAlbum(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipArtist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipShare(ctx context.Context, id string, w io.Writer) error
	ZipPlaylist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
}

func NewArchiver(ms MediaStreamer, ds model.DataStore, shares Share) Archiver {
	return &archiver{ds: ds, ms: ms, shares: shares}
}

type archiver struct {
	ds     model.DataStore
	ms     MediaStreamer
	shares Share
}

func (a *archiver) ZipAlbum(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	return a.zipAlbums(ctx, id, format, bitrate, out, squirrel.Eq{"album_id": id})
}

func (a *archiver) ZipArtist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	return a.zipAlbums(ctx, id, format, bitrate, out, squirrel.Eq{"album_artist_id": id})
}

func (a *archiver) zipAlbums(ctx context.Context, id string, format string, bitrate int, out io.Writer, filters squirrel.Sqlizer) error {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: filters, Sort: "album"})
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from artist", "id", id, err)
		return err
	}

	z := createZipWriter(out, format, bitrate)
	albums := slice.Group(mfs, func(mf model.MediaFile) string {
		return mf.AlbumID
	})
	for _, album := range albums {
		discs := slice.Group(album, func(mf model.MediaFile) int { return mf.DiscNumber })
		isMultiDisc := len(discs) > 1
		log.Debug(ctx, "Zipping album", "name", album[0].Album, "artist", album[0].AlbumArtist,
			"format", format, "bitrate", bitrate, "isMultiDisc", isMultiDisc, "numTracks", len(album))
		for _, mf := range album {
			file := a.albumFilename(mf, format, isMultiDisc)
			_ = a.addFileToZip(ctx, z, mf, format, bitrate, file)
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
	return fmt.Sprintf("%s/%s", sanitizeName(mf.Album), file)
}

func (a *archiver) ZipShare(ctx context.Context, id string, out io.Writer) error {
	s, err := a.shares.Load(ctx, id)
	if err != nil {
		return err
	}
	if !s.Downloadable {
		return model.ErrNotAuthorized
	}
	log.Debug(ctx, "Zipping share", "name", s.ID, "format", s.Format, "bitrate", s.MaxBitRate, "numTracks", len(s.Tracks))
	return a.zipMediaFiles(ctx, id, s.Format, s.MaxBitRate, out, s.Tracks)
}

func (a *archiver) ZipPlaylist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	pls, err := a.ds.Playlist(ctx).GetWithTracks(id, true, false)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from playlist", "id", id, err)
		return err
	}
	mfs := pls.MediaFiles()
	log.Debug(ctx, "Zipping playlist", "name", pls.Name, "format", format, "bitrate", bitrate, "numTracks", len(mfs))
	return a.zipMediaFiles(ctx, id, format, bitrate, out, mfs)
}

func (a *archiver) zipMediaFiles(ctx context.Context, id string, format string, bitrate int, out io.Writer, mfs model.MediaFiles) error {
	z := createZipWriter(out, format, bitrate)
	for idx, mf := range mfs {
		file := a.playlistFilename(mf, format, idx)
		_ = a.addFileToZip(ctx, z, mf, format, bitrate, file)
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
	return fmt.Sprintf("%02d - %s - %s.%s", idx+1, sanitizeName(mf.Artist), sanitizeName(mf.Title), ext)
}

func sanitizeName(target string) string {
	return strings.ReplaceAll(target, "/", "_")
}

func (a *archiver) addFileToZip(ctx context.Context, z *zip.Writer, mf model.MediaFile, format string, bitrate int, filename string) error {
	path := mf.AbsolutePath()
	w, err := z.CreateHeader(&zip.FileHeader{
		Name:     filename,
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	})
	if err != nil {
		log.Error(ctx, "Error creating zip entry", "file", path, err)
		return err
	}

	var r io.ReadCloser
	if format != "raw" && format != "" {
		r, err = a.ms.DoStream(ctx, &mf, format, bitrate, 0)
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

	_, err = io.Copy(w, r)
	if err != nil {
		log.Error(ctx, "Error zipping file", "file", path, err)
		return err
	}

	return nil
}
