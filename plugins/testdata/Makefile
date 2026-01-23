# Build test plugins used for integration testing
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Prefer tinygo if available, it produces smaller wasm binaries and
# makes the tests faster.
TINYGO := $(shell command -v tinygo 2> /dev/null)

all: $(PLUGINS:%=%.ndp)

clean:
	rm -f $(PLUGINS:%=%.ndp) $(PLUGINS:%=%.wasm)

# PDK source files that trigger rebuild when changed (recursive)
PDK_SOURCES := $(shell find ../pdk/go -name '*.go' 2>/dev/null)

# Build the .ndp package (zip containing manifest.json + plugin.wasm)
%.ndp: %.wasm %/manifest.json
	@rm -f $@
	@cp $< plugin.wasm
	zip -j $@ $*/manifest.json plugin.wasm
	@rm -f plugin.wasm
	@mv $< $<.tmp && mv $<.tmp $<  # Touch wasm to ensure it's older than ndp

# Build the wasm binary
%.wasm: %/*.go %/go.mod $(PDK_SOURCES)
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .
endif