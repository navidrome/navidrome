# Build the Cover Art Archive Python plugin
.PHONY: build test clean

WASM_FILE = coverartarchive-py.wasm

build: $(WASM_FILE)

$(WASM_FILE): plugin/__init__.py
	extism-py plugin/__init__.py -o $(WASM_FILE)

test: build
	@echo "Testing nd_manifest..."
	extism call $(WASM_FILE) nd_manifest --wasi
	@echo ""
	@echo "Testing nd_get_album_images with Portishead's Dummy MBID..."
	extism call $(WASM_FILE) nd_get_album_images --wasi \
		--input '{"name":"Dummy","artist":"Portishead","mbid":"76df3287-6cda-33eb-8e9a-044b5e15ffdd"}' \
		--allow-host "coverartarchive.org" --allow-host "archive.org"

test-error: build
	@echo "Testing error case (missing MBID)..."
	-extism call $(WASM_FILE) nd_get_album_images --wasi \
		--input '{"name":"Test Album","artist":"Test Artist"}' \
		--allow-host "coverartarchive.org"

clean:
	rm -f $(WASM_FILE)
