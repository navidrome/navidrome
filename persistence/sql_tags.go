package persistence

import (
	"context"
	"fmt"
	"slices"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	json "github.com/goccy/go-json"
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
// them is an index-backed semi-join instead of a per-row json_tree(tags) scan.
var indexedTagNames = []model.TagName{model.TagGenre, model.TagMood, model.TagReleaseType}

// IsIndexedTag reports whether tag values are stored in the *_tags join tables.
func IsIndexedTag(name model.TagName) bool {
	return slices.Contains(indexedTagNames, name)
}

type tagUpdate struct {
	itemID string
	tags   model.Tags
}

type flatTag struct {
	ItemID string `json:"item_id"`
	TagID  string `json:"tag_id"`
}

// updateTags rewrites this item's <table>_tags rows from its in-memory tags, mirroring
// updateParticipants (delete-then-insert in the same Put; JOIN to tag skips not-yet-saved ids).
func (r sqlRepository) updateTags(itemID string, tags model.Tags) error {
	return r.updateTagsBatch([]tagUpdate{{itemID: itemID, tags: tags}})
}

func (r sqlRepository) updateTagsBatch(updates []tagUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	itemIDs := make([]string, 0, len(updates))
	flatTags := make([]flatTag, 0)
	for _, update := range updates {
		itemIDs = append(itemIDs, update.itemID)
		for _, name := range indexedTagNames {
			for _, value := range update.tags.Values(name) {
				flatTags = append(flatTags, flatTag{ItemID: update.itemID, TagID: model.NewTag(name, value).ID})
			}
		}
	}
	itemIDsJSON, err := json.Marshal(itemIDs)
	if err != nil {
		return fmt.Errorf("marshaling tag item ids: %w", err)
	}
	itemIDColumn := r.tableName + "_id"
	del := Delete(r.tableName + "_tags").Where(Expr(
		itemIDColumn+" IN (SELECT value FROM json_each(?))", string(itemIDsJSON),
	))
	if _, err := r.executeSQL(del); err != nil {
		return err
	}
	if len(flatTags) == 0 {
		return nil
	}
	tagsJSON, err := json.Marshal(flatTags)
	if err != nil {
		return fmt.Errorf("marshaling indexed tags: %w", err)
	}
	query := fmt.Sprintf(`
		INSERT INTO %[1]s_tags (%[1]s_id, tag_id)
		SELECT json_extract(value, '$.item_id'), json_extract(value, '$.tag_id')
		FROM json_each(?)
		JOIN tag ON tag.id = json_extract(value, '$.tag_id')
		ON CONFLICT (%[1]s_id, tag_id) DO NOTHING`, r.tableName)
	_, err = r.executeSQL(Expr(query, string(tagsJSON)))
	return err
}

// itemTagFilterDef builds indexed tag filters for one item type.
type itemTagFilterDef struct{ idCol, table, joinCol string }

var (
	SongTags  = itemTagFilterDef{"media_file.id", "media_file_tags", "media_file_id"}
	AlbumTags = itemTagFilterDef{"album.id", "album_tags", "album_id"}
	// SongGenres and AlbumGenres are kept as aliases for genre-specific call sites.
	SongGenres  = SongTags
	AlbumGenres = AlbumTags
)

// ByID matches items tagged with any of the given tag ids (scalar or slice).
func (g itemTagFilterDef) ByID(tagIDs any) Sqlizer {
	sub, args, _ := Select(g.joinCol).From(g.table).Where(Eq{"tag_id": tagIDs}).ToSql()
	return Expr(g.idCol+" IN ("+sub+")", args...)
}

// ByTagName matches items by tag name and value through the tag dictionary.
func (g itemTagFilterDef) ByTagName(tagName model.TagName, value string) Sqlizer {
	sub, args, _ := Select("jt."+g.joinCol).From(g.table+" jt").
		Join("tag on tag.id = jt.tag_id").
		Where(And{Eq{"tag.tag_name": tagName}, Like{"tag.tag_value": value}}).ToSql()
	return Expr(g.idCol+" IN ("+sub+")", args...)
}

// ByTagValues matches items tagged with any of the given values for a tag name.
func (g itemTagFilterDef) ByTagValues(tagName model.TagName, values []string) Sqlizer {
	sub, args, _ := Select("jt."+g.joinCol).From(g.table+" jt").
		Join("tag on tag.id = jt.tag_id").
		Where(And{Eq{"tag.tag_name": tagName}, Eq{"tag.tag_value": values}}).ToSql()
	return Expr(g.idCol+" IN ("+sub+")", args...)
}

// ByName matches by genre name (Subsonic passes a name, not an id), resolved through the tag
// dictionary, which is uniquely indexed on (tag_name, tag_value).
func (g itemTagFilterDef) ByName(genre string) Sqlizer {
	return g.ByTagName(model.TagGenre, genre)
}

// AlbumArtistsByGenreID matches album artists of albums tagged with any of the genre ids. It's a
// two-table join (album_artists ⨝ album_tags), so it doesn't fit the single-table genreFilterDef.
func AlbumArtistsByGenreID(tagIDs []string) Sqlizer {
	sub, args, _ := Select("aa.artist_id").From("album_artists aa").
		Join("album_tags at on at.album_id = aa.album_id").
		Where(And{Eq{"aa.role": "albumartist"}, Eq{"at.tag_id": tagIDs}}).ToSql()
	return Expr("artist.id IN ("+sub+")", args...)
}

func genreFilter(filter itemTagFilterDef) func(_ string, v any) Sqlizer {
	return func(_ string, v any) Sqlizer {
		return filter.ByID(v)
	}
}

// tagIDFilter matches rows whose tags JSON contains the tag id(s); indexed tags use the join table.
func tagIDFilter(name string, idValue any) Sqlizer {
	tagName := model.TagName(strings.TrimSuffix(name, "_id"))
	if IsIndexedTag(tagName) {
		return SongTags.ByID(idValue)
	}
	return Exists(
		fmt.Sprintf(`json_tree(tags, "$.%s")`, tagName),
		And{
			NotEq{"json_tree.atom": nil},
			Eq{"value": idValue},
		},
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
		"library_id": wrapFilter(tagLibraryIdFilter),
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
