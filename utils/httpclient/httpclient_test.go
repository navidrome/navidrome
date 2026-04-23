package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Client Suite")
}

var _ = Describe("Transport", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Proxy.URL = ""
	})

	It("memoizes the shared transport for the same proxy URL", func() {
		first := Transport()
		second := Transport()

		Expect(second).To(BeIdenticalTo(first))
	})

	It("does not configure a proxy when Proxy.URL is empty", func() {
		Expect(Transport().Proxy).To(BeNil())
	})

	It("returns the configured proxy URL for requests", func() {
		conf.Server.Proxy.URL = "socks5://user:pass@proxy.example.com:1080"

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		proxyURL, err := Transport().Proxy(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(proxyURL.String()).To(Equal("socks5://user:pass@proxy.example.com:1080"))
		Expect(proxyURL.User.Username()).To(Equal("user"))
		password, ok := proxyURL.User.Password()
		Expect(ok).To(BeTrue())
		Expect(password).To(Equal("pass"))
	})
})

var _ = Describe("New", func() {
	var (
		proxy *httptest.Server
		rec   *proxyRecorder
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Proxy.URL = ""
		proxy, rec = newProxyServer()
		DeferCleanup(proxy.Close)
	})

	It("routes outbound HTTP requests through the configured proxy", func() {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "proxied-http")
		}))
		DeferCleanup(backend.Close)

		conf.Server.Proxy.URL = proxy.URL
		client := NewWithTimeout(2 * time.Second)

		resp, err := client.Get(backend.URL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal("proxied-http"))
		Expect(rec.absoluteRequests()).To(ContainElement(backend.URL + "/"))
	})

	It("routes outbound HTTPS requests through the configured proxy", func() {
		backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "proxied-https")
		}))
		DeferCleanup(backend.Close)

		conf.Server.Proxy.URL = proxy.URL
		transport := Transport().Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client := &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		}

		resp, err := client.Get(backend.URL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal("proxied-https"))
		Expect(rec.connectRequests()).To(ContainElement(backend.Listener.Addr().String()))
	})
})

var _ = Describe("NewWithTimeout", func() {
	It("uses the default outbound timeout in New", func() {
		Expect(New().Timeout).To(Equal(consts.DefaultHttpClientTimeOut))
	})

	It("allows overriding the default timeout", func() {
		Expect(NewWithTimeout(5 * time.Second).Timeout).To(Equal(5 * time.Second))
	})
})

type proxyRecorder struct {
	mu       sync.Mutex
	absolute []string
	connects []string
}

func (p *proxyRecorder) addAbsolute(rawURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.absolute = append(p.absolute, rawURL)
}

func (p *proxyRecorder) addConnect(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connects = append(p.connects, host)
}

func (p *proxyRecorder) absoluteRequests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.absolute))
	copy(out, p.absolute)
	return out
}

func (p *proxyRecorder) connectRequests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.connects))
	copy(out, p.connects)
	return out
}

func newProxyServer() (*httptest.Server, *proxyRecorder) {
	recorder := &proxyRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			recorder.addConnect(r.Host)
			handleConnect(w, r)
			return
		}

		recorder.addAbsolute(r.URL.String())

		req := r.Clone(r.Context())
		req.RequestURI = ""

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil

		resp, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	return server, recorder
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	targetConn, err := net.DialTimeout("tcp", r.Host, 2*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	_, _ = fmt.Fprint(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	go proxyCopy(targetConn, clientConn)
	go proxyCopy(clientConn, targetConn)
}

func proxyCopy(dst net.Conn, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	_, _ = io.Copy(dst, src)
}
