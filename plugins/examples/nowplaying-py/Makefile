# Build the Now Playing Logger Python plugin
.PHONY: build test clean

WASM_FILE = nowplaying-py.wasm

build: $(WASM_FILE)

$(WASM_FILE): plugin/__init__.py
	extism-py plugin/__init__.py -o $(WASM_FILE)

clean:
	rm -f $(WASM_FILE)
