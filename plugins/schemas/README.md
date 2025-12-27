# Navidrome Plugin Schemas

This directory contains [XTP schemas](https://docs.xtp.dylibso.com/docs/concepts/xtp-schema) 
that define the plugin capabilities for Navidrome. These schemas can be used to bootstrap 
new plugins using the `xtp` CLI tool.

## Available Schemas

| Schema                                             | Description                                                         |
|----------------------------------------------------|---------------------------------------------------------------------|
| [lifecycle.yaml](lifecycle.yaml)                   | Lifecycle callbacks (init) for plugin initialization                |
| [metadata_agent.yaml](metadata_agent.yaml)         | Metadata agent for retrieving artist/album information              |
| [scheduler_callback.yaml](scheduler_callback.yaml) | Scheduler callback for plugins using the scheduler host service     |
| [scrobbler.yaml](scrobbler.yaml)                   | Scrobbler capability for sending play data to external services     |
| [websocket_callback.yaml](websocket_callback.yaml) | WebSocket callbacks for handling messages, errors, and close events |

## Prerequisites

Install the `xtp` CLI tool. See the [XTP CLI documentation](https://docs.xtp.dylibso.com/docs/cli) 
for installation instructions, or install via:

```bash
curl -fsSL https://static.dylibso.com/cli/install.sh | bash
```

## Bootstrapping a Plugin

Use the `xtp plugin init` command to generate boilerplate code from a schema.

### Supported Languages

The XTP CLI supports multiple languages via bindgen templates:
- Go
- Rust  
- TypeScript
- Python
- C#
- Zig
- C++

### Examples

**Create a Go scrobbler plugin:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/scrobbler.yaml \
  --template go \
  --path ./my-scrobbler \
  --name my-scrobbler
```

**Create a Rust metadata agent plugin:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/metadata_agent.yaml \
  --template rust \
  --path ./my-agent \
  --name my-agent
```

**Create a TypeScript scrobbler plugin:**

```bash
xtp plugin init \
  --schema-file plugins/schemas/scrobbler.yaml \
  --template typescript \
  --path ./ts-scrobbler \
  --name ts-scrobbler
```

### Generated Files

After running `xtp plugin init`, you'll get a project structure with:

- `main.go` (or equivalent for your language) - Plugin implementation with stub functions
- `pdk.gen.go` - Generated types from the schema
- `xtp.toml` - Plugin configuration
- Build scripts for your language

### Implementing the Plugin

Edit the generated `main.go` file and replace the `panic()` calls with your implementation.

> **Note:** You don't need to implement all generated functions. Remove any functions that 
> your plugin doesn't need. Navidrome will only call the functions that are exported by your
> plugin, and will gracefully handle missing capabilities.

#### Required: The `nd_manifest` Function

In addition to the capability functions generated from the schema, **every plugin must 
implement the `nd_manifest` function**. This function returns metadata about your plugin
that Navidrome uses to identify and describe it.

**Go example:**

```go
import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

type Manifest struct {
    Name        string `json:"name"`
    Author      string `json:"author"`
    Version     string `json:"version"`
    Description string `json:"description"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
    manifest := Manifest{
        Name:        "My Scrobbler Plugin",
        Author:      "Your Name",
        Version:     "1.0.0",
        Description: "A custom scrobbler for My Service",
    }
    out, err := json.Marshal(manifest)
    if err != nil {
        pdk.SetError(err)
        return 1
    }
    pdk.Output(out)
    return 0
}
```

**Python example:**

```python
import extism

@extism.plugin_fn
def nd_manifest():
    import json
    manifest = {
        "name": "My Scrobbler Plugin",
        "author": "Your Name",
        "version": "1.0.0",
        "description": "A custom scrobbler for My Service"
    }
    extism.output_str(json.dumps(manifest))
```

#### Implementing Capability Functions

Replace the `panic()` calls in the generated stubs with your implementation:

```go
// Example: Implement the IsAuthorized function
func NdScrobblerIsAuthorized(input AuthInput) (AuthOutput, error) {
    // Your authorization logic here
    authorized := checkUserAuthorization(input.UserID, input.Username)
    return AuthOutput{Authorized: authorized}, nil
}
```

### Building the Plugin

Build the plugin to WebAssembly:

```bash
xtp plugin build
```

This creates a `.wasm` file that can be loaded by Navidrome.

## Schema Format

These schemas use the [XTP Schema v1-draft](https://docs.xtp.dylibso.com/docs/concepts/xtp-schema) format,
which is based on JSON Schema with extensions for defining plugin exports and imports.

## Resources

- [XTP Documentation](https://docs.xtp.dylibso.com/)
- [XTP Bindgen Repository](https://github.com/dylibso/xtp-bindgen)
- [XTP Schema Definition](https://raw.githubusercontent.com/dylibso/xtp-bindgen/5090518dd86ba5e734dc225a33066ecc0ed2e12d/plugin/schema.json)
- [Extism Plugin Development Kit](https://extism.org/docs/concepts/pdk)
