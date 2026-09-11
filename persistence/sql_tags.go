package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// Format of a tag in the DB
type dbTag struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}
type dbTags map[model.TagName][]dbTag

func unmarshalTags(data string) (model.Tags, error) {
	var dbTags dbTags
	err := json.Unmarshal([]byte(data), &dbTags)
	if err != nil {
		return nil, fmt.Errorf("parsing tags: %w", err)
	}

	res := make(model.Tags, len(dbTags))
	for name, tags := range dbTags {
		res[name] = make([]string, len(tags))
		for i, tag := range tags {
			res[name][i] = tag.Value
		}
	}
	return res, nil
}

func marshalTags(tags model.Tags) string {
	dbTags := dbTags{}
	for name, values := range tags {
		for _, value := range values {
			t := model.NewTag(name, value)
			dbTags[name] = append(dbTags[name], dbTag{ID: t.ID, Value: value})
		}
	}
	res, _ := json.Marshal(dbTags)
	return string(res)
}

// indexedTagNames are the tag types materialized into the <table>_tags join tables, so filtering by
// them is an index-backed semi-join instead of a per-row json_tree(tags) scan. Genre only for now.
var indexedTagNames = []model.TagName{model.TagGenre}

// updateTags rewrites this item's <table>_tags rows from its in-memory tags, mirroring
// updateParticipants (delete-then-insert in the same Put; JOIN to tag skips not-yet-saved ids).
func (r sqlRepository) updateTags(itemID string, tags model.Tags) error {
	del := Delete(r.tableName + "_tags").Where(Eq{r.tableName + "_id": itemID})
	if _, err := r.executeSQL(del); err != nil {
		return err
	}
	var tagIDs []string
	for _, name := range indexedTagNames {
		for _, value := range tags.Values(name) {
			tagIDs = append(tagIDs, model.NewTag(name, value).ID)
		}
	}
	if len(tagIDs) == 0 {
		return nil
	}
	idsJSON, err := json.Marshal(tagIDs)
	if err != nil {
		return fmt.Errorf("marshaling tag ids: %w", err)
	}
	query := fmt.Sprintf(`
		INSERT INTO %[1]s_tags (%[1]s_id, tag_id)
		SELECT ?, je.value FROM jsonb_array_elements_text(?::jsonb) AS je(value)
		JOIN tag ON tag.id = je.value
		ON CONFLICT (%[1]s_id, tag_id) DO NOTHING`, r.tableName)
	_, err = r.executeSQL(Expr(query, itemID, string(idsJSON)))
	return err
}

// genreFilterDef builds indexed genre filters for one item type. Callers use the exported SongGenres / AlbumGenres instances.
type genreFilterDef struct{ idCol, table, joinCol string }

var (
	SongGenres  = genreFilterDef{"media_file.id", "media_file_tags", "media_file_id"}
	AlbumGenres = genreFilterDef{"album.id", "album_tags", "album_id"}
)

// ByID matches items tagged with any of the given genre tag ids (scalar or slice).
func (g genreFilterDef) ByID(tagIDs any) Sqlizer {
	sub, args, _ := Select(g.joinCol).From(g.table).Where(Eq{"tag_id": tagIDs}).ToSql()
	return Expr(g.idCol+" IN ("+sub+")", args...)
}

// ByName matches by genre name (Subsonic passes a name, not an id), resolved through the tag
// dictionary, which is uniquely indexed on (tag_name, tag_value).
func (g genreFilterDef) ByName(genre string) Sqlizer {
	sub, args, _ := Select("jt." + g.joinCol).From(g.table + " jt").
		Join("tag on tag.id = jt.tag_id").
		Where(And{Eq{"tag.tag_name": "genre"}, Like{"tag.tag_value": genre}}).ToSql()
	return Expr(g.idCol+" IN ("+sub+")", args...)
}

// AlbumArtistsByGenreID matches album artists of albums tagged with any of the genre ids. It's a
// two-table join (album_artists ⨝ album_tags), so it doesn't fit the single-table genreFilterDef.
func AlbumArtistsByGenreID(tagIDs []string) Sqlizer {
	sub, args, _ := Select("aa.artist_id").From("album_artists aa").
		Join("album_tags at on at.album_id = aa.album_id").
		Where(And{Eq{"aa.role": "albumartist"}, Eq{"at.tag_id": tagIDs}}).ToSql()
	return Expr("artist.id IN ("+sub+")", args...)
}

