# Navidrome Plugin Examples

This folder contains example plugins for Navidrome that demonstrate how to build metadata agents using the plugin system.

## Building

### Prerequisites

- [TinyGo](https://tinygo.org/getting-started/install/) (recommended) or Go 1.23+
- [Extism CLI](https://extism.org/docs/install) (optional, for testing)

### Build all plugins

```bash
make
```

This will compile all example plugins and place the `.wasm` files in this directory.

### Build a specific plugin

```bash
make minimal.wasm
make wikimedia.wasm
```

### Clean build artifacts

```bash
make clean
```

## Available Examples

| Plugin                  | Description                                                   |
|-------------------------|---------------------------------------------------------------|
| [minimal](minimal/)     | A minimal example showing the basic plugin structure          |
| [wikimedia](wikimedia/) | Fetches artist metadata from Wikidata, DBpedia, and Wikipedia |

## Testing with Extism CLI

You can test any plugin using the Extism CLI:

```bash
# Test the manifest
extism call minimal.wasm nd_manifest --wasi

# Test with input
extism call minimal.wasm nd_get_artist_biography --wasi \
  --input '{"id":"1","name":"The Beatles"}'
```

For plugins that make HTTP requests, use `--allow-host` to permit access:

```bash
extism call wikimedia.wasm nd_get_artist_biography --wasi \                                                                                      3s   ▼ 
  --input '{"id":"1","name":"Yussef Dayes"}' \
  --allow-host "query.wikidata.org" --allow-host "en.wikipedia.org"
```

## Installation

Copy any `.wasm` file to your Navidrome plugins folder:

```bash
cp wikimedia.wasm /path/to/navidrome/plugins/
```

Then enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/navidrome/plugins"
```

And add the plugin to your agents list:

```toml
Agents = "lastfm,spotify,wikimedia"
```

## Creating Your Own Plugin

See the [minimal](minimal/) example for the simplest starting point, or [wikimedia](wikimedia/) for a more complete 
example with HTTP requests, created with the [XTP CLI]((https://docs.xtp.dylibso.com/docs/cli).

### Bootstrapping a New Plugin
Use the XTP CLI to bootstrap a new plugin from a schema:

```bash
xtp plugin init \
  --schema-file plugins/schemas/metadata_agent.yaml \
  --template go \
  --path ./my-plugin \
  --name my-plugin
```

See the [schemas README](../schemas/README.md) for more information about available schemas
and supported languages.

For the simplest starting point, look at [minimal](minimal/). For a more complete example
with HTTP requests, see [wikimedia](wikimedia/).


For full documentation, see the [Plugin System README](../README.md).
