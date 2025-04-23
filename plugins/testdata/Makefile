.PHONY: all clean fake_album_agent fake_artist_agent wikimedia fake_multi_agent coverartarchive

all: fake_album_agent fake_artist_agent wikimedia fake_multi_agent coverartarchive

clean:
	rm -f fake_album_agent/plugin.wasm fake_artist_agent/plugin.wasm wikimedia/plugin.wasm fake_multi_agent/plugin.wasm coverartarchive/plugin.wasm

fake_album_agent: fake_album_agent/plugin.wasm
fake_artist_agent: fake_artist_agent/plugin.wasm
wikimedia: wikimedia/plugin.wasm
fake_multi_agent: fake_multi_agent/plugin.wasm
coverartarchive: coverartarchive/plugin.wasm

fake_album_agent/plugin.wasm: fake_album_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_album_agent

fake_artist_agent/plugin.wasm: fake_artist_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_artist_agent

wikimedia/plugin.wasm: wikimedia/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./wikimedia

fake_multi_agent/plugin.wasm: fake_multi_agent/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./fake_multi_agent

coverartarchive/plugin.wasm: coverartarchive/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./coverartarchive 