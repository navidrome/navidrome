package persistence

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/ftsnormalize"
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
	participantNames := m.MediaFile.Participants.AllNames()
	fullText = append(fullText, participantNames...)
	args["full_text"] = formatFullText(fullText...)
	args["search_participants"] = strings.Join(participantNames, " ")
	args["search_normalized"] = mediaFileSearchNormalized(context.Background(), m.MediaFile)
	args["tags"] = marshalTags(m.MediaFile.Tags)
	args["participants"] = marshalParticipants(m.MediaFile.Participants)
	normalizeMediaFileNumericArgs(args)
	return nil
}

// mediaFileSearchNormalized prefers the value already computed by the Rust map_media
// worker (embedded in media_file_json) so Put/PutAll avoid a second normalize RPC.
// Recompute when Subsonic.AppendSubtitle changes FullTitle relative to the Rust title.
func mediaFileSearchNormalized(ctx context.Context, m *model.MediaFile) string {
	if m == nil {
		return ""
	}
	if m.SearchNormalized != "" && m.FullTitle() == m.Title {
		return m.SearchNormalized
	}
	normalized := ftsnormalize.NormalizeForFTS(ctx, m.FullTitle(), m.Album, m.Artist, m.AlbumArtist)
	m.SearchNormalized = normalized
	return normalized
}

func normalizeMediaFileNumericArgs(args map[string]any) {
	for _, col := range []string{
		"bpm",
		"duration",
		"bit_rate",
		"sample_rate",
		"channels",
		"disc_number",
		"track_number",
		"year",
		"size",
	} {
		if isNilValue(args[col]) {
			args[col] = 0
		}
	}
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
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
		"starred":    wrapFilter(annotationBoolFilter("starred")),
		"has_rating": wrapFilter(annotationBoolFilter("rating")),
		"genre_id":   wrapFilter(genreFilter(SongGenres)),
		"missing":    booleanFilter,
		"artists_id": wrapFilter(mediaFileArtistFilter),
		"library_id": wrapFilter(libraryIdFilter),
		"path":       startsWithFilter("media_file.path"),
	}
	// Add all album tags as filters
	for tag := range model.TagMappings() {
		if _, exists := filters[string(tag)]; !exists {
			filters[string(tag)] = wrapFilter(tagIDFilter)
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

func (r *mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.applyLibraryFilter(query)
	// The annotation join is expensive with count(distinct) and pointless unless a filter uses it.
	if filtersNeedAnnotation(r.applyFilters(query, options...)) {
		query = r.withAnnotation(query, "media_file.id")
	}
	return r.count(query, options...)
}

func (r *mediaFileRepository) CountBySuffix(options ...model.QueryOptions) (map[string]int64, error) {
	sel := r.newSelect(options...).
		Columns("lower(suffix) as suffix", "count(*) as count").
		GroupBy("lower(suffix)")
	var res []struct {
		Suffix string
		Count  int64
	}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(res))
	for _, c := range res {
		counts[c.Suffix] = c.Count
	}
	return counts, nil
}

func (r *mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"media_file.id": id})
}

func (r *mediaFileRepository) Put(m *model.MediaFile) error {
	prefillMediaFileSearchNormalized(r.ctx, []*model.MediaFile{m})
	if err := r.putMediaFile(m); err != nil {
		return err
	}
	if err := r.updateParticipants(m.ID, m.Participants); err != nil {
		return err
	}
	return r.updateTags(m.ID, m.Tags)
}

// PutAll persists scanner batches while collapsing relationship-table rewrites
// into one delete and one insert per table. SQLite remains the single writer and
// the caller's folder transaction remains the atomic consistency boundary.
func (r *mediaFileRepository) PutAll(mediaFiles ...*model.MediaFile) error {
	prefillMediaFileSearchNormalized(r.ctx, mediaFiles)
	participantUpdates := make([]participantUpdate, 0, len(mediaFiles))
	tagUpdates := make([]tagUpdate, 0, len(mediaFiles))
	for _, mediaFile := range mediaFiles {
		if err := r.putMediaFile(mediaFile); err != nil {
			return err
		}
		participantUpdates = append(participantUpdates, participantUpdate{itemID: mediaFile.ID, participants: mediaFile.Participants})
		tagUpdates = append(tagUpdates, tagUpdate{itemID: mediaFile.ID, tags: mediaFile.Tags})
	}
	if err := r.updateParticipantsBatch(participantUpdates); err != nil {
		return err
	}
	return r.updateTagsBatch(tagUpdates)
}

