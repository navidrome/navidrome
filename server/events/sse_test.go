package events

import (
	"context"

	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	var b broker

	BeforeEach(func() {
		b = broker{}
	})

	Describe("shouldSend", func() {
		var c client
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
			c = client{
				address:  "192.168.0.1:54639",
				username: "janedoe",
			}
		})
		Context("request has remote address", func() {
			It("sends message for same username, different address", func() {
				ctx = request.WithRemoteAddress(ctx, "192.168.0.2")
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
			It("does not send message for same username, same address", func() {
				ctx = request.WithRemoteAddress(ctx, "192.168.0.1")
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeFalse())
			})
			It("does not send message for different username", func() {
				ctx = request.WithRemoteAddress(ctx, "192.168.0.3")
				ctx = request.WithUsername(ctx, "johndoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeFalse())
			})
		})
		Context("request does not have remote address", func() {
			It("sends message for same username", func() {
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
			It("sends message for different username", func() {
				ctx = request.WithUsername(ctx, "johndoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
		})
	})
})
