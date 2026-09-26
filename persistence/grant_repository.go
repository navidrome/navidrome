package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type grantRepository struct {
	sqlRepository
}

func NewGrantRepository(db dbx.Builder) model.GrantRepository {
	r := &grantRepository{}
	r.db = db
	r.tableName = "api_grant"
	return r
}

const grantLastActivity = "COALESCE(last_used_at, created_at)"

func (r *grantRepository) Put(ctx context.Context, g *model.Grant) error {
	if g.ID == "" {
		g.ID = id.NewRandom()
	}
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now()
	}
	// Stored as UTC: SQLite compares these timestamps as strings.
	g.CreatedAt = g.CreatedAt.UTC()
	if g.LastUsedAt != nil {
		t := g.LastUsedAt.UTC()
		g.LastUsedAt = &t
	}
	values, err := toSQLArgs(*g)
	if err != nil {
		return err
	}
	_, err = r.executeSQL(ctx, Insert(r.tableName).SetMap(values))
	return err
}

func (r *grantRepository) Get(ctx context.Context, id string) (*model.Grant, error) {
	return r.findOne(ctx, Eq{"id": id})
}

func (r *grantRepository) FindBySecretHash(ctx context.Context, hash string) (*model.Grant, error) {
	return r.findOne(ctx, Eq{"secret_hash": hash})
}

func (r *grantRepository) findOne(ctx context.Context, cond Sqlizer) (*model.Grant, error) {
	var g model.Grant
	if err := r.queryOne(ctx, r.newSelect(ctx).Columns("*").Where(cond), &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// activeForUser skips grants left on an older epoch: they are dead but only deleted when presented.
func activeForUser(userID string, epoch int, idleSince time.Time) Sqlizer {
	return And{Eq{"user_id": userID, "user_epoch": epoch}, Expr(grantLastActivity+" >= ?", idleSince.UTC())}
}

func (r *grantRepository) GetAllForUser(ctx context.Context, userID string, epoch int, idleSince time.Time, offset, limit int) (model.Grants, error) {
	sel := r.newSelect(ctx).Columns("*").Where(activeForUser(userID, epoch, idleSince)).
		OrderBy("last_used_at IS NULL", "last_used_at desc", "created_at desc", "id").
		Offset(uint64(offset)).Limit(uint64(limit))
	var res model.Grants
	err := r.queryAll(ctx, sel, &res)
	return res, err
}

func (r *grantRepository) CountForUser(ctx context.Context, userID string, epoch int, idleSince time.Time) (int64, error) {
	return r.count(ctx, Select().Where(activeForUser(userID, epoch, idleSince)))
}

func (r *grantRepository) Delete(ctx context.Context, id string) error {
	return r.delete(ctx, Eq{"id": id})
}

func (r *grantRepository) DeleteForUser(ctx context.Context, userID, id string) error {
	n, err := r.executeSQL(ctx, Delete(r.tableName).Where(Eq{"id": id, "user_id": userID}))
	if err != nil {
		return err
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *grantRepository) DeleteOtherEpochs(ctx context.Context, userID string, epoch int) error {
	return r.delete(ctx, And{Eq{"user_id": userID}, NotEq{"user_epoch": epoch}})
}

// SetEpoch only moves grants still on fromEpoch, so grants killed by an earlier change never come back.
func (r *grantRepository) SetEpoch(ctx context.Context, userID string, fromEpoch, toEpoch int, onlyID string) error {
	cond := Eq{"user_id": userID, "user_epoch": fromEpoch}
	if onlyID != "" {
		cond["id"] = onlyID
	}
	_, err := r.executeSQL(ctx, Update(r.tableName).Set("user_epoch", toEpoch).Where(cond))
	return err
}

func (r *grantRepository) DeleteIfEpoch(ctx context.Context, id string, epoch int) error {
	return r.delete(ctx, Eq{"id": id, "user_epoch": epoch})
}

func (r *grantRepository) Touch(ctx context.Context, id, ip string, at, notSince time.Time) error {
	upd := Update(r.tableName).Set("last_used_at", at.UTC()).Set("last_used_ip", ip).
		Where(And{Eq{"id": id}, Or{Eq{"last_used_at": nil}, Lt{"last_used_at": notSince.UTC()}}})
	_, err := r.executeSQL(ctx, upd)
	return err
}

func (r *grantRepository) DeleteIdle(ctx context.Context, idleSince time.Time) (int64, error) {
	return r.executeSQL(ctx, Delete(r.tableName).Where(Expr(grantLastActivity+" < ?", idleSince.UTC())))
}
