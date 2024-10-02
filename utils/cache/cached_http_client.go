package cache

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const cacheSizeLimit = 100

type HTTPClient struct {
	cache SimpleCache[string, string]
	hc    httpDoer
	ttl   time.Duration
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

func NewHTTPClient(wrapped httpDoer, ttl time.Duration) *HTTPClient {
	c := &HTTPClient{hc: wrapped, ttl: ttl}
	c.cache = NewSimpleCache[string, string](Options{
		SizeLimit:  cacheSizeLimit,
		DefaultTTL: ttl,
	})
	return c
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := c.serializeReq(req)
	respStr, err := c.cache.GetWithLoader(key, func(key string) (string, time.Duration, error) {
		req, err := c.deserializeReq(key)
		if err != nil {
			return "", 0, err
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			return "", 0, err
		}
		defer resp.Body.Close()
		return c.serializeResponse(resp), c.ttl, nil
	})
	if err != nil {
		return nil, err
	}
	return c.deserializeResponse(req, respStr)
}

func (c *HTTPClient) serializeReq(req *http.Request) string {
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

func (c *HTTPClient) deserializeReq(reqStr string) (*http.Request, error) {
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

func (c *HTTPClient) serializeResponse(resp *http.Response) string {
	var b = &bytes.Buffer{}
	_ = resp.Write(b)
	return b.String()
}

func (c *HTTPClient) deserializeResponse(req *http.Request, respStr string) (*http.Response, error) {
	r := bufio.NewReader(strings.NewReader(respStr))
	return http.ReadResponse(r, req)
}
