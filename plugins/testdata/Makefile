# Build fake sample plugins used for testing
# Auto-discover all plugin folders (folders containing go.mod)
PLUGINS := $(patsubst %/go.mod,%,$(wildcard */go.mod))

all: $(PLUGINS:%=%.wasm)

clean:
	rm -f $(PLUGINS:%=%.wasm)

%.wasm: %/main.go %/go.mod
	cd $* && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$@ .