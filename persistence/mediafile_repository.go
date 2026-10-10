package persistence

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
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

// String guards the promoted MediaFile.String(), which would dereference a nil MediaFile.
func (m dbMediaFile) String() string {
	if m.MediaFile == nil {
		return "<nil>"
	}
	return m.MediaFile.String()
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
	participantNames := m.MediaFile.Participants.AllNames()
	fullText = append(fullText, participantNames...)
	args["full_text"] = formatFullText(fullText...)
	args["search_participants"] = strings.Join(participantNames, " ")
	args["search_normalized"] = str.NormalizeForFTS(m.FullTitle(), m.Album, m.Artist, m.AlbumArtist)
	args["tags"] = marshalTags(m.MediaFile.Tags)
	args["participants"] = marshalParticipants(m.MediaFile.Participants)
	return nil
}

type dbMediaFiles []dbMediaFile

func (m dbMediaFiles) toModels() model.MediaFiles {
	return slice.Map(m, func(mf dbMediaFile) model.MediaFile { return *mf.MediaFile })
}

func NewMediaFileRepository(db dbx.Builder) model.MediaFileRepository {
	r := &mediaFileRepository{}
	r.db = db
	r.tableName = "media_file"
	r.registerModel(&model.MediaFile{}, mediaFileFilter())
	r.setSortMappings(map[string]string{
		"title":          "order_title",
		"artist":         "order_artist_name, order_album_name, release_date, disc_number, track_number",
		"album_artist":   "order_album_artist_name, order_album_name, release_date, disc_number, track_number",
		"album":          "order_album_name, album_id, disc_number, track_number, order_artist_name, " + naturalSort("media_file.title"),
		"random":         "random",
		"created_at":     "media_file.created_at",
		"recently_added": mediaFileRecentlyAddedSort(),
		"starred_at":     "starred, starred_at",
		"rated_at":       "rating, rated_at",
		"year":           "year",
		"genre":          "genre",
		"duration":       "duration",
		"channels":       "channels",
		"bpm":            "bpm",
		"path":           "path",
		"comment":        "comment",
		"play_count":     "play_count",
		"play_date":      "play_date",
		"rating":         "rating",
	})
	return r
}

var mediaFileFilter = sync.OnceValue(func() map[string]filterFunc {
	filters := map[string]filterFunc{
		"id":         idFilter("media_file"),
		"title":      fullTextFilter("media_file", "mbz_recording_id", "mbz_release_track_id"),
		"starred":    annotationBoolFilter("starred"),
		"has_rating": annotationBoolFilter("rating"),
		"genre_id":   genreFilter(SongGenres),
		"missing":    booleanFilter,
		"artists_id": mediaFileArtistFilter,
		"library_id": libraryIdFilter,
		"path":       startsWithFilter("media_file.path"),
	}
	// Add all album tags as filters
	for tag := range model.TagMappings() {
		if _, exists := filters[string(tag)]; !exists {
			filters[string(tag)] = tagIDFilter
		}
	}
	return filters
})

func mediaFileArtistFilter(_ string, value any) Sqlizer {
	return ParticipantIDFilter("media_file", value, model.RoleAlbumArtist, model.RoleArtist)
}

func mediaFileRecentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "media_file.updated_at, media_file.id"
	}
	return "media_file.created_at, media_file.id"
}

func (r *mediaFileRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	query := r.newSelect(ctx)
	query = r.applyLibraryFilter(ctx, query)
	// The annotation join is expensive with count(distinct) and pointless unless a filter uses it.
	if filtersNeedAnnotation(r.applyFilters(query, options...)) {
		query = r.withAnnotation(ctx, query, "media_file.id")
	}
	return r.count(ctx, query, options...)
}

func (r *mediaFileRepository) CountBySuffix(ctx context.Context, options ...model.QueryOptions) (map[string]int64, error) {
	sel := r.newSelect(ctx, options...).
		Columns("lower(suffix) as suffix", "count(*) as count").
		GroupBy("lower(suffix)")
	var res []struct {
		Suffix string
		Count  int64
	}
	err := r.queryAll(ctx, sel, &res)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(res))
	for _, c := range res {
		counts[c.Suffix] = c.Count
	}
	return counts, nil
}

