package server

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// failingPutUserRepo counts users normally but fails to store them, to exercise the
// error path of the initial user creation.
type failingPutUserRepo struct {
	model.UserRepository
	err error
}

func (r *failingPutUserRepo) Put(*model.User) error { return r.err }

func dsWithFailingPut(err error) model.DataStore {
	return &tests.MockDataStore{MockedUser: &failingPutUserRepo{UserRepository: tests.CreateMockUserRepo(), err: err}}
}

var _ = Describe("initial_setup", func() {
	var ds model.DataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
	})

	Describe("createInitialAdminUser", func() {
		It("creates a new admin user with specified password if User table is empty", func() {
			Expect(createInitialAdminUser(ds, "pass123")).To(BeNil())
			ur := ds.User(context.TODO())
			admin, err := ur.FindByUsername("admin")
			Expect(err).To(BeNil())
			Expect(admin.Password).To(Equal("pass123"))
		})

		It("does not create a new admin user if User table is not empty", func() {
			Expect(createInitialAdminUser(ds, "first")).To(BeNil())
			ur := ds.User(context.TODO())
			Expect(ur.CountAll()).To(Equal(int64(1)))
			Expect(createInitialAdminUser(ds, "second")).To(BeNil())
			Expect(ur.CountAll()).To(Equal(int64(1)))
		})

		// The error was assigned to a shadowed err and dropped, so the caller saw
		// success and went on to commit the "setup complete" flag.
		It("returns the error when the user cannot be stored", func() {
			boom := errors.New("db is down")
			Expect(createInitialAdminUser(dsWithFailingPut(boom), "pass123")).To(MatchError(boom))
		})
	})
})
