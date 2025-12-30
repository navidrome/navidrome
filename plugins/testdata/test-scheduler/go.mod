module test-scheduler

go 1.25

require (
	github.com/navidrome/navidrome v0.0.0
	github.com/navidrome/navidrome/plugins/pdk/go/host v0.0.0
)

require github.com/extism/go-pdk v1.1.3 // indirect

replace github.com/navidrome/navidrome => ../../..

replace github.com/navidrome/navidrome/plugins/pdk/go/host => ../../pdk/go/host
