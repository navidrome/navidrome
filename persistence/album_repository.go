package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
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

type albumRepository struct {
	sqlRepository
}

type dbAlbum struct {
	*model.Album `structs:",flatten"`
	Discs        string `structs:"-" json:"discs"`
	Participants string `structs:"-" json:"-"`
	Tags         string `structs:"-" json:"-"`
	FolderIDs    string `structs:"-" json:"-"`
}

func (a *dbAlbum) PostScan() error {
	var err error
	if a.Discs != "" {
		if err = json.Unmarshal([]byte(a.Discs), &a.Album.Discs); err != nil {
			return fmt.Errorf("parsing album discs from db: %w", err)
		}
	}
	a.Album.Participants, err = unmarshalParticipants(a.Participants)
	if err != nil {
		return fmt.Errorf("parsing album from db: %w", err)
	}
	if a.Tags != "" {
		a.Album.Tags, err = unmarshalTags(a.Tags)
		if err != nil {
			return fmt.Errorf("parsing album from db: %w", err)
		}
		a.Genre, a.Genres = a.Album.Tags.ToGenres()
	}
	if a.FolderIDs != "" {
		var ids []string
		if err = json.Unmarshal([]byte(a.FolderIDs), &ids); err != nil {
			return fmt.Errorf("parsing album folder_ids from db: %w", err)
		}
		a.Album.FolderIDs = ids
	}
	return nil
}

func (a *dbAlbum) PostMapArgs(args map[string]any) error {
	fullText := []string{a.Name, a.SortAlbumName, a.AlbumArtist}
	fullText = append(fullText, a.Album.Participants.AllNames()...)
	fullText = append(fullText, slices.Collect(maps.Values(a.Album.Discs))...)
	fullText = append(fullText, a.Album.Tags[model.TagAlbumVersion]...)
	fullText = append(fullText, a.Album.Tags[model.TagCatalogNumber]...)
	args["full_text"] = formatFullText(fullText...)

	args["tags"] = marshalTags(a.Album.Tags)
	args["participants"] = marshalParticipants(a.Album.Participants)

	folderIDs, err := json.Marshal(a.Album.FolderIDs)
	if err != nil {
		return fmt.Errorf("marshalling album folder_ids: %w", err)
	}
	args["folder_ids"] = string(folderIDs)

	b, err := json.Marshal(a.Album.Discs)
	if err != nil {
		return fmt.Errorf("marshalling album discs: %w", err)
	}
	args["discs"] = string(b)
	return nil
}

type dbAlbums []dbAlbum

func (as dbAlbums) toModels() model.Albums {
	return slice.Map(as, func(a dbAlbum) model.Album { return *a.Album })
}

func NewAlbumRepository(ctx context.Context, db dbx.Builder) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "album"
	r.registerModel(&model.Album{}, albumFilters())
	r.setSortMappings(map[string]string{
		"name":         "order_album_name, order_album_artist_name",
		"artist":       "compilation, order_album_artist_name, order_album_name",
		"album_artist": "compilation, order_album_artist_name, order_album_name",
		// TODO Rename this to just year (or date)
		"max_year":       "coalesce(nullif(original_date,''), cast(max_year as text)), release_date, name",
		"random":         "random",
		"recently_added": recentlyAddedSort(),
		"starred_at":     "starred, starred_at",
	})
	return r
}

var albumFilters = sync.OnceValue(func() map[string]filterFunc {
	filters := map[string]filterFunc{
		"id":              idFilter("album"),
		"name":            fullTextFilter("album", "mbz_album_id", "mbz_release_group_id"),
		"compilation":     booleanFilter,
		"artist_id":       artistFilter,
		"year":            yearFilter,
		"recently_played": recentlyPlayedFilter,
		"starred":         booleanFilter,
		"has_rating":      hasRatingFilter,
		"missing":         booleanFilter,
		"genre_id":        tagIDFilter,
		"role_total_id":   allRolesFilter,
		"library_id":      libraryIdFilter,
	}
	// Add all album tags as filters
	for tag := range model.AlbumLevelTags() {
		filters[string(tag)] = tagIDFilter
	}

	for role := range model.AllRoles {
		filters["role_"+role+"_id"] = artistRoleFilter
	}

	return filters
})

func recentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "updated_at"
	}
	return "created_at"
}

func recentlyPlayedFilter(string, interface{}) Sqlizer {
	return Gt{"play_count": 0}
}

func hasRatingFilter(string, interface{}) Sqlizer {
	return Gt{"rating": 0}
}

func yearFilter(_ string, value interface{}) Sqlizer {
	return Or{
		And{
			Gt{"min_year": 0},
			LtOrEq{"min_year": value},
			GtOrEq{"max_year": value},
		},
		Eq{"max_year": value},
	}
}

func artistFilter(_ string, value interface{}) Sqlizer {
	return Or{
		Exists("json_tree(participants, '$.albumartist')", Eq{"value": value}),
		Exists("json_tree(participants, '$.artist')", Eq{"value": value}),
	}
}

func artistRoleFilter(name string, value interface{}) Sqlizer {
	roleName := strings.TrimSuffix(strings.TrimPrefix(name, "role_"), "_id")

	// Check if the role name is valid. If not, return an invalid filter
	if _, ok := model.AllRoles[roleName]; !ok {
		return Gt{"": nil}
	}
	return Exists(fmt.Sprintf("json_tree(participants, '$.%s')", roleName), Eq{"value": value})
}

