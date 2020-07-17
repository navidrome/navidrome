package scanner

import (
	"context"
	"fmt"

	"github.com/deluan/navidrome/log"
)

const (
	// batchSize used for albums/artists updates
	batchSize = 5
)

type refreshCallbackFunc = func(ids ...string) error

type flushableMap struct {
	ctx       context.Context
	flushFunc refreshCallbackFunc
	entity    string
	m         map[string]struct{}
}

func newFlushableMap(ctx context.Context, entity string, flushFunc refreshCallbackFunc) *flushableMap {
	return &flushableMap{
		ctx:       ctx,
		flushFunc: flushFunc,
		entity:    entity,
		m:         map[string]struct{}{},
	}
}

func (f *flushableMap) update(id string) error {
	f.m[id] = struct{}{}
	if len(f.m) >= batchSize {
		err := f.flush()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *flushableMap) flush() error {
	if len(f.m) == 0 {
		return nil
	}
	var ids []string
	for id := range f.m {
		ids = append(ids, id)
		delete(f.m, id)
	}
	if err := f.flushFunc(ids...); err != nil {
		log.Error(f.ctx, fmt.Sprintf("Error writing %ss to the DB", f.entity), err)
		return err
	}
	return nil
}
