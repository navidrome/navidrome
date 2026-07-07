package radio

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core/radio/icy"
)

// icyHTTPClient has no overall timeout (ICY streams are long-lived), but bounds
// connection setup and header wait so a stalled station cannot hang the reader.
var icyHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConnsPerHost:   1,
	},
}

func NewMetadataManagerService(publisher TitlePublisher) *MetadataManager {
	return NewMetadataManager(func(ctx context.Context, streamURL string, handleTitle func(string)) error {
		return classifyICYError(icy.ReadHTTPStreamTitles(ctx, icyHTTPClient, streamURL, handleTitle))
	}, publisher)
}

// classifyICYError marks errors that a retry will not fix — a stream without
// ICY metadata or a client-error HTTP status — so the reader backs off for
// much longer instead of hammering the station.
func classifyICYError(err error) error {
	if errors.Is(err, icy.ErrMissingMetaInt) {
		return MarkPermanent(err)
	}
	var statusErr *icy.ResponseStatusError
	if errors.As(err, &statusErr) &&
		statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 &&
		statusErr.StatusCode != http.StatusRequestTimeout && statusErr.StatusCode != http.StatusTooManyRequests {
		return MarkPermanent(err)
	}
	return err
}
