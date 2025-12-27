# Build example plugins for Navidrome
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Auto-discover Python plugins (folders containing plugin/__init__.py)
PYTHON_PLUGINS := $(patsubst %/plugin/__init__.py,%,$(wildcard */plugin/__init__.py))

# Prefer tinygo if available, it produces smaller wasm binaries.
TINYGO := $(shell command -v tinygo 2> /dev/null)
EXTISM_PY := $(shell command -v extism-py 2> /dev/null)

# Default target: show available plugins
.DEFAULT_GOAL := help

help:
	@echo "Available Go plugins:"
	@$(foreach p,$(PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Available Python plugins:"
	@$(foreach p,$(PYTHON_PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Usage:"
	@echo "  make <plugin>.wasm    Build a specific plugin (e.g., make $(firstword $(PLUGINS)).wasm)"
	@echo "  make all              Build all plugins"
	@echo "  make all-go           Build all Go plugins"
	@echo "  make all-python       Build all Python plugins (requires extism-py)"
	@echo "  make clean            Remove all built plugins"

all: all-go all-python

all-go: $(PLUGINS:%=%.wasm)

all-python: $(PYTHON_PLUGINS:%=%.wasm)

clean:
	rm -f $(PLUGINS:%=%.wasm) $(PYTHON_PLUGINS:%=%.wasm)

%.wasm: %/*.go %/go.mod
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .
endif

# Python plugin builds (generic rule for any folder with plugin/__init__.py)
# Use secondary expansion to get all .py files in the plugin directory as dependencies
.SECONDEXPANSION:
$(PYTHON_PLUGINS:%=%.wasm): %.wasm: $$(wildcard %/plugin/*.py)
ifndef EXTISM_PY
	$(error extism-py is not installed. Install from https://github.com/extism/python-pdk)
endif
	cd $* && PYTHONPATH=plugin extism-py plugin/__init__.py -o ../$@
