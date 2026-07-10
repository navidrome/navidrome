package musicfilemanager

import (
	"context"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
)

type navidromeRepo struct {
	ds      model.DataStore
	library core.Library
	scanner model.Scanner
}

func NewRepository(ds model.DataStore, library core.Library, scanner model.Scanner) SongRepository {
	return &navidromeRepo{
		ds:      ds,
		library: library,
		scanner: scanner,
	}
}

func (r *navidromeRepo) AddSong(ctx context.Context, song *model.MediaFile) error {
	_, err := r.scanner.ScanAll(ctx, false)
	return err
}

func (r *navidromeRepo) GetSongPath(ctx context.Context, songID string) (string, error) {
	mf, err := r.ds.MediaFile(ctx).Get(songID)
	if err != nil {
		return "", err
	}
	return mf.AbsolutePath(), nil
}

func (r *navidromeRepo) RefreshSong(ctx context.Context, songID string) error {
	_, err := r.scanner.ScanAll(ctx, false)
	return err
}

func (r *navidromeRepo) DeleteSong(ctx context.Context, songID string) error {
	_, err := r.scanner.ScanAll(ctx, false)
	return err
}
