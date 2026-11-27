package quickconnect

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var device = Device{ID: "dev-1", Name: "Pixel 7", App: "Finamp", AppVersion: "1.0.0"}

var _ = Describe("QuickConnect", func() {
	var qc QuickConnect

	BeforeEach(func() {
		qc = New()
	})

	Describe("Initiate", func() {
		It("creates a pending request with a 6-digit code and a random secret", func() {
			req, err := qc.Initiate(device)
			Expect(err).ToNot(HaveOccurred())
			Expect(req.Code).To(MatchRegexp(`^[1-9]\d{5}$`))
			Expect(req.Secret).To(MatchRegexp(`^[0-9a-f]{64}$`))
			Expect(req.Device).To(Equal(device))
			Expect(req.DateAdded).ToNot(BeZero())
			Expect(req.Authorized()).To(BeFalse())
		})

		It("never gives two live requests the same code or secret", func() {
			codes := map[string]bool{}
			secrets := map[string]bool{}
			for range 500 {
				req, err := qc.Initiate(device)
				Expect(err).ToNot(HaveOccurred())
				codes[req.Code] = true
				secrets[req.Secret] = true
			}
			Expect(codes).To(HaveLen(500))
			Expect(secrets).To(HaveLen(500))
		})

		It("refuses new requests when too many are pending", func() {
			for range maxPending {
				_, err := qc.Initiate(device)
				Expect(err).ToNot(HaveOccurred())
			}
			_, err := qc.Initiate(device)
			Expect(err).To(MatchError(ErrTooManyRequests))
		})
	})

	Describe("Status", func() {
		It("returns the request for a known secret", func() {
			req, _ := qc.Initiate(device)
			got, err := qc.Status(req.Secret)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(req))
		})

		It("returns ErrNotFound for an unknown secret", func() {
			_, err := qc.Status("nope")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Lookup", func() {
		It("finds a request by code, ignoring spaces", func() {
			req, _ := qc.Initiate(device)
			got, err := qc.Lookup(req.Code[:3] + " " + req.Code[3:])
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(req))
		})

		It("returns ErrNotFound for an unknown code", func() {
			_, err := qc.Lookup("000000")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("refuses a request that is already authorized", func() {
			req, _ := qc.Initiate(device)
			_, _ = qc.Authorize(req.Code, "user-1")
			_, err := qc.Lookup(req.Code)
			Expect(err).To(MatchError(ErrAlreadyAuthorized))
		})
	})

	Describe("Authorize", func() {
		It("marks the request as authorized by the user", func() {
			req, _ := qc.Initiate(device)
			got, err := qc.Authorize(" "+req.Code+" ", "user-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Authorized()).To(BeTrue())
			Expect(got.UserID).To(Equal("user-1"))

			status, _ := qc.Status(req.Secret)
			Expect(status.Authorized()).To(BeTrue())
		})

		It("refuses a request that is already authorized", func() {
			req, _ := qc.Initiate(device)
			_, _ = qc.Authorize(req.Code, "user-1")
			_, err := qc.Authorize(req.Code, "user-2")
			Expect(err).To(MatchError(ErrAlreadyAuthorized))
		})

		It("returns ErrNotFound for an unknown code", func() {
			_, err := qc.Authorize("000000", "user-1")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Redeem", func() {
		It("returns the approving user once, then forgets the request", func() {
			req, _ := qc.Initiate(device)
			_, _ = qc.Authorize(req.Code, "user-1")

			userID, err := qc.Redeem(req.Secret)
			Expect(err).ToNot(HaveOccurred())
			Expect(userID).To(Equal("user-1"))

			_, err = qc.Redeem(req.Secret)
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = qc.Status(req.Secret)
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = qc.Lookup(req.Code)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("refuses a request that is not authorized yet", func() {
			req, _ := qc.Initiate(device)
			_, err := qc.Redeem(req.Secret)
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = qc.Status(req.Secret)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

// Plain tests: testing/synctest needs a *testing.T, which Ginkgo doesn't give.
func TestPendingRequestExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		qc := New()
		req, _ := qc.Initiate(device)

		time.Sleep(timeout - time.Second)
		_, err := qc.Status(req.Secret)
		g.Expect(err).ToNot(HaveOccurred())

		time.Sleep(2 * time.Second)
		_, err = qc.Status(req.Secret)
		g.Expect(err).To(MatchError(model.ErrNotFound))
		_, err = qc.Authorize(req.Code, "user-1")
		g.Expect(err).To(MatchError(model.ErrNotFound))
	})
}

func TestAuthorizeExtendsExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		qc := New()
		req, _ := qc.Initiate(device)

		time.Sleep(timeout - time.Second)
		_, err := qc.Authorize(req.Code, "user-1")
		g.Expect(err).ToNot(HaveOccurred())

		time.Sleep(timeout - time.Second)
		userID, err := qc.Redeem(req.Secret)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(userID).To(Equal("user-1"))
	})
}

func TestExpiredRequestsFreeCapacity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		qc := New()
		for range maxPending {
			_, _ = qc.Initiate(device)
		}
		time.Sleep(timeout + time.Second)
		_, err := qc.Initiate(device)
		g.Expect(err).ToNot(HaveOccurred())
	})
}
