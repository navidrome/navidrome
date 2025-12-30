module test-library

go 1.25

require (
	github.com/extism/go-pdk v1.1.3
	github.com/navidrome/navidrome/plugins/pdk/go v0.0.0
)

replace github.com/navidrome/navidrome/plugins/pdk/go => ../../pdk/go
