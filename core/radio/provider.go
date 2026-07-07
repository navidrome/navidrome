package radio

import (
	"context"
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
		return icy.ReadHTTPStreamTitles(ctx, icyHTTPClient, streamURL, handleTitle)
	}, publisher)
}
