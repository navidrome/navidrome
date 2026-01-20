package publicurl

import (
	"cmp"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// ImageURL generates a public URL for artwork images.
// It creates a signed token for the artwork ID and builds a complete public URL.
func ImageURL(req *http.Request, artID model.ArtworkID, size int) string {
	token, _ := auth.CreatePublicToken(map[string]any{"id": artID.String()})
	uri := path.Join(consts.URLPathPublicImages, token)
	params := url.Values{}
	if size > 0 {
		params.Add("size", strconv.Itoa(size))
	}
	return PublicURL(req, uri, params)
}

// PublicURL builds a full URL for public-facing resources.
// It uses ShareURL from config if available, otherwise falls back to extracting
// the scheme and host from the provided http.Request.
// If req is nil and ShareURL is not set, it defaults to http://localhost.
func PublicURL(req *http.Request, u string, params url.Values) string {
	if conf.Server.ShareURL == "" {
		return AbsoluteURL(req, u, params)
	}
	shareUrl, err := url.Parse(conf.Server.ShareURL)
	if err != nil {
		return AbsoluteURL(req, u, params)
	}
	buildUrl, err := url.Parse(u)
	if err != nil {
		return AbsoluteURL(req, u, params)
	}
	buildUrl.Scheme = shareUrl.Scheme
	buildUrl.Host = shareUrl.Host
	if len(params) > 0 {
		buildUrl.RawQuery = params.Encode()
	}
	return buildUrl.String()
}

// AbsoluteURL builds an absolute URL from a relative path.
// It uses BaseHost/BaseScheme from config if available, otherwise extracts
// the scheme and host from the http.Request.
// If req is nil and BaseHost is not set, it defaults to http://localhost.
func AbsoluteURL(req *http.Request, u string, params url.Values) string {
	buildUrl, err := url.Parse(u)
	if err != nil {
		log.Error(req.Context(), "Failed to parse URL path", "url", u, err)
		return ""
	}
	if strings.HasPrefix(u, "/") {
		buildUrl.Path = path.Join(conf.Server.BasePath, buildUrl.Path)
		if conf.Server.BaseHost != "" {
			buildUrl.Scheme = cmp.Or(conf.Server.BaseScheme, "http")
			buildUrl.Host = conf.Server.BaseHost
		} else if req != nil {
			buildUrl.Scheme = req.URL.Scheme
			buildUrl.Host = req.Host
		} else {
			buildUrl.Scheme = "http"
			buildUrl.Host = "localhost"
		}
	}
	if len(params) > 0 {
		buildUrl.RawQuery = params.Encode()
	}
	return buildUrl.String()
}
