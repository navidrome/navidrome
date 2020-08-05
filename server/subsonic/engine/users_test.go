package engine

import (
	"context"

	"github.com/deluan/navidrome/core/auth"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
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
				usr, err := users.Authenticate(context.TODO(), "admin", "wordpass", "", "", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})

			It("fails authentication with wrong password", func() {
				_, err := users.Authenticate(context.TODO(), "admin", "INVALID", "", "", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("Encoded password", func() {
			It("authenticates with simple encoded password ", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "enc:776f726470617373", "", "", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})
		})

		Context("Token based authentication", func() {
			It("authenticates with token based authentication", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "", "23b342970e25c7928831c3317edd0b67", "retnlmjetrymazgkt", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})

			It("fails if salt is missing", func() {
				_, err := users.Authenticate(context.TODO(), "admin", "", "23b342970e25c7928831c3317edd0b67", "", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("JWT based authentication", func() {
			var validToken string
			BeforeEach(func() {
				u := &model.User{UserName: "admin"}
				var err error
				validToken, err = auth.CreateToken(u)
				if err != nil {
					panic(err)
				}
			})
			It("authenticates with JWT token based authentication", func() {
				usr, err := users.Authenticate(context.TODO(), "admin", "", "", "", validToken)

				Expect(err).NotTo(HaveOccurred())
				Expect(usr).To(Equal(&model.User{UserName: "admin", Password: "wordpass"}))
			})

			It("fails if JWT token is invalid", func() {
				_, err := users.Authenticate(context.TODO(), "admin", "", "", "", "invalid.token")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})

			It("fails if JWT token sub is different than username", func() {
				_, err := users.Authenticate(context.TODO(), "not_admin", "", "", "", validToken)
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})
	})
})
