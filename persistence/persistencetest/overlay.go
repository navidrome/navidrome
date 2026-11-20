package persistencetest

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type overlay struct {
	model.DataStore
	wrap func(model.DataStore) model.DataStore
}

// Overlay returns wrap(ds), and also wraps every transaction ds opens, so a stub that overrides
// one repository still applies inside WithTx. wrap's result must embed the DataStore it receives.
func Overlay(ds model.DataStore, wrap func(model.DataStore) model.DataStore) model.DataStore {
	return wrap(&overlay{DataStore: ds, wrap: wrap})
}

func (o *overlay) WithTx(block func(tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTx(func(tx model.DataStore) error {
		return block(Overlay(tx, o.wrap))
	}, scope...)
}

func (o *overlay) WithTxImmediate(block func(tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTxImmediate(func(tx model.DataStore) error {
		return block(Overlay(tx, o.wrap))
	}, scope...)
}

func (o *overlay) WithTxRetry(ctx context.Context, block func(ctx context.Context, tx model.DataStore) error, scope ...string) error {
	return o.DataStore.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
		return block(ctx, Overlay(tx, o.wrap))
	}, scope...)
}
