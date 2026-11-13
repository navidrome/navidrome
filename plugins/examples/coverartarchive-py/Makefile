# Build the Cover Art Archive Python plugin
.PHONY: build clean

WASM_FILE = coverartarchive-py.wasm

build: $(WASM_FILE)

$(WASM_FILE): plugin/__init__.py
	extism-py plugin/__init__.py -o $(WASM_FILE)

clean:
	rm -f $(WASM_FILE)
