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
	Zip(ctx context.Context, id string, w io.Writer) error
}

func NewArchiver(ds model.DataStore) Archiver {
	return &archiver{ds: ds}
}

type archiver struct {
	ds model.DataStore
}

func (a *archiver) Zip(ctx context.Context, id string, out io.Writer) error {
	mfs, err := a.loadTracks(ctx, id)
	if err != nil {
		log.Error(ctx, "Error loading media", "id", id, err)
		return err
	}
	z := zip.NewWriter(out)
	for _, mf := range mfs {
		_ = a.addFileToZip(ctx, z, mf)
	}
	err = z.Close()
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

func (a *archiver) loadTracks(ctx context.Context, id string) (model.MediaFiles, error) {
	exist, err := a.ds.Album(ctx).Exists(id)
	if err != nil {
		return nil, err
	}
	if exist {
		return a.ds.MediaFile(ctx).FindByAlbum(id)
	}
	exist, err = a.ds.Artist(ctx).Exists(id)
	if err != nil {
		return nil, err
	}
	if exist {
		return a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Sort:    "album",
			Filters: squirrel.Eq{"album_artist_id": id},
		})
	}
	mf, err := a.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}
	return model.MediaFiles{*mf}, nil
}
