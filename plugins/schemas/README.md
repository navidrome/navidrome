# Navidrome Plugin Schemas

This directory contains [XTP schemas](https://docs.xtp.dylibso.com/docs/concepts/xtp-schema) that define plugin capabilities. Use these schemas to bootstrap new plugins with the `xtp` CLI.

## Available Schemas

| Schema                                             | Description                     |
|----------------------------------------------------|---------------------------------|
| [metadata_agent.yaml](metadata_agent.yaml)         | Artist/album metadata retrieval |
| [scrobbler.yaml](scrobbler.yaml)                   | Scrobbling to external services |
| [lifecycle.yaml](lifecycle.yaml)                   | Plugin initialization callback  |
| [scheduler_callback.yaml](scheduler_callback.yaml) | Scheduled task callbacks        |
| [websocket_callback.yaml](websocket_callback.yaml) | WebSocket event callbacks       |

## Prerequisites

Install the XTP CLI:

```bash
curl -fsSL https://static.dylibso.com/cli/install.sh | bash
```

Or see [XTP CLI documentation](https://docs.xtp.dylibso.com/docs/cli) for other methods.

## Bootstrapping a Plugin

### Basic Usage

```bash
xtp plugin init \
  --schema-file <schema> \
  --template <language> \
  --path <output-dir> \
  --name <plugin-name>
```

### Supported Languages

- `go` – Go (recommended, use with TinyGo)
- `rust` – Rust
- `typescript` – TypeScript
- `python` – Python
- `csharp` – C#
- `zig` – Zig
- `cpp` – C++

### Examples

**Go metadata agent:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/metadata_agent.yaml \
  --template go \
  --path ./my-agent \
  --name my-agent
```

**Rust scrobbler:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/scrobbler.yaml \
  --template rust \
  --path ./my-scrobbler \
  --name my-scrobbler
```

**TypeScript scrobbler:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/scrobbler.yaml \
  --template typescript \
  --path ./ts-scrobbler \
  --name ts-scrobbler
```

## Generated Files

After running `xtp plugin init`, you'll get:

```
my-plugin/
├── main.go          # Plugin implementation (stubs)
├── pdk.gen.go       # Generated types from schema
├── xtp.toml         # Plugin configuration
└── go.mod           # Go module (for Go plugins)
```

## Implementing Your Plugin

### 1. Add the Manifest

Every plugin **must** implement `nd_manifest`. This is not in the schemas—add it manually:

```go
import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

type Manifest struct {
    Name        string       `json:"name"`
    Author      string       `json:"author"`
    Version     string       `json:"version"`
    Description string       `json:"description,omitempty"`
    Website     string       `json:"website,omitempty"`
    Permissions *Permissions `json:"permissions,omitempty"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
    manifest := Manifest{
        Name:        "My Plugin",
        Author:      "Your Name",
        Version:     "1.0.0",
        Description: "What this plugin does",
    }
    out, _ := json.Marshal(manifest)
    pdk.Output(out)
    return 0
}
```

### 2. Implement Capability Functions

Replace the generated `panic()` stubs with your implementation:

```go
// Generated stub:
func NdGetArtistBiography(input ArtistInput) (BiographyOutput, error) {
    panic("not implemented")
}

// Your implementation:
func NdGetArtistBiography(input ArtistInput) (BiographyOutput, error) {
    bio := fetchBiography(input.Name)
    return BiographyOutput{Biography: bio}, nil
}
```

### 3. Remove Unused Functions

You don't need to implement all functions from a schema. Delete any you don't need—Navidrome only calls functions that exist.

### 4. Build

```bash
xtp plugin build
```

Or manually with TinyGo:

```bash
tinygo build -o my-plugin.wasm -target wasip1 -buildmode=c-shared .
```

## Combining Capabilities

A single plugin can implement multiple capabilities. Generate from one schema, then manually add functions from others:

```bash
# Start with metadata agent
xtp plugin init --schema-file metadata_agent.yaml --template go --path ./my-plugin --name my-plugin

# Manually add scrobbler functions from scrobbler.yaml
# Manually add scheduler callback from scheduler_callback.yaml
```

Or combine schemas manually before generating (advanced).

## Schema Format

These schemas use [XTP Schema v1-draft](https://docs.xtp.dylibso.com/docs/concepts/xtp-schema) format, which extends JSON Schema with plugin-specific extensions for exports and imports.

## Resources

- [Plugin System Documentation](../README.md)
- [XTP Documentation](https://docs.xtp.dylibso.com/)
- [XTP Schema Reference](https://docs.xtp.dylibso.com/docs/concepts/xtp-schema)
- [Extism PDK](https://extism.org/docs/concepts/pdk)
