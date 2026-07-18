package model_test

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetEntityByID", func() {
	var ds *tests.MockDataStore
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = GinkgoT().Context()
	})

	It("returns the entity matching the id", func() {
		ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
		entity, err := model.GetEntityByID(ctx, ds, "a1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Album{}))
		Expect(entity.(*model.Album).ID).To(Equal("a1"))
	})

	It("returns ErrNotFound when no entity matches", func() {
		_, err := model.GetEntityByID(ctx, ds, "missing")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("propagates unexpected repository errors instead of reporting not-found", func() {
		ds.Album(ctx).(*tests.MockAlbumRepo).SetError(true)
		_, err := model.GetEntityByID(ctx, ds, "a1")
		Expect(err).To(HaveOccurred())
		Expect(err).ToNot(MatchError(model.ErrNotFound))
	})
})
