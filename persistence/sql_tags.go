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

func tagIDFilter(name string, idValue any) Sqlizer {
	name = strings.TrimSuffix(name, "_id")
	return Exists(
		fmt.Sprintf(`json_tree(tags, "$.%s")`, name),
		And{
			NotEq{"json_tree.atom": nil},
			Eq{"value": idValue},
		},
	)
}

// tagLibraryIdFilter filters tags based on library access through the library_tag table
func tagLibraryIdFilter(_ string, value interface{}) Sqlizer {
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

// newSelect overrides the base implementation to apply tag name filtering and library filtering.
func (r *baseTagRepository) newSelect(options ...model.QueryOptions) SelectBuilder {
	user := loggedUser(r.ctx)
	if user.ID == invalidUserId {
		// No user context - return empty result set
		return SelectBuilder{}.Where(Eq{"1": "0"})
	}
	sq := r.sqlRepository.newSelect(options...)
	if r.tagFilter != nil {
		sq = sq.Where(Eq{"tag.tag_name": *r.tagFilter})
	}
	sq = sq.Columns(
		"tag.id",
		"tag.tag_value as name",
		"COALESCE(SUM(library_tag.album_count), 0) as album_count",
		"COALESCE(SUM(library_tag.media_file_count), 0) as song_count",
	).
		LeftJoin("library_tag on library_tag.tag_id = tag.id").
		// Apply library filtering by joining only with accessible libraries
		Join("user_library on user_library.library_id = library_tag.library_id AND user_library.user_id = ?", user.ID).
		GroupBy("tag.id", "tag.tag_value")
	return sq
}

// ResourceRepository interface implementation

func (r *baseTagRepository) Count(options ...rest.QueryOptions) (int64, error) {
	// Create a query that counts distinct tags without GROUP BY
	user := loggedUser(r.ctx)
	if user.ID == invalidUserId {
		return 0, nil
	}

	// Build the same base query as newSelect but for counting
	sq := Select()
	if r.tagFilter != nil {
		sq = sq.Where(Eq{"tag.tag_name": *r.tagFilter})
	}

	// Apply the same joins as newSelect
	sq = sq.LeftJoin("library_tag on library_tag.tag_id = tag.id").
		Join("user_library on user_library.library_id = library_tag.library_id AND user_library.user_id = ?", user.ID)

	return r.count(sq, r.parseRestOptions(r.ctx, options...))
}

func (r *baseTagRepository) Read(id string) (interface{}, error) {
	query := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Tag
	err := r.queryOne(query, &res)
	return &res, err
}

func (r *baseTagRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	query := r.newSelect(r.parseRestOptions(r.ctx, options...)).Columns("*")
	var res model.TagList
	err := r.queryAll(query, &res)
	return res, err
}

func (r *baseTagRepository) EntityName() string {
	return "tag"
}

func (r *baseTagRepository) NewInstance() interface{} {
	return model.Tag{}
}

// Interface compliance check
var _ model.ResourceRepository = (*baseTagRepository)(nil)
