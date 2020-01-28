package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
)

type annotation struct {
	AnnotationID string    `orm:"pk;column(ann_id)"`
	UserID       string    `orm:"column(user_id)"`
	ItemID       string    `orm:"column(item_id)"`
	ItemType     string    `orm:"column(item_type)"`
	PlayCount    int       `orm:"column(play_count);index;null"`
	PlayDate     time.Time `orm:"column(play_date);index;null"`
	Rating       int       `orm:"null"`
	Starred      bool      `orm:"index"`
	StarredAt    time.Time `orm:"column(starred_at);null"`
}

func (u *annotation) TableUnique() [][]string {
	return [][]string{
		{"UserID", "ItemID", "ItemType"},
	}
}

type annotationRepository struct {
	sqlRepository
}

func NewAnnotationRepository(ctx context.Context, o orm.Ormer) model.AnnotationRepository {
	r := &annotationRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "annotation"
	return r
}

func (r *annotationRepository) Get(userID, itemType string, itemID string) (*model.Annotation, error) {
	q := Select("*").From(r.tableName).Where(And{
		Eq{"user_id": userId(r.ctx)},
		Eq{"item_type": itemType},
		Eq{"item_id": itemID},
	})
	var ann annotation
	err := r.queryOne(q, &ann)
	if err == model.ErrNotFound {
		return nil, nil
	}
	resp := model.Annotation(ann)
	return &resp, nil
}

func (r *annotationRepository) GetMap(userID, itemType string, itemIDs []string) (model.AnnotationMap, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	q := Select("*").From(r.tableName).Where(And{
		Eq{"user_id": userId(r.ctx)},
		Eq{"item_type": itemType},
		Eq{"item_id": itemIDs},
	})
	var res []annotation
	err := r.queryAll(q, &res)
	if err != nil {
		return nil, err
	}

	m := make(model.AnnotationMap)
	for _, a := range res {
		m[a.ItemID] = model.Annotation(a)
	}
	return m, nil
}

func (r *annotationRepository) GetAll(userID, itemType string, options ...model.QueryOptions) ([]model.Annotation, error) {
	q := Select("*").From(r.tableName).Where(And{
		Eq{"user_id": userId(r.ctx)},
		Eq{"item_type": itemType},
	})
	var res []annotation
	err := r.queryAll(q, &res)
	if err != nil {
		return nil, err
	}

	all := make([]model.Annotation, len(res))
	for i, a := range res {
		all[i] = model.Annotation(a)
	}
	return all, err
}

func (r *annotationRepository) new(userID, itemType string, itemID string) *annotation {
	id, _ := uuid.NewRandom()
	return &annotation{
		AnnotationID: id.String(),
		UserID:       userID,
		ItemID:       itemID,
		ItemType:     itemType,
	}
}

func (r *annotationRepository) IncPlayCount(userID, itemType string, itemID string, ts time.Time) error {
	uid := userId(r.ctx)
	q := Update(r.tableName).
		Set("play_count", Expr("play_count + 1")).
		Set("play_date", ts).
		Where(And{
			Eq{"user_id": uid},
			Eq{"item_type": itemType},
			Eq{"item_id": itemID},
		})
	c, err := r.executeSQL(q)
	if c == 0 || err == orm.ErrNoRows {
		ann := r.new(uid, itemType, itemID)
		ann.PlayCount = 1
		ann.PlayDate = ts
		_, err = r.ormer.Insert(ann)
	}
	return err
}

func (r *annotationRepository) SetStar(starred bool, userID, itemType string, ids ...string) error {
	uid := userId(r.ctx)
	var starredAt time.Time
	if starred {
		starredAt = time.Now()
	}
	q := Update(r.tableName).
		Set("starred", starred).
		Set("starred_at", starredAt).
		Where(And{
			Eq{"user_id": uid},
			Eq{"item_type": itemType},
			Eq{"item_id": ids},
		})
	c, err := r.executeSQL(q)
	if c == 0 || err == orm.ErrNoRows {
		for _, id := range ids {
			ann := r.new(uid, itemType, id)
			ann.Starred = starred
			ann.StarredAt = starredAt
			_, err = r.ormer.Insert(ann)
			if err != nil {
				if err.Error() != "LastInsertId is not supported by this driver" {
					return err
				}
			}
		}
	}
	return nil
}

func (r *annotationRepository) SetRating(rating int, userID, itemType string, itemID string) error {
	uid := userId(r.ctx)
	q := Update(r.tableName).
		Set("rating", rating).
		Where(And{
			Eq{"user_id": uid},
			Eq{"item_type": itemType},
			Eq{"item_id": itemID},
		})
	c, err := r.executeSQL(q)
	if c == 0 || err == orm.ErrNoRows {
		ann := r.new(uid, itemType, itemID)
		ann.Rating = rating
		_, err = r.ormer.Insert(ann)
	}
	return err
}

func (r *annotationRepository) Delete(userID, itemType string, ids ...string) error {
	return r.delete(And{
		Eq{"user_id": userId(r.ctx)},
		Eq{"item_type": itemType},
		Eq{"item_id": ids},
	})
}
