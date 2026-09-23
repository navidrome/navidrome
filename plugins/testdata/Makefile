# Build test plugins used for integration testing
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Prefer tinygo if available, it produces smaller wasm binaries and
# makes the tests faster.
TINYGO := $(shell command -v tinygo 2> /dev/null)

all: $(PLUGINS:%=%.ndp)

clean:
	rm -f $(PLUGINS:%=%.ndp) $(PLUGINS:%=%.wasm)
	rm -rf .wazero-cache $(PLUGINS:%=%.stage)

# PDK source files that trigger rebuild when changed (recursive)
PDK_SOURCES := $(shell find ../pdk/go -name '*.go' 2>/dev/null)

# Build the .ndp package (zip containing manifest.json + plugin.wasm)
# Stage under a per-target name: a shared plugin.wasm breaks concurrent builds.
%.ndp: %.wasm %/manifest.json
	@rm -f $@
	@rm -rf $*.stage && mkdir -p $*.stage
	@cp $< $*.stage/plugin.wasm
	zip -j $@ $*/manifest.json $*.stage/plugin.wasm
	@rm -rf $*.stage
	@mv $< $<.tmp && mv $<.tmp $<  # Touch wasm to ensure it's older than ndp

# Build the wasm binary. -buildvcs=false keeps the bytes stable across commits, so
# the test suite's wazero compilation cache still hits after a rebuild.
%.wasm: %/*.go %/go.mod $(PDK_SOURCES)
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o ../$@ .
endif