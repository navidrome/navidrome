// Package capabilities defines Go interfaces for Navidrome plugin capabilities.
//
// These interfaces serve as the source of truth for capability definitions.
// The ndpgen tool generates:
//   - Go export wrappers in plugins/pdk/go/<capability>/ for Go plugins
//   - XTP YAML schemas for non-Go plugins (Rust, TypeScript, etc.)
//
// Each capability is defined as an annotated interface:
//
//	//nd:capability name=metadata
//	type MetadataAgent interface {
//	    //nd:export name=nd_get_artist_biography
//	    GetArtistBiography(ArtistInput) (ArtistBiographyOutput, error)
//	}
//
// Annotation Reference:
//
//	//nd:capability name=<pkg> [required=true]
//	  - Marks an interface as a capability
//	  - name: Generated package name (e.g., name=metadata → pdk/go/metadata/)
//	  - required: If true, all methods must be implemented (default: false)
//
//	//nd:export name=<func>
//	  - Marks a method as an exported WASM function
//	  - name: The export name (e.g., nd_get_artist_biography)
//
// Generated Code Structure:
//
// For a capability like MetadataAgent with required=false:
//
//	package metadata
//
//	// Agent is the marker interface
//	type Agent interface{}
//
//	// Optional provider interfaces
//	type ArtistBiographyProvider interface {
//	    GetArtistBiography(ArtistInput) (ArtistBiographyOutput, error)
//	}
//
//	// Registration function
//	func Register(impl Agent) { ... }
//
// For a capability with required=true:
//
//	package scrobbler
//
//	// Scrobbler requires all methods
//	type Scrobbler interface {
//	    IsAuthorized(AuthInput) (AuthOutput, error)
//	    NowPlaying(NowPlayingInput) (ScrobblerOutput, error)
//	    Scrobble(ScrobbleInput) (ScrobblerOutput, error)
//	}
//
//	func Register(impl Scrobbler) { ... }
//
//go:generate go run ../cmd/ndpgen -capability-only -input=. -output=../pdk -go -rust
//go:generate go run ../cmd/ndpgen -schemas -input=.
package capabilities
