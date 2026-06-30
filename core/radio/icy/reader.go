package icy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

var ErrMissingMetaInt = errors.New("missing icy metadata interval")

type ResponseStatusError struct {
	StatusCode int
	Status     string
}

func (e *ResponseStatusError) Error() string {
	return fmt.Sprintf("icy stream returned %s", e.Status)
}

// ReadHTTPStreamTitles requests ICY metadata from streamURL and emits changed StreamTitle values.
func ReadHTTPStreamTitles(ctx context.Context, client *http.Client, streamURL string, handleTitle func(string)) error {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Icy-MetaData", "1")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &ResponseStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}

	metaIntHeader := resp.Header.Get("icy-metaint")
	if metaIntHeader == "" {
		return ErrMissingMetaInt
	}

	metaInt, err := strconv.Atoi(metaIntHeader)
	if err != nil || metaInt <= 0 {
		return fmt.Errorf("%w: %q", ErrInvalidMetaInt, metaIntHeader)
	}

	return ReadStreamTitles(ctx, resp.Body, metaInt, handleTitle)
}
