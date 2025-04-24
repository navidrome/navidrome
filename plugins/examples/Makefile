all: wikimedia coverartarchive

wikimedia: wikimedia/plugin.wasm
coverartarchive: coverartarchive/plugin.wasm

wikimedia/plugin.wasm: wikimedia/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./wikimedia

coverartarchive/plugin.wasm: coverartarchive/plugin.go
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $@ ./coverartarchive

clean:
	rm -f wikimedia/plugin.wasm coverartarchive/plugin.wasm 