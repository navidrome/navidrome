package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type mediaFileTagRepository struct {
	sqlRepository
}

func NewMediaFileTagRepository(ctx context.Context, db dbx.Builder) model.MediaFileTagRepository {
	r := &mediaFileTagRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "media_file_tag"
	return r
}

func (r *mediaFileTagRepository) TagSong(mediaFileID, tagName, source string) error {
	userID := loggedUser(r.ctx).ID
	cond := And{
		Eq{"user_id": userID},
		Eq{"media_file_id": mediaFileID},
		Eq{"tag_name": tagName},
	}
	exists, err := r.exists(cond)
	if err != nil || exists {
		return err
	}
	ins := Insert(r.tableName).
		Columns("user_id", "media_file_id", "tag_name", "source", "created_at").
		Values(userID, mediaFileID, tagName, source, time.Now())
	_, err = r.executeSQL(ins)
	return err
}

func (r *mediaFileTagRepository) UntagSong(mediaFileID, tagName string) error {
	userID := loggedUser(r.ctx).ID
	return r.delete(And{
		Eq{"user_id": userID},
		Eq{"media_file_id": mediaFileID},
		Eq{"tag_name": tagName},
	})
}

// bySourceIfSet adds a "source" equality condition when source is non-empty,
// leaving the base condition untouched otherwise - "" means "any source".
func bySourceIfSet(cond And, source string) And {
	if source == "" {
		return cond
	}
	return append(cond, Eq{"source": source})
}

// hasAggregateTagPermission gates the aggregate/browse operations (TagCounts, AllTagNames,
// SongIDsForTag) behind the admin-controlled ai_tags/my_tags features - these back the AI
// Genre/AI Mood/My Tags dashboards (and the Subsonic getAllUserTags/getAllMyTags family, the same
// capability via a different client). It deliberately does NOT gate TagSong/UntagSong/TagsForSong:
// those are single-song reads/writes used by the "Edit Tags" dialog and by the AI Auto-Tagging
// plugin's own "already tagged, skip it" idempotency check, which must keep working regardless of
// whether some other user's dashboard access is restricted. An empty/unrecognized source (no
// dashboard reads with one today) is never gated.
func (r *mediaFileTagRepository) hasAggregateTagPermission(source string) bool {
	switch source {
	case model.MediaFileTagSourceAI:
		return r.hasFeaturePermission(model.FeatureAITags)
	case model.MediaFileTagSourceUser:
		return r.hasFeaturePermission(model.FeatureMyTags)
	default:
		return true
	}
}

func (r *mediaFileTagRepository) TagsForSong(mediaFileID, source string) ([]string, error) {
	userID := loggedUser(r.ctx).ID
	cond := bySourceIfSet(And{Eq{"user_id": userID}, Eq{"media_file_id": mediaFileID}}, source)
	sel := r.newSelect().Columns("tag_name").
		Where(cond).
		OrderBy("tag_name")
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileTagRepository) AllTagNames(source string) ([]string, error) {
	if !r.hasAggregateTagPermission(source) {
		return []string{}, nil
	}
	userID := loggedUser(r.ctx).ID
	cond := bySourceIfSet(And{Eq{"user_id": userID}}, source)
	sel := r.newSelect().Distinct().Columns("tag_name").
		Where(cond).
		OrderBy("tag_name")
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileTagRepository) SongIDsForTag(tagName, source string) ([]string, error) {
	if !r.hasAggregateTagPermission(source) {
		return []string{}, nil
	}
	userID := loggedUser(r.ctx).ID
	cond := bySourceIfSet(And{Eq{"user_id": userID}, Eq{"tag_name": tagName}}, source)
	sel := r.newSelect().Columns("media_file_id").
		Where(cond)
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileTagRepository) TagCounts(source string) ([]model.TagCount, error) {
	if !r.hasAggregateTagPermission(source) {
		return []model.TagCount{}, nil
	}
	userID := loggedUser(r.ctx).ID
	cond := bySourceIfSet(And{Eq{"user_id": userID}}, source)
	sel := r.newSelect().
		Columns("tag_name", "count(distinct media_file_id) as count").
		Where(cond).
		GroupBy("tag_name").
		OrderBy("tag_name")
	var res []model.TagCount
	err := r.queryAll(sel, &res)
	return res, err
}

var _ model.MediaFileTagRepository = (*mediaFileTagRepository)(nil)
