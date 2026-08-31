package publicurl

import (
	"cmp"
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// ImageURL generates a public URL for artwork images.
// It creates a signed token for the artwork ID and builds a complete public URL.
func ImageURL(ctx context.Context, artID model.ArtworkID, size int) string {
	token, _ := auth.CreatePublicToken(auth.Claims{ID: artID.String()})
	uri := path.Join(consts.URLPathPublicImages, token)
	params := url.Values{}
	if size > 0 {
		params.Add("size", strconv.Itoa(size))
	}
	return PublicURL(ctx, uri, params)
}

// PublicURL builds a full URL for public-facing resources.
// It uses ShareURL from config if available, otherwise falls back to the address the
// client used to reach the server, recorded in the context.
func PublicURL(ctx context.Context, u string, params url.Values) string {
	if conf.Server.ShareURL == "" {
		return AbsoluteURL(ctx, u, params)
	}
	shareUrl, err := url.Parse(conf.Server.ShareURL)
	if err != nil {
		return AbsoluteURL(ctx, u, params)
	}
	buildUrl, err := url.Parse(u)
	if err != nil {
		return AbsoluteURL(ctx, u, params)
	}
	buildUrl.Scheme = shareUrl.Scheme
	buildUrl.Host = shareUrl.Host
	if basePath := strings.TrimRight(shareUrl.Path, "/"); basePath != "" {
		buildUrl.Path = path.Join(basePath, buildUrl.Path)
	}
	if len(params) > 0 {
		buildUrl.RawQuery = params.Encode()
	}
	return buildUrl.String()
}

// AbsoluteURL builds an absolute URL from a relative path.
// It uses BaseHost/BaseScheme from config if available, otherwise the address the client
// used to reach the server, recorded in the context by the server's address middleware.
func AbsoluteURL(ctx context.Context, u string, params url.Values) string {
	buildUrl, err := url.Parse(u)
	if err != nil {
		log.Error(ctx, "Failed to parse URL path", "url", u, err)
		return ""
	}
	if strings.HasPrefix(u, "/") {
		buildUrl.Path = path.Join(conf.Server.BasePath, buildUrl.Path)
		if conf.Server.BaseHost != "" {
			buildUrl.Scheme = cmp.Or(conf.Server.BaseScheme, "http")
			buildUrl.Host = conf.Server.BaseHost
		} else if scheme, host, ok := request.ServerAddressFrom(ctx); ok {
			buildUrl.Scheme = scheme
			buildUrl.Host = host
		} else {
			log.Debug(ctx, "Building a public URL with no public address available; set ShareURL to make it reachable", "url", u)
			// Both a cert and a key are required, matching the server's own TLS switch.
			buildUrl.Scheme = "http"
			if conf.Server.TLSCert != "" && conf.Server.TLSKey != "" {
				buildUrl.Scheme = "https"
			}
			buildUrl.Host = "localhost:" + strconv.Itoa(conf.Server.Port)
		}
	}
	if len(params) > 0 {
		buildUrl.RawQuery = params.Encode()
	}
	return buildUrl.String()
}
