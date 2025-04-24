.PHONY: all clean fake_album_agent fake_artist_agent fake_multi_agent fake_scrobbler

all: fake_album_agent fake_artist_agent fake_multi_agent fake_scrobbler

clean:
	rm -f fake_album_agent/plugin.wasm fake_artist_agent/plugin.wasm fake_multi_agent/plugin.wasm fake_scrobbler/plugin.wasm

# Fake sample plugins used for testing
fake_album_agent: fake_album_agent/plugin.wasm
fake_artist_agent: fake_artist_agent/plugin.wasm
fake_multi_agent: fake_multi_agent/plugin.wasm
fake_scrobbler: fake_scrobbler/plugin.wasm

fake_album_agent/plugin.wasm: fake_album_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_album_agent

fake_artist_agent/plugin.wasm: fake_artist_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_artist_agent

fake_multi_agent/plugin.wasm: fake_multi_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_multi_agent

fake_scrobbler/plugin.wasm: fake_scrobbler/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_scrobbler 