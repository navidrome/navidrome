module discord-rich-presence

go 1.25

require (
	github.com/extism/go-pdk v1.1.3
	github.com/navidrome/navidrome v0.0.0-00010101000000-000000000000
	github.com/navidrome/navidrome/plugins/pdk/go/host v0.0.0-00010101000000-000000000000
)

replace github.com/navidrome/navidrome => ../../..

replace github.com/navidrome/navidrome/plugins/pdk/go/host => ../../pdk/go/host
