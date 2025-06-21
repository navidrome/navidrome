package plugins

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	hosthttp "github.com/navidrome/navidrome/plugins/host/http"
)

type httpServiceImpl struct {
	pluginID    string
	permissions *httpPermissions
}

const defaultTimeout = 10 * time.Second

func (s *httpServiceImpl) Get(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodGet, req)
}

func (s *httpServiceImpl) Post(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodPost, req)
}

func (s *httpServiceImpl) Put(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodPut, req)
}

func (s *httpServiceImpl) Delete(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodDelete, req)
}

func (s *httpServiceImpl) Patch(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodPatch, req)
}

func (s *httpServiceImpl) Head(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodHead, req)
}

func (s *httpServiceImpl) Options(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return s.doHttp(ctx, http.MethodOptions, req)
}

func (s *httpServiceImpl) doHttp(ctx context.Context, method string, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	// Check permissions if they exist
	if s.permissions != nil {
		if err := s.permissions.IsRequestAllowed(req.Url, method); err != nil {
			log.Warn(ctx, "HTTP request blocked by permissions", "plugin", s.pluginID, "url", req.Url, "method", method, err)
			return &hosthttp.HttpResponse{Error: "Request blocked by plugin permissions: " + err.Error()}, nil
		}
	}
	client := &http.Client{
		Timeout: cmp.Or(time.Duration(req.TimeoutMs)*time.Millisecond, defaultTimeout),
	}

	// Configure redirect policy based on permissions
	if s.permissions != nil {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			// Enforce maximum redirect limit
			if len(via) >= httpMaxRedirects {
				log.Warn(ctx, "HTTP redirect limit exceeded", "plugin", s.pluginID, "url", req.URL.String(), "redirectCount", len(via))
				return http.ErrUseLastResponse
			}

			// Check if redirect destination is allowed
			if err := s.permissions.IsRequestAllowed(req.URL.String(), req.Method); err != nil {
				log.Warn(ctx, "HTTP redirect blocked by permissions", "plugin", s.pluginID, "url", req.URL.String(), "method", req.Method, err)
				return http.ErrUseLastResponse
			}

			return nil // Allow redirect
		}
	}
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.Url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Trace(ctx, "HttpService request error", "method", method, "url", req.Url, "headers", req.Headers, err)
		return &hosthttp.HttpResponse{Error: err.Error()}, nil
	}
	log.Trace(ctx, "HttpService request", "method", method, "url", req.Url, "headers", req.Headers, "resp.status", resp.StatusCode)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Trace(ctx, "HttpService request error", "method", method, "url", req.Url, "headers", req.Headers, "resp.status", resp.StatusCode, err)
		return &hosthttp.HttpResponse{Error: err.Error()}, nil
	}
	headers := map[string]string{}
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return &hosthttp.HttpResponse{
		Status:  int32(resp.StatusCode),
		Body:    respBody,
		Headers: headers,
	}, nil
}
