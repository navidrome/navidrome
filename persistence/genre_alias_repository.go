package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type genreAliasRepository struct {
	sqlRepository
}

func NewGenreAliasRepository(ctx context.Context, db dbx.Builder) model.GenreAliasRepository {
	r := &genreAliasRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.GenreAlias{}, nil)
	return r
}

func (r *genreAliasRepository) isPermitted() bool {
	return loggedUser(r.ctx).IsAdmin
}

func (r *genreAliasRepository) Get(id string) (*model.GenreAlias, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.GenreAlias
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreAliasRepository) GetAll(options ...model.QueryOptions) (model.GenreAliases, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.GenreAliases{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *genreAliasRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.newSelect(), options...)
}

func (r *genreAliasRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.delete(Eq{"id": id})
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

// canonicalOf reports whether name is currently used as another row's alias_name, and if so,
// that row's canonical_name. Used both to flatten chains (resolve a requested canonical target
// through any existing alias) and to detect when a name being merged is itself already an alias.
func (r *genreAliasRepository) canonicalOf(name string) (canonical string, isAlias bool, err error) {
	sel := r.newSelect().Columns("canonical_name").Where(Eq{"alias_name": name})
	var res model.GenreAlias
	err = r.queryOne(sel, &res)
	if errors.Is(err, model.ErrNotFound) {
		return name, false, nil
	}
	if err != nil {
		return "", false, err
	}
	return res.CanonicalName, true, nil
}

// Put validates and upserts a genre alias. Two invariants keep the mapping always exactly one
// level deep, regardless of the order aliases are created in:
//   - Flatten: if the requested canonical target is itself currently an alias of something else,
//     resolve through to THAT row's canonical name, rather than creating a chain.
//   - Repoint: if the alias being created is currently used as some OTHER row's canonical target,
//     repoint those rows at the new canonical name too, since this name is now itself an alias.
func (r *genreAliasRepository) Put(a *model.GenreAlias) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	if a.AliasName == a.CanonicalName {
		return errors.New("a genre cannot be merged into itself")
	}

	if resolved, isAlias, err := r.canonicalOf(a.CanonicalName); err != nil {
		return err
	} else if isAlias {
		a.CanonicalName = resolved
	}
	if a.AliasName == a.CanonicalName {
		return errors.New("a genre cannot be merged into itself")
	}

	repoint := Update(r.tableName).
		Where(Eq{"canonical_name": a.AliasName}).
		Set("canonical_name", a.CanonicalName).
		Set("updated_at", time.Now())
	if _, err := r.executeSQL(repoint); err != nil {
		return err
	}

	a.UpdatedAt = time.Now()
	if a.ID == "" {
		a.CreatedAt = time.Now()
	}
	_, err := r.put(a.ID, a)
	return err
}

func (r *genreAliasRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(r.newSelect(), r.parseRestOptions(r.ctx, options...))
}

func (r *genreAliasRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *genreAliasRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *genreAliasRepository) EntityName() string {
	return "genreAlias"
}

func (r *genreAliasRepository) NewInstance() any {
	return &model.GenreAlias{}
}

func (r *genreAliasRepository) Save(entity any) (string, error) {
	a := entity.(*model.GenreAlias)
	if err := r.Put(a); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return "", rest.ErrNotFound
		}
		return "", err
	}
	return a.ID, nil
}

func (r *genreAliasRepository) Update(id string, entity any, _ ...string) error {
	a := entity.(*model.GenreAlias)
	a.ID = id
	err := r.Put(a)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.GenreAliasRepository = (*genreAliasRepository)(nil)
var _ rest.Repository = (*genreAliasRepository)(nil)
var _ rest.Persistable = (*genreAliasRepository)(nil)
