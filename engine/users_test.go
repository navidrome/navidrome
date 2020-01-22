package engine

import (
	"context"

	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users", func() {
	Describe("Authenticate", func() {
		var users Users
		BeforeEach(func() {
			ds := &persistence.MockDataStore{}
			users = NewUsers(ds)
		})

		Context("Plaintext password", func() {
			It("authenticates with plaintext password ", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "wordpass", "", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})

			It("fails authentication with wrong password", func() {
				_, err := users.Authenticate(context.TODO(), "admin", "INVALID", "", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("Encoded password", func() {
			It("authenticates with simple encoded password ", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "enc:776f726470617373", "", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})
		})

		Context("Token based authentication", func() {
			It("authenticates with token based authentication", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "", "23b342970e25c7928831c3317edd0b67", "retnlmjetrymazgkt")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})

			It("fails if salt is missing", func() {
				_, err := users.Authenticate(context.TODO(), "admin", "", "23b342970e25c7928831c3317edd0b67", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})
	})
})
