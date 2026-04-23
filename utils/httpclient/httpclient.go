package httpclient

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

var (
	mu          sync.Mutex
	cachedURL   string
	cachedValue *http.Transport
)

func Transport() *http.Transport {
	mu.Lock()
	defer mu.Unlock()

	currentURL := conf.Server.Proxy.URL
	if cachedValue != nil && cachedURL == currentURL {
		return cachedValue
	}
	if cachedValue != nil {
		cachedValue.CloseIdleConnections()
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil

	if currentURL != "" {
		proxyURL, err := url.Parse(currentURL)
		if err != nil {
			// Config validation should prevent this, but keep runtime behavior safe.
			log.Error("Invalid Proxy.URL configured for outbound HTTP transport", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	cachedURL = currentURL
	cachedValue = transport
	return cachedValue
}

func New() *http.Client {
	return NewWithTimeout(consts.DefaultHttpClientTimeOut)
}

func NewWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: Transport(),
	}
}
