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

func (r *mediaFileTagRepository) TagSong(mediaFileID, tagName string) error {
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
		Columns("user_id", "media_file_id", "tag_name", "created_at").
		Values(userID, mediaFileID, tagName, time.Now())
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

func (r *mediaFileTagRepository) TagsForSong(mediaFileID string) ([]string, error) {
	userID := loggedUser(r.ctx).ID
	sel := r.newSelect().Columns("tag_name").
		Where(And{Eq{"user_id": userID}, Eq{"media_file_id": mediaFileID}}).
		OrderBy("tag_name")
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileTagRepository) AllTagNames() ([]string, error) {
	userID := loggedUser(r.ctx).ID
	sel := r.newSelect().Distinct().Columns("tag_name").
		Where(Eq{"user_id": userID}).
		OrderBy("tag_name")
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileTagRepository) SongIDsForTag(tagName string) ([]string, error) {
	userID := loggedUser(r.ctx).ID
	sel := r.newSelect().Columns("media_file_id").
		Where(And{Eq{"user_id": userID}, Eq{"tag_name": tagName}})
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

var _ model.MediaFileTagRepository = (*mediaFileTagRepository)(nil)