func allRolesFilter(_ string, value interface{}) Sqlizer {
	return Like{"participants": fmt.Sprintf(`%%"%s"%%`, value)}
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.withAnnotation(query, "album.id")
	query = r.applyLibraryFilter(query)
	return r.count(query, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"album.id": id})
}

func (r *albumRepository) Put(al *model.Album) error {
	al.ImportedAt = time.Now()
	id, err := r.put(al.ID, &dbAlbum{Album: al})
	if err != nil {
		return err
	}
	al.ID = id
	if len(al.Participants) > 0 {
		err = r.updateParticipants(al.ID, al.Participants)
		if err != nil {
			return err
		}
	}
	return err
}

// TODO Move external metadata to a separated table
func (r *albumRepository) UpdateExternalInfo(al *model.Album) error {
	_, err := r.put(al.ID, &dbAlbum{Album: al}, "description", "small_image_url", "medium_image_url", "large_image_url", "external_url", "external_info_updated_at")
	return err
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("album.*", "library.path as library_path", "library.name as library_name").
		LeftJoin("library on album.library_id = library.id")
	sql = r.withAnnotation(sql, "album.id")
	return r.applyLibraryFilter(sql)
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	res, err := r.GetAll(model.QueryOptions{Filters: Eq{"album.id": id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	var res dbAlbums
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), err
}

func (r *albumRepository) CopyAttributes(fromID, toID string, columns ...string) error {
	var from dbx.NullStringMap
	err := r.queryOne(Select(columns...).From(r.tableName).Where(Eq{"id": fromID}), &from)
	if err != nil {
		return fmt.Errorf("getting album to copy fields from: %w", err)
	}
	to := make(map[string]interface{})
	for _, col := range columns {
		to[col] = from[col]
	}
	_, err = r.executeSQL(Update(r.tableName).SetMap(to).Where(Eq{"id": toID}))
	return err
}

// Touch flags an album as being scanned by the scanner, but not necessarily updated.
// This is used for when missing tracks are detected for an album during scan.
func (r *albumRepository) Touch(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	for ids := range slices.Chunk(ids, 200) {
		upd := Update(r.tableName).Set("imported_at", time.Now()).Where(Eq{"id": ids})
		c, err := r.executeSQL(upd)
		if err != nil {
			return fmt.Errorf("error touching albums: %w", err)
		}
		log.Debug(r.ctx, "Touching albums", "ids", ids, "updated", c)
	}
	return nil
}

// TouchByMissingFolder touches all albums that have missing folders
func (r *albumRepository) TouchByMissingFolder() (int64, error) {
	upd := Update(r.tableName).Set("imported_at", time.Now()).
		Where(And{
			NotEq{"folder_ids": nil},
			ConcatExpr("EXISTS (SELECT 1 FROM json_each(folder_ids) AS je JOIN main.folder AS f ON je.value = f.id WHERE f.missing = true)"),
		})
	c, err := r.executeSQL(upd)
	if err != nil {
		return 0, fmt.Errorf("error touching albums by missing folder: %w", err)
	}
	return c, nil
}

// GetTouchedAlbums returns all albums that were touched by the scanner for a given library, in the
// current library scan run.
// It does not need to load participants, as they are not used by the scanner.
func (r *albumRepository) GetTouchedAlbums(libID int) (model.AlbumCursor, error) {
	query := r.selectAlbum().
		Where(And{
			Eq{"library.id": libID},
			ConcatExpr("album.imported_at > library.last_scan_at"),
		})
	cursor, err := queryWithStableResults[dbAlbum](r.sqlRepository, query)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.Album, error) bool) {
		for a, err := range cursor {
			if a.Album == nil {
				yield(model.Album{}, fmt.Errorf("unexpected nil album: %v", a))
				return
			}
			if !yield(*a.Album, err) || err != nil {
				return
			}
		}
	}, nil
}

// RefreshPlayCounts updates the play count and last play date annotations for all albums, based
// on the media files associated with them.
func (r *albumRepository) RefreshPlayCounts() (int64, error) {
	query := Expr(`
with play_counts as (
    select user_id, album_id, sum(play_count) as total_play_count, max(play_date) as last_play_date
    from media_file
             join annotation on item_id = media_file.id
    group by user_id, album_id
)
insert into annotation (user_id, item_id, item_type, play_count, play_date)
select user_id, album_id, 'album', total_play_count, last_play_date
from play_counts
where total_play_count > 0
on conflict (user_id, item_id, item_type) do update
    set play_count = excluded.play_count,
        play_date  = excluded.play_date;
`)
	return r.executeSQL(query)
}

func (r *albumRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_id) from media_file)")
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("purging empty albums: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Purged empty albums", "totalDeleted", c)
	}
	return nil
}

func (r *albumRepository) Search(q string, offset int, size int, options ...model.QueryOptions) (model.Albums, error) {
	var res dbAlbums
	if uuid.Validate(q) == nil {
		err := r.searchByMBID(r.selectAlbum(options...), q, []string{"mbz_album_id", "mbz_release_group_id"}, &res)
		if err != nil {
			return nil, fmt.Errorf("searching album by MBID %q: %w", q, err)
		}
	} else {
		err := r.doSearch(r.selectAlbum(options...), q, offset, size, &res, "album.rowid", "name")
		if err != nil {
			return nil, fmt.Errorf("searching album by query %q: %w", q, err)
		}
	}
	return res.toModels(), nil
}

func (r *albumRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *albumRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *albumRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *albumRepository) EntityName() string {
	return "album"
}

func (r *albumRepository) NewInstance() interface{} {
	return &model.Album{}
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
