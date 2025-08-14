// Package cache provides an HTTP client with caching capabilities for HTTP responses.
// It wraps an http.Client (or any type implementing the httpDoer interface) to cache
// responses based on request details, using a simple in-memory cache with a configurable TTL.
package cache

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

// cacheSizeLimit defines the maximum number of entries in the cache.
const cacheSizeLimit = 100

// HTTPClient is a caching HTTP client that wraps an httpDoer to cache responses.
// It serializes requests and responses to store them in a SimpleCache, using a
// specified TTL for cache entries.
type HTTPClient struct {
	cache SimpleCache[string, string]
	hc    httpDoer
	ttl   time.Duration
}

// httpDoer defines an interface for performing HTTP requests, typically implemented
// by http.Client.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// requestData represents the serialized form of an HTTP request for caching.
type requestData struct {
	Method string
	Header http.Header
	URL    string
	Body   *string
}

// NewHTTPClient creates a new caching HTTP client.
// It initializes a SimpleCache with the specified size limit and TTL.
//
// Parameters:
//   - wrapped: The underlying HTTP client (or httpDoer) to perform requests.
//   - ttl: The time-to-live duration for cached responses.
//
// Returns:
//   - A pointer to the initialized HTTPClient.
func NewHTTPClient(wrapped httpDoer, ttl time.Duration) *HTTPClient {
	c := &HTTPClient{hc: wrapped, ttl: ttl}
	c.cache = NewSimpleCache[string, string](Options{
		SizeLimit:  cacheSizeLimit,
		DefaultTTL: ttl,
	})
	return c
}

// Do executes an HTTP request, returning a cached response if available or fetching
// a new one if not. The request is serialized to generate a cache key, and the response
// is cached for the configured TTL.
//
// Parameters:
//   - req: The HTTP request to execute.
//
// Returns:
//   - The HTTP response (cached or fresh).
//   - An error if the request fails, serialization/deserialization fails, or the cache operation fails.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	key, err := c.serializeReq(req)
	if err != nil {
		return nil, err
	}
	cached := true
	start := time.Now()
	respStr, err := c.cache.GetWithLoader(key, func(key string) (string, time.Duration, error) {
		cached = false
		req, err := c.deserializeReq(key)
		if err != nil {
			log.Trace(req.Context(), "CachedHTTPClient.Do", "key", key, err)
			return "", 0, err
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			log.Trace(req.Context(), "CachedHTTPClient.Do", "req", req, err)
			return "", 0, err
		}
		defer resp.Body.Close()
		respSerialized, err := c.serializeResponse(resp)
		if err != nil {
			log.Trace(req.Context(), "CachedHTTPClient.Do", "req", req, err)
			return "", 0, err
		}
		return respSerialized, c.ttl, nil
	})
	log.Trace(req.Context(), "CachedHTTPClient.Do", "key", key, "cached", cached, "elapsed", time.Since(start), err)
	if err != nil {
		return nil, err
	}
	return c.deserializeResponse(req, respStr)
}

// serializeReq converts an HTTP request to a JSON string for use as a cache key.
// It includes the method, URL, headers, and body (base64-encoded if present).
// The request body is closed after reading to prevent resource leaks.
//
// Parameters:
//   - req: The HTTP request to serialize.
//
// Returns:
//   - The serialized request as a JSON string.
//   - An error if reading the body or marshaling to JSON fails.
func (c *HTTPClient) serializeReq(req *http.Request) (string, error) {
	var bodyB64 *string
	if req.Body != nil {
		defer req.Body.Close()
		bodyData, err := io.ReadAll(req.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read request body: %w", err)
		}
		b64 := base64.StdEncoding.EncodeToString(bodyData)
		bodyB64 = &b64
	}
	data := requestData{
		Method: req.Method,
		Header: req.Header,
		URL:    req.URL.String(),
		Body:   bodyB64,
	}

	j, err := json.Marshal(&data)
	if err != nil {
		log.Trace(req.Context(), "CachedHTTPClient.serializeReq", "req", req, err)
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}
	return string(j), nil
}

// deserializeReq reconstructs an HTTP request from its serialized JSON form.
// The body, if present, is decoded from base64 and set as the request body.
//
// Parameters:
//   - reqStr: The serialized request (JSON string).
//
// Returns:
//   - The reconstructed HTTP request.
//   - An error if unmarshaling, decoding the body, or creating the request fails.
func (c *HTTPClient) deserializeReq(reqStr string) (*http.Request, error) {
	var data requestData
	err := json.Unmarshal([]byte(reqStr), &data)
	if err != nil {
		log.Trace("CachedHTTPClient.deserializeReq", err)
		return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
	}
	var body io.Reader
	if data.Body != nil {
		bodyStr, err := base64.StdEncoding.DecodeString(*data.Body)
		if err != nil {
			log.Trace("CachedHTTPClient.deserializeReq", err)
			return nil, fmt.Errorf("failed to decode request body: %w", err)
		}
		body = strings.NewReader(string(bodyStr))
	}
	req, err := http.NewRequest(data.Method, data.URL, body)
	if err != nil {
		return nil, err
	}
	req.Header = data.Header
	if data.Body != nil && req.ContentLength == 0 {
		req.ContentLength = int64(len(*data.Body))
	}
	return req, nil
}

// serializeResponse converts an HTTP response to a string for caching.
// It writes the response (including headers and body) to a buffer.
//
// Parameters:
//   - resp: The HTTP response to serialize.
//
// Returns:
//   - The serialized response as a string.
//   - An error if writing the response fails.
func (c *HTTPClient) serializeResponse(resp *http.Response) (string, error) {
	var b = &bytes.Buffer{}
	err := resp.Write(b)
	if err != nil {
		log.Trace("CachedHTTPClient.serializeResponse", err)
		return "", fmt.Errorf("failed to serialize response: %w", err)
	}
	return b.String(), nil
}

// deserializeResponse reconstructs an HTTP response from its serialized form.
// It reads the response from a string and associates it with the original request.
//
// Parameters:
//   - req: The original HTTP request associated with the response.
//   - respStr: The serialized response string.
//
// Returns:
//   - The reconstructed HTTP response.
//   - An error if parsing the response fails.
func (c *HTTPClient) deserializeResponse(req *http.Request, respStr string) (*http.Response, error) {
	r := bufio.NewReader(strings.NewReader(respStr))
	return http.ReadResponse(r, req)
}
