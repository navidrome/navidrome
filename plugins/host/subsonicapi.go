package host

import "context"

// SubsonicAPIService provides access to Navidrome's Subsonic API from plugins.
//
// This service allows plugins to make Subsonic API requests on behalf of the plugin's user,
// enabling access to library data, user preferences, and other Subsonic-compatible operations.
//
//nd:hostservice name=SubsonicAPI permission=subsonicapi
type SubsonicAPIService interface {
	// Call executes a Subsonic API request and returns the JSON response.
	//
	// The uri parameter should be the Subsonic API path without the server prefix,
	// e.g., "getAlbumList2?type=random&size=10". The response is returned as raw JSON.
	//nd:hostfunc
	Call(ctx context.Context, uri string) (responseJSON string, err error)

	// CallRaw executes a Subsonic API request and returns the raw binary response.
	// Designed for binary endpoints like getCoverArt and stream that return
	// non-JSON data. The data is base64-encoded over JSON on the wire.
	//nd:hostfunc
	CallRaw(ctx context.Context, uri string) (contentType string, data []byte, err error)
}
