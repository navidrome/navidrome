package plugins

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/plugins/host"
)

type HttpService struct {
}

func (s *HttpService) Get(ctx context.Context, req *host.HttpRequest) (*host.HttpResponse, error) {
	return doHttp(ctx, http.MethodGet, req)
}

func (s *HttpService) Post(ctx context.Context, req *host.HttpRequest) (*host.HttpResponse, error) {
	return doHttp(ctx, http.MethodPost, req)
}

func (s *HttpService) Put(ctx context.Context, req *host.HttpRequest) (*host.HttpResponse, error) {
	return doHttp(ctx, http.MethodPut, req)
}

func (s *HttpService) Delete(ctx context.Context, req *host.HttpRequest) (*host.HttpResponse, error) {
	return doHttp(ctx, http.MethodDelete, req)
}

func doHttp(ctx context.Context, method string, req *host.HttpRequest) (*host.HttpResponse, error) {
	client := &http.Client{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.Url, body)
	if err != nil {
		return &host.HttpResponse{Error: err.Error()}, nil
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return &host.HttpResponse{Error: err.Error()}, nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &host.HttpResponse{Error: err.Error()}, nil
	}
	headers := map[string]string{}
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return &host.HttpResponse{
		Status:  int32(resp.StatusCode),
		Body:    respBody,
		Headers: headers,
	}, nil
}
