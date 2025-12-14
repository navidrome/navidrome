package http

import (
	"cmp"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

func SetSSECORSHeaders(w http.ResponseWriter, r *http.Request) {
	rScheme, rHost := ServerAddress(r)
	w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("%s://%s", rScheme, rHost))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func SetJWTEventCookie(w http.ResponseWriter, r *http.Request, token string) {
	AddCookie(w, r, consts.JWTCookie, token, "/api/events", int(consts.DefaultSessionTimeout/time.Second))
}

func ClearJWTEventCookie(w http.ResponseWriter, r *http.Request) {
	AddCookie(w, r, consts.JWTCookie, "", "/api/events", -1)
}

func AddCookie(w http.ResponseWriter, r *http.Request, name string, value string, path string, maxAge int) {
	isSecure := strings.HasPrefix(conf.Server.BaseScheme, "https")

	// http by default so cookie is not rejected
	sameSite := http.SameSiteLaxMode
	if isSecure {
		// only if https is enabled
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		MaxAge:   maxAge,
	})
}

// Define constants for the X-Forwarded-* header keys.
var (
	xForwardedHost   = http.CanonicalHeaderKey("X-Forwarded-Host")
	xForwardedProto  = http.CanonicalHeaderKey("X-Forwarded-Proto")
	xForwardedScheme = http.CanonicalHeaderKey("X-Forwarded-Scheme")
)

// serverAddress is a helper function that returns the scheme and host of the server
// handling the given request, as determined by the presence of X-Forwarded-* headers
// or the scheme and host of the request URL.
func ServerAddress(r *http.Request) (scheme, host string) {
	// Save the original request host for later comparison.
	origHost := r.Host

	// Determine the protocol of the request based on the presence of a TLS connection.
	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}

	// Get the X-Forwarded-Host header and extract the first host name if there are
	// multiple hosts listed. If there is no X-Forwarded-Host header, use the original
	// request host as the default.
	xfh := r.Header.Get(xForwardedHost)
	if xfh != "" {
		i := strings.Index(xfh, ",")
		if i == -1 {
			i = len(xfh)
		}
		xfh = xfh[:i]
	}
	host = cmp.Or(xfh, r.Host)

	// Determine the protocol and scheme of the request based on the presence of
	// X-Forwarded-* headers or the scheme of the request URL.
	scheme = cmp.Or(
		r.Header.Get(xForwardedProto),
		r.Header.Get(xForwardedScheme),
		r.URL.Scheme,
		protocol,
	)

	// If the request host has changed due to the X-Forwarded-Host header, log a trace
	// message with the original and new host values, as well as the scheme and URL.
	if host != origHost {
		log.Trace(r.Context(), "Request host has changed", "origHost", origHost, "host", host, "scheme", scheme, "url", r.URL)
	}

	// Return the scheme and host of the server handling the request.
	return scheme, host
}
