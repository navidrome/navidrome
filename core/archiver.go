package core

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Archiver interface {
	ZipAlbum(ctx context.Context, id string, w io.Writer) error
	ZipArtist(ctx context.Context, id string, w io.Writer) error
	ZipPlaylist(ctx context.Context, id string, w io.Writer) error
}

func NewArchiver(ds model.DataStore) Archiver {
	return &archiver{ds: ds}
}

type archiver struct {
	ds   model.DataStore
	fsys fs.FS
}

type createHeader func(idx int, mf model.MediaFile) *zip.FileHeader

func (a *archiver) ZipAlbum(ctx context.Context, id string, out io.Writer) error {
	mfs, err := a.ds.MediaFile(ctx).FindByAlbum(id)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from album", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, out, mfs, a.createHeader)
}

func (a *archiver) ZipArtist(ctx context.Context, id string, out io.Writer) error {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Sort:    "album",
		Filters: squirrel.Eq{"album_artist_id": id},
	})
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from artist", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, out, mfs, a.createHeader)
}

func (a *archiver) ZipPlaylist(ctx context.Context, id string, out io.Writer) error {
	pls, err := a.ds.Playlist(ctx).Get(id)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from playlist", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, out, pls.Tracks, a.createPlaylistHeader)
}

func (a *archiver) zipTracks(ctx context.Context, id string, out io.Writer, mfs model.MediaFiles, ch createHeader) error {
	z := zip.NewWriter(out)
	for idx, mf := range mfs {
		_ = a.addFileToZip(ctx, z, mf, ch(idx, mf))
	}
	err := z.Close()
	if err != nil {
		log.Error(ctx, "Error closing zip file", "id", id, err)
	}
	return err
}

func (a *archiver) createHeader(idx int, mf model.MediaFile) *zip.FileHeader {
	_, file := filepath.Split(mf.Path)
	return &zip.FileHeader{
		Name:     fmt.Sprintf("%s/%s", mf.Album, file),
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	}
}

func (a *archiver) createPlaylistHeader(idx int, mf model.MediaFile) *zip.FileHeader {
	_, file := filepath.Split(mf.Path)
	return &zip.FileHeader{
		Name:     fmt.Sprintf("%d - %s-%s", idx, mf.AlbumArtist, file),
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	}
}

func (a *archiver) addFileToZip(ctx context.Context, z *zip.Writer, mf model.MediaFile, zh *zip.FileHeader) error {
	w, err := z.CreateHeader(zh)
	if err != nil {
		log.Error(ctx, "Error creating zip entry", "file", mf.Path, err)
		return err
	}
	f, err := a.fsys.Open(mf.Path)
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
