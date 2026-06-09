# Build example plugins for Navidrome
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Auto-discover Python plugins (folders containing plugin/__init__.py)
PYTHON_PLUGINS := $(patsubst %/plugin/__init__.py,%,$(wildcard */plugin/__init__.py))

# Auto-discover Rust plugins (folders containing Cargo.toml)
RUST_PLUGINS := $(patsubst %/Cargo.toml,%,$(wildcard */Cargo.toml))

# Prefer tinygo if available, it produces smaller wasm binaries.
TINYGO := $(shell command -v tinygo 2> /dev/null)
EXTISM_PY := $(shell command -v extism-py 2> /dev/null)

# PDK source files that trigger rebuild when changed (recursive)
PDK_GO_SOURCES := $(shell find ../pdk/go -name '*.go' 2>/dev/null)
PDK_PY_SOURCES := $(shell find ../pdk/python -name '*.py' 2>/dev/null)
PDK_RS_SOURCES := $(shell find ../pdk/rust -name '*.rs' 2>/dev/null)

# Allow building plugins without .ndp extension (e.g., make minimal instead of make minimal.ndp)
.PHONY: $(PLUGINS) $(PYTHON_PLUGINS) $(RUST_PLUGINS)
$(PLUGINS): %: %.ndp
$(PYTHON_PLUGINS): %: %.ndp
$(RUST_PLUGINS): %: %.ndp

# Default target: show available plugins
.DEFAULT_GOAL := help

help:
	@echo "Available Go plugins:"
	@$(foreach p,$(PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Available Python plugins:"
	@$(foreach p,$(PYTHON_PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Available Rust plugins:"
	@$(foreach p,$(RUST_PLUGINS),echo "  $(p)";)
	@echo ""
	@echo "Usage:"
	@echo "  make <plugin>         Build a specific plugin (e.g., make $(firstword $(PLUGINS)))"
	@echo "  make all              Build all plugins"
	@echo "  make all-go           Build all Go plugins"
	@echo "  make all-python       Build all Python plugins (requires extism-py)"
	@echo "  make all-rust         Build all Rust plugins (requires cargo)"
	@echo "  make clean            Remove all built plugins (.ndp and .wasm files)"

all: all-go all-python all-rust

all-go: $(PLUGINS:%=%.ndp)

all-python: $(PYTHON_PLUGINS:%=%.ndp)

all-rust: $(RUST_PLUGINS:%=%.ndp)

clean:
	rm -f $(PLUGINS:%=%.ndp) $(PYTHON_PLUGINS:%=%.ndp) $(RUST_PLUGINS:%=%.ndp)
	rm -f $(PLUGINS:%=%.wasm) $(PYTHON_PLUGINS:%=%.wasm) $(RUST_PLUGINS:%=%.wasm)
	@$(foreach p,$(RUST_PLUGINS),(cd $(p) && cargo clean 2>/dev/null) || true;)

# Mark .wasm files as intermediate so Make deletes them after building .ndp
.INTERMEDIATE: $(PLUGINS:%=%.wasm) $(PYTHON_PLUGINS:%=%.wasm) $(RUST_PLUGINS:%=%.wasm)

# Build .ndp package from .wasm and manifest.json
# Go plugins
%.ndp: %.wasm %/manifest.json
	@rm -f $@
	@cp $< plugin.wasm
	zip -j $@ $*/manifest.json plugin.wasm
	@rm -f plugin.wasm
	@mv $< $<.tmp && mv $<.tmp $<  # Touch wasm to ensure it's older than ndp

# Use secondary expansion to properly track all Go source files
.SECONDEXPANSION:
$(PLUGINS:%=%.wasm): %.wasm: $$(shell find % -name '*.go' 2>/dev/null) %/go.mod $(PDK_GO_SOURCES)
ifdef TINYGO
	cd $* && tinygo build -target wasip1 -buildmode=c-shared -o ../$@ .
else
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .
endif

# Python plugin builds (generic rule for any folder with plugin/__init__.py)
# Use secondary expansion to get all .py files in the plugin directory as dependencies
.SECONDEXPANSION:
$(PYTHON_PLUGINS:%=%.wasm): %.wasm: $$(wildcard %/plugin/*.py) $(PDK_PY_SOURCES)
ifndef EXTISM_PY
	$(error extism-py is not installed. Install from https://github.com/extism/python-pdk)
endif
	cd $* && PYTHONPATH=plugin extism-py plugin/__init__.py -o ../$@

# Rust plugin builds (generic rule for any folder with Cargo.toml)
# Note: Rust crate names use underscores, but plugin names use hyphens
# All Rust plugins use wasm32-wasip1 for WASI support (filesystem, etc.)
RUST_TARGET := wasm32-wasip1
RUSTUP_CARGO := $(shell rustup which cargo 2>/dev/null || echo cargo)
RUSTUP_RUSTC := $(shell rustup which rustc 2>/dev/null)
$(RUST_PLUGINS:%=%.wasm): %.wasm: %/Cargo.toml $$(wildcard %/src/*.rs) $(PDK_RS_SOURCES)
	cd $* && CARGO_BUILD_RUSTC=$(RUSTUP_RUSTC) $(RUSTUP_CARGO) build --release --target $(RUST_TARGET)
	cp $*/target/$(RUST_TARGET)/release/$(subst -,_,$*).wasm $@
