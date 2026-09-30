package nativeapi

import (
	"context"
	"errors"
	"maps"
	"net/http"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

type missingRepository struct {
	rest.Repository[model.MediaFile]
	mfRepo model.MediaFileRepository
}

func newMissingRepository(ds model.DataStore) rest.Repository[model.MediaFile] {
	mf := ds.MediaFile()
	return &missingRepository{Repository: mf, mfRepo: mf}
}

func (r *missingRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.Repository.Count(ctx, r.parseOptions(options))
}

func (r *missingRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.MediaFile, error) {
	return r.Repository.ReadAll(ctx, r.parseOptions(options))
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

func (r *missingRepository) Read(ctx context.Context, id string) (*model.MediaFile, error) {
	mf, err := r.mfRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !mf.Missing {
		return nil, model.ErrNotFound
	}
	return mf, nil
}

func deleteMissingFiles(maintenance core.Maintenance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		p := req.Params(r)
		ids := p.Strings("id")

		var err error
		if len(ids) == 0 {
			err = maintenance.DeleteAllMissingFiles(ctx)
		} else {
			err = maintenance.DeleteMissingFiles(ctx, ids)
		}

		if len(ids) == 1 && errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "Missing file not found", "id", ids[0])
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to delete missing files", http.StatusInternalServerError)
			return
		}

		writeDeleteManyResponse(w, r, ids)
	}
}
