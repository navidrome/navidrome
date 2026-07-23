package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type radioRepository struct {
	sqlRepository
}

func NewRadioRepository(ctx context.Context, db dbx.Builder) model.RadioRepository {
	r := &radioRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Radio{}, map[string]filterFunc{
		"name": containsFilter("name"),
	})
	return r
}

func (r *radioRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *radioRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect()
	return r.count(sql, options...)
}

func (r *radioRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	return r.delete(Eq{"id": id})
}

func (r *radioRepository) Get(id string) (*model.Radio, error) {
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.Radio{}
	err := r.queryOne(sel, &res)
	if err != nil {
		return &res, err
	}
	list := model.Radios{res}
	r.hydrateArtwork(list)
	return &list[0], nil
}

func (r *radioRepository) GetAll(options ...model.QueryOptions) (model.Radios, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.Radios{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return res, err
	}
	r.hydrateArtwork(res)
	return res, nil
}

// hydrateArtwork fills each radio's ImageHash/ImageAbsent from one batched item_artwork lookup.
func (r *radioRepository) hydrateArtwork(radios model.Radios) {
	if len(radios) == 0 {
		return
	}
	ids := slice.Map(radios, func(rd model.Radio) string { return rd.ID })
	infos := hydrateItemImages(r.ctx, r.db, model.KindRadioArtwork.Prefix(), ids)
	for i := range radios {
		applyItemImage(infos, radios[i].ID, &radios[i].ItemImage)
	}
}

// GetAllIDs returns just the radio IDs. Used by bulk enumeration (artwork backfill).
func (r *radioRepository) GetAllIDs(options ...model.QueryOptions) ([]string, error) {
	sel := r.newSelect(options...).Columns("id")
	ids := []string{}
	err := r.queryAllSlice(sel, &ids)
	return ids, err
}

func (r *radioRepository) Put(radio *model.Radio, colsToUpdate ...string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	radio.UpdatedAt = time.Now()
	if radio.ID == "" {
		radio.CreatedAt = time.Now()
		radio.ID = id.NewRandom()
	}
	if len(colsToUpdate) > 0 {
		colsToUpdate = append(colsToUpdate, "UpdatedAt")
	}
	_, err := r.put(radio.ID, radio, colsToUpdate...)
	if err != nil {
		return err
	}
	// Enqueue artwork resolution for the created/updated radio at Bump priority so a new
	// radio's cover resolves proactively. Never fails the save.
	item := model.ArtworkQueueItem{ItemKind: "ra", ItemID: radio.ID, ImageType: model.ImageTypePrimary,
		Priority: model.ArtworkPriorityBump}
	if err := NewArtworkQueueRepository(r.ctx, r.db).Enqueue(item); err != nil {
		log.Warn(r.ctx, "could not enqueue radio artwork", "id", radio.ID, err)
	}
	return nil
}

func (r *radioRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *radioRepository) EntityName() string {
	return "radio"
}

func (r *radioRepository) NewInstance() any {
	return &model.Radio{}
}

func (r *radioRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *radioRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *radioRepository) Save(entity any) (string, error) {
	t := entity.(*model.Radio)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *radioRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.Radio)
	t.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.RadioRepository = (*radioRepository)(nil)
var _ rest.Repository = (*radioRepository)(nil)
var _ rest.Persistable = (*radioRepository)(nil)
