package radio

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/core/radio/icy"
)

func NewMetadataManagerService(publisher TitlePublisher) *MetadataManager {
	return NewMetadataManager(func(ctx context.Context, streamURL string, handleTitle func(string)) error {
		return icy.ReadHTTPStreamTitles(ctx, http.DefaultClient, streamURL, handleTitle)
	}, publisher)
}
