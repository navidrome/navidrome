.PHONY: all fake_album_agent fake_artist_agent wikimedia

all: fake_album_agent fake_artist_agent wikimedia

fake_album_agent: fake_album_agent/plugin.wasm
fake_artist_agent: fake_artist_agent/plugin.wasm
wikimedia: wikimedia/plugin.wasm

fake_album_agent/plugin.wasm: fake_album_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_album_agent

fake_artist_agent/plugin.wasm: fake_artist_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_artist_agent

wikimedia/plugin.wasm: wikimedia/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./wikimedia 