func (r *mediaFileRepository) Exists(ctx context.Context, id string) (bool, error) {
	// The exists() helper applies no library filter, so it would report rows the caller cannot see.
	c, err := r.count(ctx, r.applyLibraryFilter(ctx, r.newSelect(ctx).Where(Eq{"media_file.id": id})))
	return c > 0, err
}

func (r *mediaFileRepository) Put(ctx context.Context, m *model.MediaFile) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	id, err := r.putByMatch(ctx, Eq{"path": m.Path, "library_id": m.LibraryID}, m.ID, &dbMediaFile{MediaFile: m})
	if err != nil {
		return err
	}
	m.ID = id
	if err := r.updateParticipants(ctx, m.ID, m.Participants); err != nil {
		return err
	}
	return r.updateTags(ctx, m.ID, m.Tags)
}

func (r *mediaFileRepository) UpdateProbeData(ctx context.Context, id string, data string) error {
	_, err := r.executeSQL(ctx, Update(r.tableName).Set("probe_data", data).Where(Eq{"id": id}))
	return err
}

func (r *mediaFileRepository) selectMediaFile(ctx context.Context, options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(ctx, options...).Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id")
	sql = r.withAnnotation(ctx, sql, "media_file.id")
	sql = r.withBookmark(ctx, sql, "media_file.id")
	return r.applyLibraryFilter(ctx, sql)
}

