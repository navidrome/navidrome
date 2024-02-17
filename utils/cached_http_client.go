package utils

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v2"
	"github.com/navidrome/navidrome/log"
)

const cacheSizeLimit = 100

type CachedHTTPClient struct {
	cache *ttlcache.Cache
	hc    httpDoer
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type requestData struct {
	Method string
	Header http.Header
	URL    string
	Body   *string
}

func NewCachedHTTPClient(wrapped httpDoer, ttl time.Duration) *CachedHTTPClient {
	c := &CachedHTTPClient{hc: wrapped}
	c.cache = ttlcache.NewCache()
	c.cache.SetCacheSizeLimit(cacheSizeLimit)
	c.cache.SkipTTLExtensionOnHit(true)
	c.cache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		req, err := c.deserializeReq(key)
		if err != nil {
			return nil, 0, err
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		return c.serializeResponse(resp), ttl, nil
	})
	c.cache.SetNewItemCallback(func(key string, value interface{}) {
		log.Trace("New request cached", "req", key, "resp", value)
	})
	return c
}

func (c *CachedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := c.serializeReq(req)
	respStr, err := c.cache.Get(key)
	if err != nil {
		return nil, err
	}
	return c.deserializeResponse(req, respStr.(string))
}

func (c *CachedHTTPClient) serializeReq(req *http.Request) string {
	data := requestData{
		Method: req.Method,
		Header: req.Header,
		URL:    req.URL.String(),
	}
	if req.Body != nil {
		bodyData, _ := io.ReadAll(req.Body)
		bodyStr := base64.StdEncoding.EncodeToString(bodyData)
		data.Body = &bodyStr
	}
	j, _ := json.Marshal(&data)
	return string(j)
}

func (c *CachedHTTPClient) deserializeReq(reqStr string) (*http.Request, error) {
	var data requestData
	_ = json.Unmarshal([]byte(reqStr), &data)
	var body io.Reader
	if data.Body != nil {
		bodyStr, _ := base64.StdEncoding.DecodeString(*data.Body)
		body = strings.NewReader(string(bodyStr))
	}
	req, err := http.NewRequest(data.Method, data.URL, body)
	if err != nil {
		return nil, err
	}
	req.Header = data.Header
	return req, nil
}

func (c *CachedHTTPClient) serializeResponse(resp *http.Response) string {
	var b = &bytes.Buffer{}
	_ = resp.Write(b)
	return b.String()
}

func (c *CachedHTTPClient) deserializeResponse(req *http.Request, respStr string) (*http.Response, error) {
	r := bufio.NewReader(strings.NewReader(respStr))
	return http.ReadResponse(r, req)
}
