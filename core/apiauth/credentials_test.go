package apiauth

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeChecker struct {
	res CredentialResult
	err error
	hit bool
}

func (f *fakeChecker) Check(context.Context, string, string) (CredentialResult, error) {
	f.hit = true
	return f.res, f.err
}

var _ = Describe("credential chain", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("authenticates against the database with the stored password", func() {
		u := createUser(ctx, "pw", false)
		res, err := checkCredentials(ctx, []CredentialChecker{dbChecker{ds: realDS}}, u.UserName, "pw")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Outcome).To(Equal(Authenticated))
		Expect(res.User.ID).To(Equal(u.ID))
		Expect(res.Provider).To(Equal("password"))
		Expect(res.PasswordLocal).To(BeTrue())
	})

	It("rejects a wrong password and an unknown user the same way", func() {
		u := createUser(ctx, "pw", false)
		_, err := checkCredentials(ctx, []CredentialChecker{dbChecker{ds: realDS}}, u.UserName, "nope")
		Expect(err).To(MatchError(model.ErrInvalidAuth))
		_, err = checkCredentials(ctx, []CredentialChecker{dbChecker{ds: realDS}}, "ghost", "pw")
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	It("moves on only from NotMine, and an owner's rejection stops the chain", func() {
		owner := &fakeChecker{res: CredentialResult{Outcome: Rejected}}
		later := &fakeChecker{res: CredentialResult{Outcome: Authenticated, User: &model.User{ID: "x"}}}
		_, err := checkCredentials(ctx, []CredentialChecker{&fakeChecker{res: CredentialResult{Outcome: NotMine}}, owner, later}, "a", "b")
		Expect(err).To(MatchError(model.ErrInvalidAuth))
		Expect(later.hit).To(BeFalse())
	})

	It("maps Unavailable to ErrNotAvailable and passes through checker errors", func() {
		_, err := checkCredentials(ctx, []CredentialChecker{&fakeChecker{res: CredentialResult{Outcome: Unavailable}}}, "a", "b")
		Expect(err).To(MatchError(model.ErrNotAvailable))
		boom := errors.New("boom")
		_, err = checkCredentials(ctx, []CredentialChecker{&fakeChecker{err: boom}}, "a", "b")
		Expect(err).To(MatchError(boom))
	})
})