func genreFilter(filter genreFilterDef) func(_ string, v any) Sqlizer {
	return func(_ string, v any) Sqlizer {
		return filter.ByID(v)
	}
}

// tagIDFilter matches rows whose tags JSON contains the tag id(s); a "<name>_id" key maps to "$.<name>".
func tagIDFilter(name string, idValue any) Sqlizer {
	name = strings.TrimSuffix(name, "_id")
	return Exists(
		fmt.Sprintf(`jsonb_array_elements(coalesce(tags->'%s', '[]'::jsonb)) AS jt(v)`, name),
		Eq{"jt.v->>'id'": idValue},
	)
}

// tagLibraryIdFilter filters tags based on library access through the library_tag table
func tagLibraryIdFilter(_ string, value any) Sqlizer {
	return Eq{"library_tag.library_id": value}
}

// baseTagRepository provides common functionality for all tag-based repositories.
// It handles CRUD operations with optional filtering by tag name.
type baseTagRepository struct {
	sqlRepository
	tagFilter *model.TagName // nil = no filter (all tags), non-nil = filter by specific tag name
}

// newBaseTagRepository creates a new base tag repository with optional tag filtering.
// If tagFilter is nil, the repository will work with all tags.
// If tagFilter is provided, the repository will only work with tags of that specific name.
func newBaseTagRepository(ctx context.Context, db dbx.Builder, tagFilter *model.TagName) *baseTagRepository {
	r := &baseTagRepository{
		tagFilter: tagFilter,
	}
	r.ctx = ctx
	r.db = db
	r.tableName = "tag"
	r.registerModel(&model.Tag{}, map[string]filterFunc{
		"name":       containsFilter("tag_value"),
		"library_id": tagLibraryIdFilter,
	})
	r.setSortMappings(map[string]string{
		"name": "tag_value",
	})
	return r
}

// applyLibraryFiltering adds the appropriate library joins based on user context
func (r *baseTagRepository) applyLibraryFiltering(sq SelectBuilder) SelectBuilder {
	// Add library_tag join
	sq = sq.LeftJoin("library_tag on library_tag.tag_id = tag.id")

	// For authenticated users, also join with user_library to filter by accessible libraries
	user := loggedUser(r.ctx)
	if user.ID != invalidUserId {
		sq = sq.Join("user_library on user_library.library_id = library_tag.library_id AND user_library.user_id = ?", user.ID)
	}

	return sq
}

// newSelect overrides the base implementation to apply tag name filtering and library filtering.
func (r *baseTagRepository) newSelect(options ...model.QueryOptions) SelectBuilder {
	sq := r.sqlRepository.newSelect(options...)

	// Apply tag name filtering if specified
	if r.tagFilter != nil {
		sq = sq.Where(Eq{"tag.tag_name": *r.tagFilter})
	}

	// Apply library filtering and set up aggregation columns
	sq = r.applyLibraryFiltering(sq).Columns(
		"tag.id",
		"tag.tag_name",
		"tag.tag_value",
		"COALESCE(SUM(library_tag.album_count), 0) as album_count",
		"COALESCE(SUM(library_tag.media_file_count), 0) as song_count",
	).GroupBy("tag.id", "tag.tag_name", "tag.tag_value")

	return sq
}

// ResourceRepository interface implementation

func (r *baseTagRepository) Count(options ...rest.QueryOptions) (int64, error) {
	sq := Select("COUNT(DISTINCT tag.id)").From("tag")

	// Apply tag name filtering if specified
	if r.tagFilter != nil {
		sq = sq.Where(Eq{"tag.tag_name": *r.tagFilter})
	}

	// Apply library filtering
	sq = r.applyLibraryFiltering(sq)

	return r.count(sq, r.parseRestOptions(r.ctx, options...))
}

func (r *baseTagRepository) Read(id string) (any, error) {
	query := r.newSelect().Where(Eq{"id": id})
	var res model.Tag
	err := r.queryOne(query, &res)
	return &res, err
}

func (r *baseTagRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	query := r.newSelect(r.parseRestOptions(r.ctx, options...))
	var res model.TagList
	err := r.queryAll(query, &res)
	return res, err
}

func (r *baseTagRepository) EntityName() string {
	return "tag"
}

func (r *baseTagRepository) NewInstance() any {
	return model.Tag{}
}

// Interface compliance check
var _ model.ResourceRepository = (*baseTagRepository)(nil)
