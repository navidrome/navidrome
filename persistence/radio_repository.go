package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type radioRepository struct {
	sqlRepository
}

func NewRadioRepository(db dbx.Builder) model.RadioRepository {
	r := &radioRepository{}
	r.db = db
	r.registerModel(&model.Radio{}, map[string]filterFunc{
		"name": containsFilter("name"),
	})
	return r
}

func (r *radioRepository) isPermitted(ctx context.Context) bool {
	user := loggedUser(ctx)
	return user.IsAdmin
}

func (r *radioRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect(ctx)
	return r.count(ctx, sql, options...)
}

// Exists needs no library or ownership filter: radios are visible to every user.
func (r *radioRepository) Exists(ctx context.Context, id string) (bool, error) {
	return r.exists(ctx, Eq{"id": id})
}

func (r *radioRepository) Delete(ctx context.Context, ids ...string) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}

	for _, id := range ids {
		if err := r.deleteByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *radioRepository) Get(ctx context.Context, id string) (*model.Radio, error) {
	sel := r.newSelect(ctx).Where(Eq{"id": id}).Columns("*")
	res := model.Radio{}
	err := r.queryOne(ctx, sel, &res)
	if err != nil {
		return &res, err
	}
	list := model.Radios{res}
	r.hydrateArtwork(ctx, list)
	return &list[0], nil
}

func (r *radioRepository) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Radios, error) {
	sel := r.newSelect(ctx, options...).Columns("*")
	res := model.Radios{}
	err := r.queryAll(ctx, sel, &res)
	if err != nil {
		return res, err
	}
	r.hydrateArtwork(ctx, res)
	return res, nil
}

// hydrateArtwork fills each radio's ImageHash/ImageAbsent from one batched item_artwork lookup.
func (r *radioRepository) hydrateArtwork(ctx context.Context, radios model.Radios) {
	hydrateItems(ctx, r.db, model.KindRadioArtwork, radios,
		func(rd *model.Radio) (string, *model.ItemImage) { return rd.ID, &rd.ItemImage })
}

func (r *radioRepository) Put(ctx context.Context, radio *model.Radio, colsToUpdate ...string) error {
	if !r.isPermitted(ctx) {
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
	_, err := r.put(ctx, radio.ID, radio, colsToUpdate...)
	if err != nil {
		return err
	}
	// Enqueue artwork resolution for the created/updated radio at Bump priority so a new
	// radio's cover resolves proactively. Never fails the save.
	item := model.ArtworkQueueItem{ItemKind: model.KindRadioArtwork.Prefix(), ItemID: radio.ID, ImageType: model.ImageTypePrimary,
		Priority: model.ArtworkPriorityBump}
	if err := NewArtworkQueueRepository(ctx, r.db).Enqueue(item); err != nil {
		log.Warn(ctx, "could not enqueue radio artwork", "id", radio.ID, err)
	}
	return nil
}

func (r *radioRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *radioRepository) Read(ctx context.Context, id string) (*model.Radio, error) {
	return r.Get(ctx, id)
}

func (r *radioRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Radio, error) {
	return r.GetAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *radioRepository) Save(ctx context.Context, t *model.Radio) (string, error) {
	if !r.isPermitted(ctx) {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(ctx, t)
	return t.ID, err
}

func (r *radioRepository) Update(ctx context.Context, id string, entity model.Radio, cols ...string) error {
	t := &entity
	t.ID = id
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	return r.Put(ctx, t, cols...)
}

var _ model.RadioRepository = (*radioRepository)(nil)
var _ rest.Repository[model.Radio] = (*radioRepository)(nil)
var _ rest.Persistable[model.Radio] = (*radioRepository)(nil)
