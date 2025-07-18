package persistence

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/run"
	"github.com/pocketbase/dbx"
)

type libraryRepository struct {
	sqlRepository
}

var (
	libCache = map[int]string{}
	libLock  sync.RWMutex
)

func NewLibraryRepository(ctx context.Context, db dbx.Builder) model.LibraryRepository {
	r := &libraryRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Library{}, nil)
	return r
}

func (r *libraryRepository) Get(id int) (*model.Library, error) {
	sq := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Library
	err := r.queryOne(sq, &res)
	return &res, err
}

func (r *libraryRepository) GetPath(id int) (string, error) {
	l := func() string {
		libLock.RLock()
		defer libLock.RUnlock()
		if l, ok := libCache[id]; ok {
			return l
		}
		return ""
	}()
	if l != "" {
		return l, nil
	}

	libLock.Lock()
	defer libLock.Unlock()
	libs, err := r.GetAll()
	if err != nil {
		log.Error(r.ctx, "Error loading libraries from DB", err)
		return "", err
	}
	for _, l := range libs {
		libCache[l.ID] = l.Path
	}
	if l, ok := libCache[id]; ok {
		return l, nil
	} else {
		return "", model.ErrNotFound
	}
}

func (r *libraryRepository) Put(l *model.Library) error {
	if l.ID == model.DefaultLibraryID {
		currentLib, err := r.Get(1)
		// if we are creating it, it's ok.
		if err == nil { // it exists, so we are updating it
			if currentLib.Path != l.Path {
				return fmt.Errorf("%w: path for library with ID 1 cannot be changed", model.ErrValidation)
			}
		}
	}

	var err error
	l.UpdatedAt = time.Now()
	if l.ID == 0 {
		// Insert with autoassigned ID
		l.CreatedAt = time.Now()
		err = r.db.Model(l).Insert()
	} else {
		// Try to update first
		cols := map[string]any{
			"name":              l.Name,
			"path":              l.Path,
			"remote_path":       l.RemotePath,
			"default_new_users": l.DefaultNewUsers,
			"updated_at":        l.UpdatedAt,
		}
		sq := Update(r.tableName).SetMap(cols).Where(Eq{"id": l.ID})
		rowsAffected, updateErr := r.executeSQL(sq)
		if updateErr != nil {
			return updateErr
		}

		// If no rows were affected, the record doesn't exist, so insert it
		if rowsAffected == 0 {
			l.CreatedAt = time.Now()
			l.UpdatedAt = time.Now()
			err = r.db.Model(l).Insert()
		}
	}
	if err != nil {
		return err
	}

	// Auto-assign all libraries to all admin users
	sql := Expr(`
INSERT INTO user_library (user_id, library_id)
SELECT u.id, l.id
FROM user u
CROSS JOIN library l
WHERE u.is_admin = true
ON CONFLICT (user_id, library_id) DO NOTHING;`,
	)
	if _, err = r.executeSQL(sql); err != nil {
		return fmt.Errorf("failed to assign library to admin users: %w", err)
	}

	libLock.Lock()
	defer libLock.Unlock()
	libCache[l.ID] = l.Path
	return nil
}

// TODO Remove this method when we have a proper UI to add libraries
// This is a temporary method to store the music folder path from the config in the DB
func (r *libraryRepository) StoreMusicFolder() error {
	sq := Update(r.tableName).Set("path", conf.Server.MusicFolder).
		Set("updated_at", time.Now()).
		Where(Eq{"id": model.DefaultLibraryID})
	_, err := r.executeSQL(sq)
	if err != nil {
		libLock.Lock()
		defer libLock.Unlock()
		libCache[model.DefaultLibraryID] = conf.Server.MusicFolder
	}
	return err
}

func (r *libraryRepository) AddArtist(id int, artistID string) error {
	sq := Insert("library_artist").Columns("library_id", "artist_id").Values(id, artistID).
		Suffix(`on conflict(library_id, artist_id) do nothing`)
	_, err := r.executeSQL(sq)
	if err != nil {
		return err
	}
	return nil
}

func (r *libraryRepository) ScanBegin(id int, fullScan bool) error {
	sq := Update(r.tableName).
		Set("last_scan_started_at", time.Now()).
		Set("full_scan_in_progress", fullScan).
		Where(Eq{"id": id})
	_, err := r.executeSQL(sq)
	return err
}

func (r *libraryRepository) ScanEnd(id int) error {
	sq := Update(r.tableName).
		Set("last_scan_at", time.Now()).
		Set("full_scan_in_progress", false).
		Set("last_scan_started_at", time.Time{}).
		Where(Eq{"id": id})
	_, err := r.executeSQL(sq)
	if err != nil {
		return err
	}
	// https://www.sqlite.org/pragma.html#pragma_optimize
	_, err = r.executeSQL(Expr("PRAGMA optimize=0x10012;"))
	return err
}

