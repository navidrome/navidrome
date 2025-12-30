# Navidrome Plugin Capabilities

This directory contains the Go interface definitions for Navidrome plugin capabilities. These interfaces are the **source of truth** for plugin development and are used to generate:

1. **Go PDK packages** (`pdk/go/*/`) - Type-safe wrappers for Go plugin developers
2. **XTP YAML schemas** (`*.yaml`) - Schema files for non-Go plugin developers

## For Go Plugin Developers

Go developers should use the generated PDK packages in `plugins/pdk/go/`. See the example plugins in `plugins/examples/` for usage patterns.

## For Non-Go Plugin Developers

If you're developing plugins in other languages (TypeScript, Rust, Python, C#, Zig, C++), you can use the XTP CLI to generate type-safe bindings from the YAML schema files in this directory.

### Prerequisites

Install the XTP CLI:

```bash
# macOS
brew install dylibso/tap/xtp

# Other platforms - see https://docs.xtp.dylibso.com/docs/cli
curl https://static.dylibso.com/cli/install.sh | bash
```

### Generating Plugin Scaffolding

Use the XTP CLI to generate plugin boilerplate from any capability schema:

```bash
# TypeScript
xtp plugin init --schema-file plugins/capabilities/metadata_agent.yaml \
    --template typescript --path my-plugin

# Rust
xtp plugin init --schema-file plugins/capabilities/scrobbler.yaml \
    --template rust --path my-plugin

# Python
xtp plugin init --schema-file plugins/capabilities/lifecycle.yaml \
    --template python --path my-plugin

# C#
xtp plugin init --schema-file plugins/capabilities/scheduler_callback.yaml \
    --template csharp --path my-plugin

# Go (alternative to using the PDK packages)
xtp plugin init --schema-file plugins/capabilities/websocket_callback.yaml \
    --template go --path my-plugin
```

### Available Capabilities

| Capability         | Schema File               | Description                                                 |
|--------------------|---------------------------|-------------------------------------------------------------|
| Metadata Agent     | `metadata_agent.yaml`     | Fetch artist biographies, album images, and similar artists |
| Scrobbler          | `scrobbler.yaml`          | Report listening activity to external services              |
| Lifecycle          | `lifecycle.yaml`          | Plugin initialization callbacks                             |
| Scheduler Callback | `scheduler_callback.yaml` | Scheduled task execution                                    |
| WebSocket Callback | `websocket_callback.yaml` | Real-time WebSocket message handling                        |

### Building Your Plugin

After generating the scaffolding, implement the required functions and build your plugin as a WebAssembly module. The exact build process depends on your chosen language - see the [Extism PDK documentation](https://extism.org/docs/concepts/pdk) for language-specific guides.

## Schema Generation

The YAML schemas are automatically generated from the Go interfaces using `ndpgen`:

```bash
go run ./plugins/cmd/ndpgen -schemas -input=./plugins/capabilities
```

### Technical Note: XTP Schema Compatibility

The generated schemas include `type: object` on object schemas. While this is technically not valid according to the [XTP JSON Schema specification](https://raw.githubusercontent.com/dylibso/xtp-bindgen/5090518dd86ba5e734dc225a33066ecc0ed2e12d/plugin/schema.json), it is **required** as a workaround for XTP's code generator to properly resolve type information (especially for structs with empty properties). XTP tolerates this with a validation warning but generates correct code.

## Resources

- [XTP Documentation](https://docs.xtp.dylibso.com/)
- [XTP Bindgen Repository](https://github.com/dylibso/xtp-bindgen)
- [Extism Plugin Development Kit](https://extism.org/docs/concepts/pdk)
- [XTP Schema Definition](https://raw.githubusercontent.com/dylibso/xtp-bindgen/5090518dd86ba5e734dc225a33066ecc0ed2e12d/plugin/schema.json)
