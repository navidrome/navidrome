all: wikimedia coverartarchive crypto-ticker discord-rich-presence

wikimedia: wikimedia/plugin.wasm
coverartarchive: coverartarchive/plugin.wasm
crypto-ticker: crypto-ticker/plugin.wasm
discord-rich-presence: discord-rich-presence/plugin.wasm

wikimedia/plugin.wasm: wikimedia/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./wikimedia

coverartarchive/plugin.wasm: coverartarchive/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./coverartarchive

crypto-ticker/plugin.wasm: crypto-ticker/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./crypto-ticker

DISCORD_RP_FILES=$(shell find discord-rich-presence -type f -name "*.go")
discord-rich-presence/plugin.wasm: $(DISCORD_RP_FILES)
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./discord-rich-presence/...

clean:
	rm -f wikimedia/plugin.wasm coverartarchive/plugin.wasm crypto-ticker/plugin.wasm discord-rich-presence/plugin.wasm