func (r *libraryRepository) ScanInProgress() (bool, error) {
	query := r.newSelect().Where(NotEq{"last_scan_started_at": time.Time{}})
	count, err := r.count(query)
	return count > 0, err
}

func (r *libraryRepository) RefreshStats(id int) error {
	var songsRes, albumsRes, artistsRes, foldersRes, filesRes, missingRes struct{ Count int64 }
	var sizeRes struct{ Sum int64 }
	var durationRes struct{ Sum float64 }

	err := run.Parallel(
		func() error {
			return r.queryOne(Select("count(*) as count").From("media_file").Where(Eq{"library_id": id, "missing": false}), &songsRes)
		},
		func() error {
			return r.queryOne(Select("count(*) as count").From("album").Where(Eq{"library_id": id, "missing": false}), &albumsRes)
		},
		func() error {
			return r.queryOne(Select("count(*) as count").From("library_artist la").
				Join("artist a on la.artist_id = a.id").
				Where(Eq{"la.library_id": id, "a.missing": false}), &artistsRes)
		},
		func() error {
			return r.queryOne(Select("count(*) as count").From("folder").
				Where(And{
					Eq{"library_id": id, "missing": false},
					Gt{"num_audio_files": 0},
				}), &foldersRes)
		},
		func() error {
			return r.queryOne(Select("ifnull(sum(num_audio_files + num_playlists + json_array_length(image_files)),0) as count").
				From("folder").Where(Eq{"library_id": id, "missing": false}), &filesRes)
		},
		func() error {
			return r.queryOne(Select("count(*) as count").From("media_file").Where(Eq{"library_id": id, "missing": true}), &missingRes)
		},
		func() error {
			return r.queryOne(Select("ifnull(sum(size),0) as sum").From("album").Where(Eq{"library_id": id, "missing": false}), &sizeRes)
		},
		func() error {
			return r.queryOne(Select("ifnull(sum(duration),0) as sum").From("album").Where(Eq{"library_id": id, "missing": false}), &durationRes)
		},
	)()
	if err != nil {
		return err
	}

	sq := Update(r.tableName).
		Set("total_songs", songsRes.Count).
		Set("total_albums", albumsRes.Count).
		Set("total_artists", artistsRes.Count).
		Set("total_folders", foldersRes.Count).
		Set("total_files", filesRes.Count).
		Set("total_missing_files", missingRes.Count).
		Set("total_size", sizeRes.Sum).
		Set("total_duration", durationRes.Sum).
		Set("updated_at", time.Now()).
		Where(Eq{"id": id})
	_, err = r.executeSQL(sq)
	return err
}

func (r *libraryRepository) Delete(id int) error {
	if !loggedUser(r.ctx).IsAdmin {
		return model.ErrNotAuthorized
	}
	if id == 1 {
		return fmt.Errorf("%w: library with ID 1 cannot be deleted", model.ErrValidation)
	}

	err := r.delete(Eq{"id": id})
	if err != nil {
		return err
	}

	// Clear cache entry for this library only if DB operation was successful
	libLock.Lock()
	defer libLock.Unlock()
	delete(libCache, id)

	return nil
}

func (r *libraryRepository) GetAll(ops ...model.QueryOptions) (model.Libraries, error) {
	sq := r.newSelect(ops...).Columns("*")
	res := model.Libraries{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *libraryRepository) CountAll(ops ...model.QueryOptions) (int64, error) {
	sq := r.newSelect(ops...)
	return r.count(sq)
}

// User-library association methods

func (r *libraryRepository) GetUsersWithLibraryAccess(libraryID int) (model.Users, error) {
	sel := Select("u.*").
		From("user u").
		Join("user_library ul ON u.id = ul.user_id").
		Where(Eq{"ul.library_id": libraryID}).
		OrderBy("u.name")

	var res model.Users
	err := r.queryAll(sel, &res)
	return res, err
}

// REST interface methods

func (r *libraryRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *libraryRepository) Read(id string) (interface{}, error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Trace(r.ctx, "invalid library id: %s", id, err)
		return nil, rest.ErrNotFound
	}
	return r.Get(idInt)
}

func (r *libraryRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *libraryRepository) EntityName() string {
	return "library"
}

func (r *libraryRepository) NewInstance() interface{} {
	return &model.Library{}
}

func (r *libraryRepository) Save(entity interface{}) (string, error) {
	lib := entity.(*model.Library)
	lib.ID = 0 // Reset ID to ensure we create a new library
	err := r.Put(lib)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(lib.ID), nil
}

func (r *libraryRepository) Update(id string, entity interface{}, cols ...string) error {
	lib := entity.(*model.Library)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid library ID: %s", id)
	}

	lib.ID = idInt
	return r.Put(lib)
}

var _ model.LibraryRepository = (*libraryRepository)(nil)
var _ rest.Repository = (*libraryRepository)(nil)
