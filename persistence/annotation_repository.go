package persistence

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
)

type annotation struct {
	AnnotationID string    `orm:"pk;column(ann_id)"`
	UserID       string    `orm:"column(user_id)"`
	ItemID       string    `orm:"column(item_id)"`
	ItemType     string    `orm:"column(item_type)"`
	PlayCount    int       `orm:"index;null"`
	PlayDate     time.Time `orm:"index;null"`
	Rating       int       `orm:"index;null"`
	Starred      bool      `orm:"index"`
	StarredAt    time.Time `orm:"null"`
}

func (u *annotation) TableUnique() [][]string {
	return [][]string{
		[]string{"UserID", "ItemID", "ItemType"},
	}
}

type annotationRepository struct {
	sqlRepository
}

func NewAnnotationRepository(o orm.Ormer) model.AnnotationRepository {
	r := &annotationRepository{}
	r.ormer = o
	r.tableName = "annotation"
	return r
}

func (r *annotationRepository) Get(userID, itemType string, itemID string) (*model.Annotation, error) {
	if userID == "" {
		return nil, model.ErrInvalidAuth
	}
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id", itemID)
	var ann annotation
	err := q.One(&ann)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	resp := model.Annotation(ann)
	return &resp, nil
}

func (r *annotationRepository) GetMap(userID, itemType string, itemID []string) (model.AnnotationMap, error) {
	if userID == "" {
		return nil, model.ErrInvalidAuth
	}
	if len(itemID) == 0 {
		return nil, nil
	}
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id__in", itemID)
	var res []annotation
	_, err := q.All(&res)
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
	if userID == "" {
		return nil, model.ErrInvalidAuth
	}
	q := r.newQuery(options...).Filter("user_id", userID).Filter("item_type", itemType)
	var res []annotation
	_, err := q.All(&res)
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
	if userID == "" {
		return model.ErrInvalidAuth
	}
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id", itemID)
	c, err := q.Update(orm.Params{
		"play_count": orm.ColValue(orm.ColAdd, 1),
		"play_date":  ts,
	})
	if c == 0 || err == orm.ErrNoRows {
		ann := r.new(userID, itemType, itemID)
		ann.PlayCount = 1
		ann.PlayDate = ts
		_, err = r.ormer.Insert(ann)
	}
	return err
}

func (r *annotationRepository) SetStar(starred bool, userID, itemType string, ids ...string) error {
	if userID == "" {
		return model.ErrInvalidAuth
	}
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id__in", ids)
	var starredAt time.Time
	if starred {
		starredAt = time.Now()
	}
	c, err := q.Update(orm.Params{
		"starred":    starred,
		"starred_at": starredAt,
	})
	if c == 0 || err == orm.ErrNoRows {
		for _, id := range ids {
			ann := r.new(userID, itemType, id)
			ann.Starred = starred
			ann.StarredAt = starredAt
			_, err = r.ormer.Insert(ann)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *annotationRepository) SetRating(rating int, userID, itemType string, itemID string) error {
	if userID == "" {
		return model.ErrInvalidAuth
	}
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id", itemID)
	c, err := q.Update(orm.Params{
		"rating": rating,
	})
	if c == 0 || err == orm.ErrNoRows {
		ann := r.new(userID, itemType, itemID)
		ann.Rating = rating
		_, err = r.ormer.Insert(ann)
	}
	return err

}

func (r *annotationRepository) Delete(userID, itemType string, itemID ...string) error {
	q := r.newQuery().Filter("user_id", userID).Filter("item_type", itemType).Filter("item_id__in", itemID)
	_, err := q.Delete()
	return err
}