func (r *mediaFileRepository) Get(ctx context.Context, id string) (*model.MediaFile, error) {
	res, err := r.GetAll(ctx, model.QueryOptions{Filters: Eq{"media_file.id": id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *mediaFileRepository) GetWithParticipants(ctx context.Context, id string) (*model.MediaFile, error) {
	m, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Participants, err = r.getParticipants(ctx, m)
	return m, err
}

func (r *mediaFileRepository) GetAll(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(ctx, options...)
	var res dbMediaFiles
	err := r.queryAll(ctx, sq, &res, options...)
	if err != nil {
		return nil, err
	}
	mfs := res.toModels()
	r.hydrateArtwork(ctx, mfs)
	return mfs, nil
}

func (r *mediaFileRepository) hydrateArtwork(ctx context.Context, mfs model.MediaFiles) {
	hydrateMediaFileArtwork(ctx, r.db, mfs)
}

// GetRandom uses two passes so the random sort runs over a narrow rowid index instead of the
// wide media_file row: pick random rowids first, then hydrate only those.
func (r *mediaFileRepository) GetRandom(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	var opt model.QueryOptions
	if len(options) > 0 {
		opt = options[0]
	}

	rowidQuery := Select("media_file.rowid").From(r.tableName)
	rowidQuery = r.applyFilters(rowidQuery, model.QueryOptions{Filters: opt.Filters})
	rowidQuery = r.applyLibraryFilter(ctx, rowidQuery)
	rowidQuery = rowidQuery.OrderBy("random()")
	if opt.Max > 0 {
		rowidQuery = rowidQuery.Limit(uint64(opt.Max))
	}

	var rowids []int64
	if err := r.queryAllSlice(ctx, rowidQuery, &rowids); err != nil {
		return nil, err
	}
	if len(rowids) == 0 {
		return model.MediaFiles{}, nil
	}

	// Re-shuffle in Phase 2: `WHERE rowid IN (...)` returns rows in ascending rowid order, not
	// the random order from Phase 1. Sorting only the (<=Max) hydrated rows is negligible.
	sq := r.selectMediaFile(ctx).Where(Eq{"media_file.rowid": rowids}).OrderBy("random()")
	var res dbMediaFiles
	if err := r.queryAll(ctx, sq, &res); err != nil {
		return nil, err
	}
	mfs := res.toModels()
	r.hydrateArtwork(ctx, mfs)
	return mfs, nil
}

func (r *mediaFileRepository) GetAllByTags(ctx context.Context, tag model.TagName, values []string, options ...model.QueryOptions) (model.MediaFiles, error) {
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	tagFilter := Expr(
		fmt.Sprintf("exists (select 1 from json_tree(media_file.tags, '$.%s') where key='value' and value in (%s))",
			tag, strings.Join(placeholders, ",")),
		args...,
	)

	var opts model.QueryOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.Filters != nil {
		opts.Filters = And{tagFilter, opts.Filters}
	} else {
		opts.Filters = tagFilter
	}
	return r.GetAll(ctx, opts)
}

func (r *mediaFileRepository) GetCursor(ctx context.Context, options ...model.QueryOptions) (model.MediaFileCursor, error) {
	sq := r.selectMediaFile(ctx, options...)
	cursor, err := queryWithStableResults[dbMediaFile](ctx, r.sqlRepository, sq)
	if err != nil {
		return nil, err
	}
	return wrapMediaFileCursor(cursor), nil
}

// getAllIDs returns the IDs of GetAll's row set, skipping its wide column projection.
func (r *mediaFileRepository) getAllIDs(ctx context.Context, options ...model.QueryOptions) ([]string, error) {
	sq := r.applyLibraryFilter(ctx, r.newSelect(ctx, options...).Columns("media_file.id"))
	if filtersNeedAnnotation(sq) {
		sq = r.withAnnotation(ctx, sq, "media_file.id")
	}
	ids := []string{}
	err := r.queryAllSlice(ctx, sq, &ids)
	return ids, err
}

func (r *mediaFileRepository) GetAlbumIDsByFolder(ctx context.Context, lib model.Library, folderIDs ...string) ([]string, error) {
	ids := []string{}
	for chunk := range slices.Chunk(folderIDs, 200) {
		// A folder's own cover also covers albums whose tracks sit in its disc subfolders.
		inFolders := Select("f.id").From("folder f").Where(And{
			Eq{"f.library_id": lib.ID},
			Eq{"f.missing": false},
			Or{Eq{"f.id": chunk}, Eq{"f.parent_id": chunk}},
		})
		sq := Select("distinct album_id").From("media_file").
			Where(And{Eq{"missing": false}, ConcatExpr("folder_id IN (", inFolders, ")")})
		var chunkIDs []string
		if err := r.queryAllSlice(ctx, sq, &chunkIDs); err != nil {
			return nil, err
		}
		ids = append(ids, chunkIDs...)
	}
	return ids, nil
}

// GetCursorWithArtwork streams the same rows as GetCursor, hydrated, via an id pre-pass.
func (r *mediaFileRepository) GetCursorWithArtwork(ctx context.Context, options ...model.QueryOptions) (model.MediaFileCursor, error) {
	ids, err := r.getAllIDs(ctx, options...)
	if err != nil {
		return nil, err
	}
	opts := chunkOptions(options, "media_file.id")
	return model.MediaFileCursor(streamByIDs(ids, func(chunk []string) (model.MediaFiles, error) {
		return r.GetAll(ctx, opts(chunk))
	})), nil
}

// FindByPaths finds media files by their paths.
// The paths can be library-qualified (format: "libraryID:path") or unqualified ("path").
// Library-qualified paths search within the specified library, while unqualified paths
// search across all libraries for backward compatibility.
func (r *mediaFileRepository) FindByPaths(ctx context.Context, paths []string) (model.MediaFiles, error) {
	// One IN list per library instead of one OR term per path: SQLite abandons the
	// path index at just two OR-ed equality terms and scans the whole table.
	byLibrary := map[int][]string{}
	var unqualified []string

	for _, path := range paths {
		// A numeric prefix is ambiguous: "1:foo.mp3" qualifies a library, but "1999: A Life/01.mp3"
		// is a plain path. Search both ways rather than guessing.
		if id, rest, ok := strings.Cut(path, ":"); ok {
			if libraryID, err := strconv.Atoi(id); err == nil {
				byLibrary[libraryID] = append(byLibrary[libraryID], rest)
			}
		}
		unqualified = append(unqualified, path)
	}

	query := Or{}
	for _, libraryID := range slices.Sorted(maps.Keys(byLibrary)) {
		query = append(query, And{
			Eq{"path collate nocase": byLibrary[libraryID]},
			Eq{"library_id": libraryID},
		})
	}
	if len(unqualified) > 0 {
		query = append(query, Eq{"path collate nocase": unqualified})
	}

	if len(query) == 0 {
		return model.MediaFiles{}, nil
	}

	sel := r.applyLibraryFilter(ctx, r.newSelect(ctx).Columns("*").Where(query))
	var res dbMediaFiles
	if err := r.queryAll(ctx, sel, &res); err != nil {
		return nil, err
	}

	return res.toModels(), nil
}

func (r *mediaFileRepository) Delete(ctx context.Context, id string) error {
	return r.delete(ctx, Eq{"id": id})
}

func (r *mediaFileRepository) ReassignReferences(ctx context.Context, prevID, newID string) error {
	if err := r.ReassignAnnotation(ctx, prevID, newID); err != nil {
		return fmt.Errorf("reassigning annotations: %w", err)
	}
	if err := r.reassignBookmark(ctx, prevID, newID); err != nil {
		return fmt.Errorf("reassigning bookmarks: %w", err)
	}
	upd := Update("playlist_tracks").Set("media_file_id", newID).Where(Eq{"media_file_id": prevID})
	if _, err := r.executeSQL(ctx, upd); err != nil {
		return fmt.Errorf("reassigning playlist tracks: %w", err)
	}
	upd = Update("scrobbles").Set("media_file_id", newID).Where(Eq{"media_file_id": prevID})
	if _, err := r.executeSQL(ctx, upd); err != nil {
		return fmt.Errorf("reassigning scrobbles: %w", err)
	}
	// OR IGNORE: scrobble_buffer is unique on (user_id, service, media_file_id, play_time)
	buf := Expr("update or ignore scrobble_buffer set media_file_id = ? where media_file_id = ?", newID, prevID)
	if _, err := r.executeSQL(ctx, buf); err != nil {
		return fmt.Errorf("reassigning buffered scrobbles: %w", err)
	}
	return nil
}

func (r *mediaFileRepository) DeleteAllMissing(ctx context.Context) (int64, error) {
	user := loggedUser(ctx)
	if !user.IsAdmin {
		return 0, rest.ErrPermissionDenied
	}
	del := Delete(r.tableName).Where(Eq{"missing": true})
	return r.executeSQL(ctx, del)
}

func (r *mediaFileRepository) DeleteMissing(ctx context.Context, ids []string) error {
	user := loggedUser(ctx)
	if !user.IsAdmin {
		return rest.ErrPermissionDenied
	}
	return r.delete(ctx,
		And{
			Eq{"missing": true},
			Eq{"id": ids},
		},
	)
}

func (r *mediaFileRepository) MarkMissing(ctx context.Context, missing bool, mfs ...*model.MediaFile) error {
	ids := slice.SeqFunc(mfs, func(m *model.MediaFile) string { return m.ID })
	for chunk := range slice.CollectChunks(ids, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", time.Now()).
			Where(Eq{"id": chunk})
		c, err := r.executeSQL(ctx, upd)
		if err != nil || c == 0 {
			log.Error(ctx, "Error setting mediafile missing flag", "ids", chunk, err)
			return err
		}
		log.Debug(ctx, "Marked missing mediafiles", "total", c, "ids", chunk)
	}
	return nil
}

func (r *mediaFileRepository) MarkMissingByFolder(ctx context.Context, missing bool, folderIDs ...string) error {
	for chunk := range slices.Chunk(folderIDs, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", time.Now()).
			Where(And{
				Eq{"folder_id": chunk},
				Eq{"missing": !missing},
			})
		c, err := r.executeSQL(ctx, upd)
		if err != nil {
			log.Error(ctx, "Error setting mediafile missing flag", "folderIDs", chunk, err)
			return err
		}
		log.Debug(ctx, "Marked missing mediafiles from missing folders", "total", c, "folders", chunk)
	}
	return nil
}

// GetMissingAndMatching returns all mediafiles that are missing and their potential matches (comparing PIDs)
// that were added/updated after the last scan started. The result is ordered by PID.
// It does not need to load bookmarks, annotations and participants, as they are not used by the scanner.
func (r *mediaFileRepository) GetMissingAndMatching(ctx context.Context, libId int) (model.MediaFileCursor, error) {
	subQ := r.newSelect(ctx).Columns("pid").
		Where(And{
			Eq{"media_file.missing": true},
			Eq{"library_id": libId},
		})
	subQText, subQArgs, err := subQ.PlaceholderFormat(Question).ToSql()
	if err != nil {
		return nil, err
	}
	sel := r.newSelect(ctx).Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id").
		Where("pid in ("+subQText+")", subQArgs...).
		Where(Or{
			Eq{"missing": true},
			ConcatExpr("media_file.created_at > library.last_scan_started_at"),
		}).
		OrderBy("pid")
	cursor, err := queryWithStableResults[dbMediaFile](ctx, r.sqlRepository, sel)
	if err != nil {
		return nil, err
	}
	return wrapMediaFileCursor(cursor), nil
}

func wrapMediaFileCursor(cursor iter.Seq2[dbMediaFile, error]) model.MediaFileCursor {
	return model.MediaFileCursor(wrapCursor(cursor, func(m dbMediaFile) *model.MediaFile { return m.MediaFile }))
}

// FindRecentFilesByMBZTrackID finds recently added files by MusicBrainz Track ID in other libraries
// It uses a lightweight query without annotation/bookmark joins since those are not needed for matching
func (r *mediaFileRepository) FindRecentFilesByMBZTrackID(ctx context.Context, missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.newSelect(ctx).Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id").
		Where(And{
			NotEq{"media_file.library_id": missing.LibraryID},
			Eq{"media_file.mbz_release_track_id": missing.MbzReleaseTrackID},
			NotEq{"media_file.mbz_release_track_id": ""}, // Exclude empty MBZ Track IDs
			Eq{"media_file.suffix": missing.Suffix},
			Gt{"media_file.created_at": since},
			Eq{"media_file.missing": false},
		}).OrderBy("media_file.created_at DESC")

	var res dbMediaFiles
	err := r.queryAll(ctx, sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

// FindRecentFilesByProperties finds recently added files by intrinsic properties in other libraries
// It uses a lightweight query without annotation/bookmark joins since those are not needed for matching
func (r *mediaFileRepository) FindRecentFilesByProperties(ctx context.Context, missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.newSelect(ctx).Columns("media_file.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on media_file.library_id = library.id").
		Where(And{
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
	err := r.queryAll(ctx, sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

var mediaFileSearchConfig = searchConfig{
	NaturalOrder: "media_file.rowid",
	OrderBy:      []string{"title"},
	MBIDFields:   []string{"mbz_recording_id", "mbz_release_track_id"},
}

func (r *mediaFileRepository) MatchesCriteria(ctx context.Context, id string, c criteria.Criteria) (bool, error) {
	usr := loggedUser(ctx)
	rulesSQL := newSmartPlaylistCriteria(c, withSmartPlaylistOwner(*usr))
	cond, err := rulesSQL.where()
	if err != nil {
		return false, err
	}
	sq := Select("count(*) as count").From("media_file")
	sq = rulesSQL.applyExpressionJoins(sq, usr.ID)
	sq = sq.Where(And{Eq{"media_file.id": id}, cond})
	var res struct{ Count int64 }
	if err := r.queryOne(ctx, sq, &res); err != nil {
		return false, err
	}
	return res.Count > 0, nil
}

func (r *mediaFileRepository) Search(ctx context.Context, q string, options ...model.QueryOptions) (model.MediaFiles, error) {
	var opts model.QueryOptions
	if len(options) > 0 {
		opts = options[0]
	}
	var res dbMediaFiles
	err := r.doSearch(ctx, r.selectMediaFile(ctx, options...), q, &res, mediaFileSearchConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("searching media_file %q: %w", q, err)
	}
	mfs := res.toModels()
	r.hydrateArtwork(ctx, mfs)
	return mfs, nil
}

func (r *mediaFileRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *mediaFileRepository) Read(ctx context.Context, id string) (*model.MediaFile, error) {
	return r.Get(ctx, id)
}

func (r *mediaFileRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.MediaFile, error) {
	return r.GetAll(ctx, r.parseRestOptions(ctx, options...))
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ rest.Repository[model.MediaFile] = (*mediaFileRepository)(nil)
