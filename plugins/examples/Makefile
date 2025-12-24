# Build example plugins for Navidrome
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Prefer tinygo if available, it produces smaller wasm binaries.
TINYGO := $(shell command -v tinygo 2> /dev/null)

all: $(PLUGINS:%=%.wasm)

clean:
	rm -f $(PLUGINS:%=%.wasm)

%.wasm: %/*.go %/go.mod
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .
endif
