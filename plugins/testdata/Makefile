# Fake sample plugins used for testing
PLUGINS := fake_album_agent fake_artist_agent fake_scrobbler multi_plugin fake_init_service unauthorized_plugin

all: $(PLUGINS:%=%/plugin.wasm)

clean:
	rm -f $(PLUGINS:%=%/plugin.wasm)

%/plugin.wasm: %/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./$*