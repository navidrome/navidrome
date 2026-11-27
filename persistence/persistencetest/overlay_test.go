package persistencetest_test

import (
	"context"
	"errors"
	"reflect"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var errStub = errors.New("stubbed album error")

type failingAlbumRepo struct{ model.AlbumRepository }

func (failingAlbumRepo) Get(context.Context, string) (*model.Album, error) { return nil, errStub }

type failingAlbums struct{ model.DataStore }

func (f failingAlbums) Album() model.AlbumRepository {
	return failingAlbumRepo{f.DataStore.Album()}
}

type txRunner func(ctx context.Context, ds model.DataStore, block func(tx model.DataStore) error) error

var txRunners = []TableEntry{
	Entry("WithTx", txRunner(func(_ context.Context, ds model.DataStore, block func(tx model.DataStore) error) error {
		return ds.WithTx(block)
	})),
	Entry("WithTxImmediate", txRunner(func(_ context.Context, ds model.DataStore, block func(tx model.DataStore) error) error {
		return ds.WithTxImmediate(block)
	})),
	Entry("WithTxRetry", txRunner(func(ctx context.Context, ds model.DataStore, block func(tx model.DataStore) error) error {
		return ds.WithTxRetry(ctx, func(_ context.Context, tx model.DataStore) error { return block(tx) })
	})),
	Entry("a nested WithTx", txRunner(func(_ context.Context, ds model.DataStore, block func(tx model.DataStore) error) error {
		return ds.WithTx(func(tx model.DataStore) error { return tx.WithTx(block) })
	})),
}

func albumGetErrInTx(ctx context.Context, ds model.DataStore, run txRunner) error {
	GinkgoHelper()
	var err error
	Expect(run(ctx, ds, func(tx model.DataStore) error {
		_, err = tx.Album().Get(ctx, "any")
		return nil
	})).To(Succeed())
	return err
}

var _ = Describe("Overlay", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistencetest.Overlay(persistence.New(db.Db()), func(ds model.DataStore) model.DataStore {
			return failingAlbums{ds}
		})
	})

	It("applies the stub outside a transaction", func() {
		_, err := ds.Album().Get(ctx, "any")
		Expect(err).To(MatchError(errStub))
	})

	DescribeTable("applies the stub inside a transaction", func(run txRunner) {
		Expect(albumGetErrInTx(ctx, ds, run)).To(MatchError(errStub))
	}, txRunners)
})

func repoAccessors() []reflect.Method {
	var accessors []reflect.Method
	for m := range reflect.TypeFor[model.DataStore]().Methods() {
		if m.Type.NumIn() == 0 && m.Type.NumOut() == 1 {
			accessors = append(accessors, m)
		}
	}
	return accessors
}

func callAccessor(ds model.DataStore, name string) any {
	return reflect.ValueOf(ds).MethodByName(name).Call(nil)[0].Interface()
}

var _ = Describe("Stub", func() {
	var ctx context.Context
	var realDS model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		realDS = persistence.New(db.Db())
	})

	It("has a wrapper for every repository accessor of model.DataStore", func() {
		repos := reflect.TypeFor[persistencetest.Repos]()
		for _, m := range repoAccessors() {
			_, ok := repos.FieldByName(m.Name)
			Expect(ok).To(BeTrue(), "Repos has no %s field", m.Name)
		}
	})

	DescribeTable("returns the wrapper's result from its accessor, given the real repository",
		func(name string) {
			var repos persistencetest.Repos
			field := reflect.ValueOf(&repos).Elem().FieldByName(name)
			var received any
			field.Set(reflect.MakeFunc(field.Type(), func(args []reflect.Value) []reflect.Value {
				received = args[0].Interface()
				return []reflect.Value{reflect.Zero(field.Type().Out(0))}
			}))

			Expect(callAccessor(persistencetest.Stub(realDS, repos), name)).To(BeNil())
			Expect(reflect.TypeOf(received)).To(Equal(reflect.TypeOf(callAccessor(realDS, name))))
		},
		func() []TableEntry {
			var entries []TableEntry
			for _, f := range reflect.VisibleFields(reflect.TypeFor[persistencetest.Repos]()) {
				entries = append(entries, Entry(f.Name, f.Name))
			}
			return entries
		}(),
	)

	It("returns the real repository from every accessor without a wrapper", func() {
		stub := persistencetest.Stub(realDS, persistencetest.Repos{})
		for _, m := range repoAccessors() {
			got := callAccessor(stub, m.Name)
			Expect(got).ToNot(BeNil(), m.Name)
			Expect(reflect.TypeOf(got)).To(Equal(reflect.TypeOf(callAccessor(realDS, m.Name))), m.Name)
		}
	})

	DescribeTable("applies the wrappers inside a transaction", func(run txRunner) {
		ds := persistencetest.Stub(realDS, persistencetest.Repos{
			Album: func(r model.AlbumRepository) model.AlbumRepository { return failingAlbumRepo{r} },
		})
		Expect(albumGetErrInTx(ctx, ds, run)).To(MatchError(errStub))
	}, txRunners)
})
