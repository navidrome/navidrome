# Build example plugins for Navidrome
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Prefer tinygo if available, it produces smaller wasm binaries.
TINYGO := $(shell command -v tinygo 2> /dev/null)

# Default target: show available plugins
.DEFAULT_GOAL := help

help:
	@echo "Available plugins:"
	@$(foreach p,$(PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Usage:"
	@echo "  make <plugin>    Build a specific plugin (e.g., make $(firstword $(PLUGINS)))"
	@echo "  make all         Build all plugins"
	@echo "  make clean       Remove all built plugins"

all: $(PLUGINS:%=%.wasm)

clean:
	rm -f $(PLUGINS:%=%.wasm)

%.wasm: %/*.go %/go.mod
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .
endif
