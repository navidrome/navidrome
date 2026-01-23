# Navidrome Plugin Examples

This folder contains example plugins demonstrating various capabilities and languages supported by Navidrome's plugin system.

## Available Examples

| Plugin                                                | Language | Capabilities                                    | Description                    |
|-------------------------------------------------------|----------|-------------------------------------------------|--------------------------------|
| [minimal](minimal/)                                   | Go       | MetadataAgent                                   | Basic plugin structure         |
| [wikimedia](wikimedia/)                               | Go       | MetadataAgent                                   | Wikidata/Wikipedia metadata    |
| [crypto-ticker](crypto-ticker/)                       | Go       | Scheduler, WebSocket, Cache                     | Real-time crypto prices (demo) |
| [discord-rich-presence](discord-rich-presence/)       | Go       | Scrobbler, Scheduler, WebSocket, Cache, Artwork | Discord integration            |
| [coverartarchive-py](coverartarchive-py/)             | Python   | MetadataAgent                                   | Cover Art Archive              |
| [nowplaying-py](nowplaying-py/)                       | Python   | Scheduler, SubsonicAPI                          | Now playing logger             |
| [webhook-rs](webhook-rs/)                             | Rust     | Scrobbler                                       | HTTP webhook on scrobble       |
| [library-inspector-rs](library-inspector-rs/)         | Rust     | Library, Scheduler                              | Periodic library stats logging |
| [discord-rich-presence-rs](discord-rich-presence-rs/) | Rust     | Scrobbler, Scheduler, WebSocket, Cache, Artwork | Discord integration (Rust)     |

## Building

### Prerequisites

- **Go plugins:** [TinyGo](https://tinygo.org/getting-started/install/) 0.30+
- **Python plugins:** [extism-py](https://github.com/extism/python-pdk)
- **Rust plugins:** [Rust](https://rustup.rs/) with `wasm32-unknown-unknown` target

### Build All Plugins

```bash
make all
```

This creates `.ndp` package files for each plugin.

### Build Individual Plugin

```bash
make minimal.ndp
make wikimedia.ndp
make discord-rich-presence.ndp
```

### Clean

```bash
make clean
```

## Testing Plugins

### With Extism CLI

Test any plugin without running Navidrome. First extract the `.wasm` file from the `.ndp` package:

```bash
# Install: https://extism.org/docs/install

# Extract the wasm file from the package
unzip -p minimal.ndp plugin.wasm > minimal.wasm

# Test a capability function
extism call minimal.wasm nd_get_artist_biography --wasi \
  --input '{"id":"1","name":"The Beatles"}'
```

For plugins that make HTTP requests, allow the hosts:

```bash
unzip -p wikimedia.ndp plugin.wasm > wikimedia.wasm
extism call wikimedia.wasm nd_get_artist_biography --wasi \
  --input '{"id":"1","name":"Yussef Dayes"}' \
  --allow-host "query.wikidata.org" \
  --allow-host "en.wikipedia.org"
```

### With Navidrome

1. Copy the `.ndp` file to your plugins folder
2. Enable plugins in `navidrome.toml`:
   ```toml
   [Plugins]
   Enabled = true
   Folder = "/path/to/plugins"
   ```
3. For metadata agents, add to your agents list:
   ```toml
   Agents = "lastfm,spotify,wikimedia"
   ```

## Creating Your Own Plugin

### Option 1: Start from Minimal

Copy the [minimal](minimal/) example and modify:

```bash
cp -r minimal my-plugin
cd my-plugin
# Edit main.go and manifest.json
tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
zip -j my-plugin.ndp manifest.json plugin.wasm
```

### Option 2: Bootstrap with XTP CLI

Generate boilerplate from a schema:

```bash
# Install XTP: https://docs.xtp.dylibso.com/docs/cli

xtp plugin init \
  --schema-file ../schemas/metadata_agent.yaml \
  --template go \
  --path ./my-plugin \
  --name my-plugin

# Then create manifest.json and package
cd my-plugin
xtp plugin build
zip -j my-plugin.ndp manifest.json dist/plugin.wasm
```

Available schemas in [../schemas/](../schemas/):
- `metadata_agent.yaml` – Artist/album metadata
- `scrobbler.yaml` – Scrobbling integration
- `lifecycle.yaml` – Init callbacks
- `scheduler_callback.yaml` – Scheduled tasks
- `websocket_callback.yaml` – WebSocket events

### Option 3: Different Language

See language-specific examples:
- **Python:** [coverartarchive-py](coverartarchive-py/)
- **Rust:** [webhook-rs](webhook-rs/)

## Example Breakdown

### Minimal (Go)

The simplest possible plugin. Shows:
- Manifest export
- Single capability function
- Basic input/output handling

### Wikimedia (Go)

Real-world metadata agent. Shows:
- HTTP requests to external APIs
- SPARQL queries (Wikidata)
- Error handling
- Host allowlisting

### Discord Rich Presence (Go)

Complex multi-capability plugin. Shows:
- **Scrobbler** – Receives play events
- **WebSocket** – Maintains Discord gateway connection
- **Scheduler** – Heartbeat and timeout management
- **Cache** – Connection state storage
- **Artwork** – Getting album art URLs

### Cover Art Archive (Python)

Python metadata agent. Shows:
- extism-py plugin structure
- HTTP requests
- JSON handling

### Webhook (Rust)

Rust scrobbler. Shows:
- extism-rs plugin structure
- HTTP POST requests
- Minimal dependencies

## Resources

- [Plugin System Documentation](../README.md)
- [Extism PDK Docs](https://extism.org/docs/concepts/pdk)
- [TinyGo WebAssembly](https://tinygo.org/docs/guides/webassembly/)
- [XTP CLI](https://docs.xtp.dylibso.com/docs/cli)
