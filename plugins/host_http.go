package plugins

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	hosthttp "github.com/navidrome/navidrome/plugins/host/http"
)

type httpServiceImpl struct {
	pluginName string
}

func (s *httpServiceImpl) Get(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return doHttp(ctx, http.MethodGet, req)
}

func (s *httpServiceImpl) Post(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return doHttp(ctx, http.MethodPost, req)
}

func (s *httpServiceImpl) Put(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return doHttp(ctx, http.MethodPut, req)
}

func (s *httpServiceImpl) Delete(ctx context.Context, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	return doHttp(ctx, http.MethodDelete, req)
}

func doHttp(ctx context.Context, method string, req *hosthttp.HttpRequest) (*hosthttp.HttpResponse, error) {
	client := &http.Client{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut {
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
