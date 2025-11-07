package persistence

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type mediaFileRepository struct {
	sqlRepository
}

type dbMediaFile struct {
	*model.MediaFile `structs:",flatten"`
	Participants     string `structs:"-" json:"-"`
	Tags             string `structs:"-" json:"-"`
	// These are necessary to map the correct names (rg_*) to the correct fields (RG*)
	// without using `db` struct tags in the model.MediaFile struct
	RgAlbumGain *float64 `structs:"-" json:"-"`
	RgAlbumPeak *float64 `structs:"-" json:"-"`
	RgTrackGain *float64 `structs:"-" json:"-"`
	RgTrackPeak *float64 `structs:"-" json:"-"`
}

func (m *dbMediaFile) PostScan() error {
	m.RGTrackGain = m.RgTrackGain
	m.RGTrackPeak = m.RgTrackPeak
	m.RGAlbumGain = m.RgAlbumGain
	m.RGAlbumPeak = m.RgAlbumPeak
	var err error
	m.MediaFile.Participants, err = unmarshalParticipants(m.Participants)
	if err != nil {
		return fmt.Errorf("parsing media_file from db: %w", err)
	}
	if m.Tags != "" {
		m.MediaFile.Tags, err = unmarshalTags(m.Tags)
		if err != nil {
			return fmt.Errorf("parsing media_file from db: %w", err)
		}
		m.Genre, m.Genres = m.MediaFile.Tags.ToGenres()
	}
	return nil
}

func (m *dbMediaFile) PostMapArgs(args map[string]any) error {
	fullText := []string{m.FullTitle(), m.Album, m.Artist, m.AlbumArtist,
		m.SortTitle, m.SortAlbumName, m.SortArtistName, m.SortAlbumArtistName, m.DiscSubtitle}
	fullText = append(fullText, m.MediaFile.Participants.AllNames()...)
	args["full_text"] = formatFullText(fullText...)
	args["tags"] = marshalTags(m.MediaFile.Tags)
	args["participants"] = marshalParticipants(m.MediaFile.Participants)
	return nil
}

type dbMediaFiles []dbMediaFile

func (m dbMediaFiles) toModels() model.MediaFiles {
	return slice.Map(m, func(mf dbMediaFile) model.MediaFile { return *mf.MediaFile })
}

func NewMediaFileRepository(ctx context.Context, db dbx.Builder) model.MediaFileRepository {
	r := &mediaFileRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "media_file"
	r.registerModel(&model.MediaFile{}, mediaFileFilter())
	r.setSortMappings(map[string]string{
		"title":          "order_title",
		"artist":         "order_artist_name, order_album_name, release_date, disc_number, track_number",
		"album_artist":   "order_album_artist_name, order_album_name, release_date, disc_number, track_number",
		"album":          "order_album_name, album_id, disc_number, track_number, order_artist_name, title",
		"random":         "random",
		"created_at":     "media_file.created_at",
		"recently_added": mediaFileRecentlyAddedSort(),
		"starred_at":     "starred, starred_at",
	})
	return r
}

var mediaFileFilter = sync.OnceValue(func() map[string]filterFunc {
	filters := map[string]filterFunc{
		"id":         idFilter("media_file"),
		"title":      fullTextFilter("media_file", "mbz_recording_id", "mbz_release_track_id"),
		"starred":    booleanFilter,
		"genre_id":   tagIDFilter,
		"missing":    booleanFilter,
		"artists_id": artistFilter,
		"library_id": libraryIdFilter,
	}
	// Add all album tags as filters
	for tag := range model.TagMappings() {
		if _, exists := filters[string(tag)]; !exists {
			filters[string(tag)] = tagIDFilter
		}
	}
	return filters
})

func mediaFileRecentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "media_file.updated_at"
	}
	return "media_file.created_at"
}

func (r *mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.withAnnotation(query, "media_file.id")
	query = r.applyLibraryFilter(query)
	return r.count(query, options...)
}

func (r *mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"media_file.id": id})
}

func (r *mediaFileRepository) Put(m *model.MediaFile) error {
	m.CreatedAt = time.Now()
	id, err := r.putByMatch(Eq{"path": m.Path, "library_id": m.LibraryID}, m.ID, &dbMediaFile{MediaFile: m})
	if err != nil {
		return err
	}
	m.ID = id
	return r.updateParticipants(m.ID, m.Participants)
}

func (r *mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id")
	sql = r.withAnnotation(sql, "media_file.id")
	sql = r.withBookmark(sql, "media_file.id")
	return r.applyLibraryFilter(sql)
}

func (r *mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	res, err := r.GetAll(model.QueryOptions{Filters: Eq{"media_file.id": id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *mediaFileRepository) GetWithParticipants(id string) (*model.MediaFile, error) {
	m, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	m.Participants, err = r.getParticipants(m)
	return m, err
}

func (r *mediaFileRepository) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...)
	var res dbMediaFiles
	err := r.queryAll(sq, &res, options...)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) GetCursor(options ...model.QueryOptions) (model.MediaFileCursor, error) {
	sq := r.selectMediaFile(options...)
	cursor, err := queryWithStableResults[dbMediaFile](r.sqlRepository, sq)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.MediaFile, error) bool) {
		for m, err := range cursor {
			if m.MediaFile == nil {
				yield(model.MediaFile{}, fmt.Errorf("unexpected nil mediafile: %v", m))
				return
			}
			if !yield(*m.MediaFile, err) || err != nil {
				return
			}
		}
	}, nil
}

