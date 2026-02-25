package httpclient_test

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/utils/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	When("EnablePostQuantumTLS is false (default)", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnablePostQuantumTLS = false
		})

		It("returns a client with classic curve preferences", func() {
			client := httpclient.New(10 * time.Second)
			Expect(client).ToNot(BeNil())
			Expect(client.Timeout).To(Equal(10 * time.Second))

			transport, ok := client.Transport.(*http.Transport)
			Expect(ok).To(BeTrue())
			Expect(transport.TLSClientConfig).ToNot(BeNil())
			Expect(transport.TLSClientConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
			Expect(transport.TLSClientConfig.CurvePreferences).To(Equal([]tls.CurveID{
				tls.X25519, tls.CurveP256, tls.CurveP384,
			}))
		})

		It("does not modify http.DefaultTransport", func() {
			_ = httpclient.New(5 * time.Second)

			defaultTransport := http.DefaultTransport.(*http.Transport)
			if defaultTransport.TLSClientConfig != nil {
				Expect(defaultTransport.TLSClientConfig.CurvePreferences).To(BeNil())
			}
		})
	})

	When("EnablePostQuantumTLS is true", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnablePostQuantumTLS = true
		})

		It("returns a client with default transport (no custom TLS config)", func() {
			client := httpclient.New(10 * time.Second)
			Expect(client).ToNot(BeNil())
			Expect(client.Timeout).To(Equal(10 * time.Second))
			Expect(client.Transport).To(BeNil())
		})
	})

	It("respects the timeout parameter", func() {
		DeferCleanup(configtest.SetupConfig())
		client := httpclient.New(30 * time.Second)
		Expect(client.Timeout).To(Equal(30 * time.Second))
	})

	It("works with zero timeout", func() {
		DeferCleanup(configtest.SetupConfig())
		client := httpclient.New(0)
		Expect(client.Timeout).To(Equal(time.Duration(0)))
	})
})
