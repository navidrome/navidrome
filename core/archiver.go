package core

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type Archiver interface {
	ZipAlbum(ctx context.Context, id string, w io.Writer) error
	ZipArtist(ctx context.Context, id string, w io.Writer) error
}

func NewArchiver(ds model.DataStore) Archiver {
	return &archiver{ds: ds}
}

type archiver struct {
	ds model.DataStore
}

func (a *archiver) ZipAlbum(ctx context.Context, id string, out io.Writer) error {
	mfs, err := a.ds.MediaFile(ctx).FindByAlbum(id)
	if err != nil {
		log.Error(ctx, "Error loading mediafiles from album", "id", id, err)
		return err
	}
	return a.zipTracks(ctx, id, out, mfs)
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
	return a.zipTracks(ctx, id, out, mfs)
}

func (a *archiver) zipTracks(ctx context.Context, id string, out io.Writer, mfs model.MediaFiles) error {
	z := zip.NewWriter(out)
	for _, mf := range mfs {
		_ = a.addFileToZip(ctx, z, mf)
	}
	err := z.Close()
	if err != nil {
		log.Error(ctx, "Error closing zip file", "id", id, err)
	}
	return err
}

func (a *archiver) addFileToZip(ctx context.Context, z *zip.Writer, mf model.MediaFile) error {
	_, file := filepath.Split(mf.Path)
	w, err := z.CreateHeader(&zip.FileHeader{
		Name:     fmt.Sprintf("%s/%s", mf.Album, file),
		Modified: mf.UpdatedAt,
		Method:   zip.Store,
	})
	if err != nil {
		log.Error(ctx, "Error creating zip entry", "file", mf.Path, err)
		return err
	}
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