// prefillMediaFileSearchNormalized fills SearchNormalized for rows that did not
// come from Rust map_media, collapsing N Put normalize hops into one batch RPC.
func prefillMediaFileSearchNormalized(ctx context.Context, mediaFiles []*model.MediaFile) {
	groups := make([][]string, 0, len(mediaFiles))
	indexes := make([]int, 0, len(mediaFiles))
	for i, mediaFile := range mediaFiles {
		if mediaFile == nil {
			continue
		}
		if mediaFile.SearchNormalized != "" && mediaFile.FullTitle() == mediaFile.Title {
			continue
		}
		groups = append(groups, []string{mediaFile.FullTitle(), mediaFile.Album, mediaFile.Artist, mediaFile.AlbumArtist})
		indexes = append(indexes, i)
	}
	if len(groups) == 0 {
		return
	}
	normalized := ftsnormalize.NormalizeMany(ctx, groups)
	for j, idx := range indexes {
		mediaFiles[idx].SearchNormalized = normalized[j]
	}
}

func (r *mediaFileRepository) putMediaFile(m *model.MediaFile) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	id, err := r.putByMatch(Eq{"path": m.Path, "library_id": m.LibraryID}, m.ID, &dbMediaFile{MediaFile: m})
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

func (r *mediaFileRepository) UpdateProbeData(id string, data string) error {
	_, err := r.executeSQL(Update(r.tableName).Set("probe_data", data).Where(Eq{"id": id}))
	return err
}

func (r *mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	columns := []string{"media_file.*", "library.path as library_path", "library.name as library_name"}
	if len(options) > 0 && options[0].ExcludeHeavyFields {
		columns = browseMediaFileColumnExprs("media_file")
	}
	sql := r.newSelect(options...).Columns(columns...).
		LeftJoin("library on media_file.library_id = library.id")
	sql = r.withAnnotation(sql, "media_file.id")
	sql = r.withBookmark(sql, "media_file.id")
	return r.applyLibraryFilter(sql)
}

