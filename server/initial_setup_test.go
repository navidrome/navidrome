package server

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingPutUserRepo struct {
	model.UserRepository
	err error
}

func (r *failingPutUserRepo) Put(context.Context, *model.User) error { return r.err }

func dsWithFailingPut(err error) model.DataStore {
	return &tests.MockDataStore{MockedUser: &failingPutUserRepo{UserRepository: tests.CreateMockUserRepo(), err: err}}
}

var _ = Describe("initial_setup", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = GinkgoT().Context()
	})

	Describe("createInitialAdminUser", func() {
		It("creates a new admin user with specified password if User table is empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "pass123")).To(BeNil())
			ur := ds.User()
			admin, err := ur.FindByUsername(ctx, "admin")
			Expect(err).To(BeNil())
			Expect(admin.Password).To(Equal("pass123"))
		})

		It("does not create a new admin user if User table is not empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "first")).To(BeNil())
			ur := ds.User()
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
			Expect(createInitialAdminUser(ctx, ds, "second")).To(BeNil())
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
		})

		It("returns the error when the user cannot be stored", func() {
			boom := errors.New("db is down")
			Expect(createInitialAdminUser(ctx, dsWithFailingPut(boom), "pass123")).To(MatchError(boom))
		})

		It("returns the error when the user table cannot be read", func() {
			boom := errors.New("db is down")
			ds = &tests.MockDataStore{MockedUser: &tests.MockedUserRepo{Error: boom}}
			Expect(createInitialAdminUser(ctx, ds, "pass123")).To(MatchError(boom))
		})
	})
})
