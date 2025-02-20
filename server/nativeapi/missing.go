package nativeapi

import (
	"context"
	"errors"
	"maps"
	"net/http"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

type missingRepository struct {
	model.ResourceRepository
	mfRepo model.MediaFileRepository
}

func newMissingRepository(ds model.DataStore) rest.RepositoryConstructor {
	return func(ctx context.Context) rest.Repository {
		return &missingRepository{mfRepo: ds.MediaFile(ctx), ResourceRepository: ds.Resource(ctx, model.MediaFile{})}
	}
}

func (r *missingRepository) Count(options ...rest.QueryOptions) (int64, error) {
	opt := r.parseOptions(options)
	return r.ResourceRepository.Count(opt)
}

func (r *missingRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	opt := r.parseOptions(options)
	return r.ResourceRepository.ReadAll(opt)
}

func (r *missingRepository) parseOptions(options []rest.QueryOptions) rest.QueryOptions {
	var opt rest.QueryOptions
	if len(options) > 0 {
		opt = options[0]
		opt.Filters = maps.Clone(opt.Filters)
	}
	opt.Filters["missing"] = "true"
	return opt
}

func (r *missingRepository) Read(id string) (any, error) {
	all, err := r.mfRepo.GetAll(model.QueryOptions{Filters: squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"missing": true},
	}})
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, model.ErrNotFound
	}
	return all[0], nil
}

func (r *missingRepository) EntityName() string {
	return "missing_files"
}

func deleteMissingFiles(ds model.DataStore, w http.ResponseWriter, r *http.Request) {
	repo := ds.MediaFile(r.Context())
	p := req.Params(r)
	ids, _ := p.Strings("id")
	err := ds.WithTx(func(tx model.DataStore) error {
		return repo.DeleteMissing(ids)
	})
	if len(ids) == 1 && errors.Is(err, model.ErrNotFound) {
		log.Warn(r.Context(), "Missing file not found", "id", ids[0])
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error(r.Context(), "Error deleting missing tracks from DB", "ids", ids, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ds.GC(r.Context())
	if err != nil {
		log.Error(r.Context(), "Error running GC after deleting missing tracks", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeDeleteManyResponse(w, r, ids)
}

var _ model.ResourceRepository = &missingRepository{}