func browseMediaFileColumnExprs(table string) []string {
	cols := []string{
		"id", "library_id", "path", "title", "album", "artist", "album_artist",
		"sort_title", "sort_album_name", "sort_artist_name", "sort_album_artist_name",
		"order_title", "order_album_name", "order_artist_name", "order_album_artist_name",
		"genre", "compilation", "track_number", "disc_number", "disc_subtitle",
		"duration", "size", "suffix", "bit_rate", "sample_rate", "bit_depth", "channels", "codec",
		"explicit_status", "original_year", "original_date", "release_year", "release_date",
		"year", "date", "mbz_recording_id", "mbz_release_track_id", "mbz_album_id",
		"mbz_release_group_id", "mbz_album_type", "rg_album_peak", "rg_album_gain",
		"rg_track_peak", "rg_track_gain", "created_at", "updated_at", "album_id",
		"artist_id", "album_artist_id", "catalog_num", "comment", "bpm", "tags", "participants",
	}
	out := make([]string, 0, len(cols)+2)
	for _, col := range cols {
		out = append(out, table+"."+col)
	}
	out = append(out, "library.path as library_path", "library.name as library_name")
	return out
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

func (r *mediaFileRepository) GetForStreaming(id string) (*model.MediaFile, error) {
	sq := r.newSelect().Columns(
		"media_file.id",
		"media_file.library_id",
		"media_file.path",
		"media_file.title",
		"media_file.artist",
		"media_file.suffix",
		"media_file.duration",
		"media_file.size",
		"media_file.bit_rate",
		"media_file.sample_rate",
		"media_file.bit_depth",
		"media_file.channels",
		"media_file.codec",
		"media_file.probe_data",
		"media_file.updated_at",
		"'{}' as participants",
		"'{}' as tags",
		"library.path as library_path",
		"library.name as library_name",
	).
		LeftJoin("library on media_file.library_id = library.id").
		Where(Eq{"media_file.id": id})
	sq = r.applyLibraryFilter(sq)

	var res dbMediaFile
	if err := r.queryOne(sq, &res); err != nil {
		return nil, err
	}
	return res.MediaFile, nil
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

// GetRandom uses two passes so the random sort runs over a narrow rowid index instead of the
// wide media_file row: pick random rowids first, then hydrate only those.
func (r *mediaFileRepository) GetRandom(options ...model.QueryOptions) (model.MediaFiles, error) {
	var opt model.QueryOptions
	if len(options) > 0 {
		opt = options[0]
	}

	rowidQuery := Select("media_file.rowid").From(r.tableName)
	rowidQuery = r.applyFilters(rowidQuery, model.QueryOptions{Filters: opt.Filters})
	rowidQuery = r.applyLibraryFilter(rowidQuery)
	rowidQuery = rowidQuery.OrderBy("random()")
	if opt.Max > 0 {
		rowidQuery = rowidQuery.Limit(uint64(opt.Max))
	}

	var rowids []int64
	if err := r.queryAllSlice(rowidQuery, &rowids); err != nil {
		return nil, err
	}
	if len(rowids) == 0 {
		return model.MediaFiles{}, nil
	}

	// Re-shuffle in Phase 2: `WHERE rowid IN (...)` returns rows in ascending rowid order, not
	// the random order from Phase 1. Sorting only the (<=Max) hydrated rows is negligible.
	sq := r.selectMediaFile(opt).Where(Eq{"media_file.rowid": rowids}).OrderBy("random()")
	var res dbMediaFiles
	if err := r.queryAll(sq, &res); err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) GetAllByTags(tag model.TagName, values []string, options ...model.QueryOptions) (model.MediaFiles, error) {
	var tagFilter Sqlizer
	if IsIndexedTag(tag) {
		tagFilter = SongTags.ByTagValues(tag, values)
	} else {
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = "?"
			args[i] = v
		}
		tagFilter = Expr(
			fmt.Sprintf("exists (select 1 from json_tree(media_file.tags, '$.%s') where key='value' and value in (%s))",
				tag, strings.Join(placeholders, ",")),
			args...,
		)
	}

	var opts model.QueryOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.Filters != nil {
		opts.Filters = And{tagFilter, opts.Filters}
	} else {
		opts.Filters = tagFilter
	}
	return r.GetAll(opts)
}

func (r *mediaFileRepository) GetCursor(options ...model.QueryOptions) (model.MediaFileCursor, error) {
	sq := r.selectMediaFile(options...)
	cursor, err := queryWithStableResults[dbMediaFile](r.sqlRepository, sq)
	if err != nil {
		return nil, err
	}
	return wrapMediaFileCursor(cursor), nil
}

// FindByPaths finds media files by their paths.
// The paths can be library-qualified (format: "libraryID:path") or unqualified ("path").
// Library-qualified paths search within the specified library, while unqualified paths
// search across all libraries for backward compatibility.
func (r *mediaFileRepository) FindByPaths(paths []string) (model.MediaFiles, error) {
	// One IN list per library instead of one OR term per path: SQLite abandons the
	// path index at just two OR-ed equality terms and scans the whole table.
	byLibrary := map[int][]string{}
	var unqualified []string

	for _, path := range paths {
		parts := strings.SplitN(path, ":", 2)
		if len(parts) == 2 {
			// Library-qualified path: "libraryID:path"
			libraryID, err := strconv.Atoi(parts[0])
			if err != nil {
				// Invalid format, skip
				continue
			}
			byLibrary[libraryID] = append(byLibrary[libraryID], parts[1])
		} else {
			// Unqualified path: search across all libraries
			unqualified = append(unqualified, path)
		}
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

	sel := r.applyLibraryFilter(r.newSelect().Columns("*").Where(query))
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
	return wrapMediaFileCursor(cursor), nil
}

func wrapMediaFileCursor(cursor iter.Seq2[dbMediaFile, error]) model.MediaFileCursor {
	return func(yield func(model.MediaFile, error) bool) {
		for m, err := range cursor {
			if m.MediaFile == nil {
				yield(model.MediaFile{}, fmt.Errorf("unexpected nil mediafile (%v): %w", m, err))
				return
			}
			if !yield(*m.MediaFile, err) || err != nil {
				return
			}
		}
	}
}

// FindRecentFilesByMBZTrackID finds recently added files by MusicBrainz Track ID in other libraries
// It uses a lightweight query without annotation/bookmark joins since those are not needed for matching
func (r *mediaFileRepository) FindRecentFilesByMBZTrackID(missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.newSelect().Columns("media_file.*", "library.path as library_path", "library.name as library_name").
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
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

// FindRecentFilesByProperties finds recently added files by intrinsic properties in other libraries
// It uses a lightweight query without annotation/bookmark joins since those are not needed for matching
func (r *mediaFileRepository) FindRecentFilesByProperties(missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	sel := r.newSelect().Columns("media_file.*", "library.path as library_path", "library.name as library_name").
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
	err := r.queryAll(sel, &res)
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

func (r *mediaFileRepository) Search(q string, options ...model.QueryOptions) (model.MediaFiles, error) {
	var opts model.QueryOptions
	if len(options) > 0 {
		opts = options[0]
	}
	var res dbMediaFiles
	err := r.doSearch(r.selectMediaFile(options...), q, &res, mediaFileSearchConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("searching media_file %q: %w", q, err)
	}
	return res.toModels(), nil
}

func (r *mediaFileRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *mediaFileRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) EntityName() string {
	return "mediafile"
}

func (r *mediaFileRepository) NewInstance() any {
	return &model.MediaFile{}
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ model.ResourceRepository = (*mediaFileRepository)(nil)
