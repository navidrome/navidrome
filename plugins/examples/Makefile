all: wikimedia coverartarchive crypto-ticker

wikimedia: wikimedia/plugin.wasm
coverartarchive: coverartarchive/plugin.wasm
crypto-ticker: crypto-ticker/plugin.wasm

wikimedia/plugin.wasm: wikimedia/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./wikimedia

coverartarchive/plugin.wasm: coverartarchive/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./coverartarchive

crypto-ticker/plugin.wasm: crypto-ticker/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./crypto-ticker

clean:
	rm -f wikimedia/plugin.wasm coverartarchive/plugin.wasm crypto-ticker/plugin.wasm