func (r *mediaFileRepository) FindByPaths(paths []string) (model.MediaFiles, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"path collate nocase": paths})
	var res dbMediaFiles
	if err := r.queryAll(sel, &res); err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r *mediaFileRepository) DeleteAllMissing() (int64, error) {
	user := loggedUser(r.ctx)
	if !user.IsAdmin {
		return 0, rest.ErrPermissionDenied
	}
	del := Delete(r.tableName).Where(Eq{"missing": true})
	return r.executeSQL(del)
}

func (r *mediaFileRepository) DeleteMissing(ids []string) error {
	user := loggedUser(r.ctx)
	if !user.IsAdmin {
		return rest.ErrPermissionDenied
	}
	return r.delete(
		And{
			Eq{"missing": true},
			Eq{"id": ids},
		},
	)
}

func (r *mediaFileRepository) MarkMissing(missing bool, mfs ...*model.MediaFile) error {
	ids := slice.SeqFunc(mfs, func(m *model.MediaFile) string { return m.ID })
	for chunk := range slice.CollectChunks(ids, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", time.Now()).
			Where(Eq{"id": chunk})
		c, err := r.executeSQL(upd)
		if err != nil || c == 0 {
			log.Error(r.ctx, "Error setting mediafile missing flag", "ids", chunk, err)
			return err
		}
		log.Debug(r.ctx, "Marked missing mediafiles", "total", c, "ids", chunk)
	}
	return nil
}

func (r *mediaFileRepository) MarkMissingByFolder(missing bool, folderIDs ...string) error {
	for chunk := range slices.Chunk(folderIDs, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", time.Now()).
			Where(And{
				Eq{"folder_id": chunk},
				Eq{"missing": !missing},
			})
		c, err := r.executeSQL(upd)
		if err != nil {
			log.Error(r.ctx, "Error setting mediafile missing flag", "folderIDs", chunk, err)
			return err
		}
		log.Debug(r.ctx, "Marked missing mediafiles from missing folders", "total", c, "folders", chunk)
	}
	return nil
}

// GetMissingAndMatching returns all mediafiles that are missing and their potential matches (comparing PIDs)
// that were added/updated after the last scan started. The result is ordered by PID.
// It does not need to load bookmarks, annotations and participants, as they are not used by the scanner.
func (r *mediaFileRepository) GetMissingAndMatching(libId int) (model.MediaFileCursor, error) {
	subQ := r.newSelect().Columns("pid").
		Where(And{
			Eq{"media_file.missing": true},
			Eq{"library_id": libId},
		})
	subQText, subQArgs, err := subQ.PlaceholderFormat(Question).ToSql()
	if err != nil {
		return nil, err
	}
	sel := r.newSelect().Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id").
		Where("pid in ("+subQText+")", subQArgs...).
		Where(Or{
			Eq{"missing": true},
			ConcatExpr("media_file.created_at > library.last_scan_started_at"),
		}).
		OrderBy("pid")
	cursor, err := queryWithStableResults[dbMediaFile](r.sqlRepository, sel)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.MediaFile, error) bool) {
		for m, err := range cursor {
			if !yield(*m.MediaFile, err) || err != nil {
				return
			}
		}
	}, nil
}

// FindRecentFilesByMBZTrackID finds recently added files by MusicBrainz Track ID in other libraries
func (r *mediaFileRepository) FindRecentFilesByMBZTrackID(missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.selectMediaFile().Where(And{
		NotEq{"media_file.library_id": missing.LibraryID},
		Eq{"media_file.mbz_release_track_id": missing.MbzReleaseTrackID},
		NotEq{"media_file.mbz_release_track_id": ""}, // Exclude empty MBZ Track IDs
		Eq{"media_file.suffix": missing.Suffix},
		Gt{"media_file.created_at": since},
		Eq{"media_file.missing": false},
	}).OrderBy("media_file.created_at DESC")

	var res dbMediaFiles
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

// FindRecentFilesByProperties finds recently added files by intrinsic properties in other libraries
func (r *mediaFileRepository) FindRecentFilesByProperties(missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.selectMediaFile().Where(And{
		NotEq{"media_file.library_id": missing.LibraryID},
		Eq{"media_file.title": missing.Title},
		Eq{"media_file.size": missing.Size},
		Eq{"media_file.suffix": missing.Suffix},
		Eq{"media_file.disc_number": missing.DiscNumber},
		Eq{"media_file.track_number": missing.TrackNumber},
		Eq{"media_file.album": missing.Album},
		Eq{"media_file.mbz_release_track_id": ""}, // Exclude files with MBZ Track ID
		Gt{"media_file.created_at": since},
		Eq{"media_file.missing": false},
	}).OrderBy("media_file.created_at DESC")

	var res dbMediaFiles
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) Search(q string, offset int, size int, options ...model.QueryOptions) (model.MediaFiles, error) {
	var res dbMediaFiles
	if uuid.Validate(q) == nil {
		err := r.searchByMBID(r.selectMediaFile(options...), q, []string{"mbz_recording_id", "mbz_release_track_id"}, &res)
		if err != nil {
			return nil, fmt.Errorf("searching media_file by MBID %q: %w", q, err)
		}
	} else {
		err := r.doSearch(r.selectMediaFile(options...), q, offset, size, &res, "media_file.rowid", "title")
		if err != nil {
			return nil, fmt.Errorf("searching media_file by query %q: %w", q, err)
		}
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *mediaFileRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) EntityName() string {
	return "mediafile"
}

func (r *mediaFileRepository) NewInstance() interface{} {
	return &model.MediaFile{}
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ model.ResourceRepository = (*mediaFileRepository)(nil)
