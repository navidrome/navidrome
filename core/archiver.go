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
)

type Archiver interface {
	ZipAlbum(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipArtist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
	ZipPlaylist(ctx context.Context, id string, format string, bitrate int, w io.Writer) error
}

func NewArchiver(ms MediaStreamer, ds model.DataStore) Archiver {
	return &archiver{ds: ds, ms: ms}
}

type archiver struct {
	ds model.DataStore
	ms MediaStreamer
}

type createHeader func(idx int, mf model.MediaFile, format string) *zip.FileHeader

func (a *archiver) ZipAlbum(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"album_id": id},
		Sort:    "album",
	})
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from album", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, format, bitrate, out, mfs, a.createHeader)
}

func (a *archiver) ZipArtist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Sort:    "album",
		Filters: squirrel.Eq{"album_artist_id": id},
	})
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from artist", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, format, bitrate, out, mfs, a.createHeader)
}

func (a *archiver) ZipPlaylist(ctx context.Context, id string, format string, bitrate int, out io.Writer) error {
	pls, err := a.ds.Playlist(ctx).GetWithTracks(id, true)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from playlist", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, format, bitrate, out, pls.MediaFiles(), a.createPlaylistHeader)
}

func (a *archiver) zipTracks(ctx context.Context, id string, format string, bitrate int, out io.Writer, mfs model.MediaFiles, ch createHeader) error {
	z := zip.NewWriter(out)

	for idx, mf := range mfs {
		_ = a.addFileToZip(ctx, z, mf, format, bitrate, ch(idx, mf, format))
	}

	err := z.Close()
	if err != nil {
		log.Error(ctx, "Error closing zip file", "id", id, err)
	}
	return err
}

func (a *archiver) createHeader(idx int, mf model.MediaFile, format string) *zip.FileHeader {
	_, file := filepath.Split(mf.Path)

	if format != "raw" {
		file = strings.Replace(file, "."+mf.Suffix, "."+format, 1)
	}

	return &zip.FileHeader{
		Name:     fmt.Sprintf("%s/%s", mf.Album, file),
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	}
}

func (a *archiver) createPlaylistHeader(idx int, mf model.MediaFile, format string) *zip.FileHeader {
	_, file := filepath.Split(mf.Path)

	if format != "raw" {
		file = strings.Replace(file, "."+mf.Suffix, "."+format, 1)
	}

	return &zip.FileHeader{
		Name:     fmt.Sprintf("%d - %s - %s", idx+1, mf.AlbumArtist, file),
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	}
}

func (a *archiver) addFileToZip(ctx context.Context, z *zip.Writer, mf model.MediaFile, format string, bitrate int, zh *zip.FileHeader) error {
	w, err := z.CreateHeader(zh)
	if err != nil {
		log.Error(ctx, "Error creating zip entry", "file", mf.Path, err)
		return err
	}

	if format != "raw" {
		stream, err := a.ms.DoStream(ctx, &mf, format, bitrate)

		if err != nil {
			return err
		}

		defer func() {
			if err := stream.Close(); err != nil && log.CurrentLevel() >= log.LevelDebug {
				log.Error("Error closing stream", "id", mf.ID, "file", stream.Name(), err)
			}
		}()

		_, err = io.Copy(w, stream)

		if err != nil {
			log.Error(ctx, "Error zipping file", "file", mf.Path, err)
			return err
		}

		return nil
	} else {
		f, err := os.Open(mf.Path)
		defer func() { _ = f.Close() }()
		if err != nil {
			log.Error(ctx, "Error opening file for zipping", "file", mf.Path, err)
			return err
		}
		_, err = io.Copy(w, f)
		if err != nil {
			log.Error(ctx, "Error zipping file", "file", mf.Path, err)
			return err
		}
		return nil
	}
}
