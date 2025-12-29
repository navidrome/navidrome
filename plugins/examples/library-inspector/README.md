# Library Inspector Plugin

A Navidrome plugin written in Rust that demonstrates the Library host service. It periodically logs details about all configured music libraries and finds the largest file in the root of each library directory.

## Features

- Logs comprehensive library statistics (songs, albums, artists, size, duration)
- Lists the largest file found in each library's root directory
- Configurable inspection interval via cron expression
- Runs an initial inspection on plugin load

## Requirements

- Rust toolchain with `wasm32-wasip1` target
- Navidrome with plugins enabled

## Building

```bash
# Install the WASM target if you haven't already
rustup target add wasm32-wasip1

# Build the plugin
cargo build --target wasm32-wasip1 --release

# Package as .ndp
zip -j library-inspector.ndp manifest.json target/wasm32-wasip1/release/library_inspector.wasm
```

Or use the provided Makefile from the examples directory:

```bash
cd plugins/examples
make library-inspector.ndp
```

## Installation

1. Copy the `.ndp` file to your Navidrome plugins folder
2. Enable plugins in your Navidrome configuration:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/plugins"
```

3. Restart Navidrome and enable the plugin in the UI

## Configuration

Configure the inspection interval in the Navidrome UI (Settings → Plugins → library-inspector):

| Key    | Description                              | Default      |
|--------|------------------------------------------|--------------|
| `cron` | Cron expression for inspection interval  | `@every 1m`  |

## Permissions

This plugin requires:

- **Library** (with filesystem): To read library metadata and scan directories
- **Scheduler**: To schedule periodic inspections

## Example Output

```
=== Library Inspection Started ===
Found 2 libraries
----------------------------------------
Library: My Music (ID: 1)
  Songs:    5432 tracks
  Albums:   456
  Artists:  234
  Size:     45.67 GB
  Duration: 312h 45m
  Mount:    /libraries/1
  Largest file in root: cover.jpg (2.34 MB)
----------------------------------------
Library: Podcasts (ID: 2)
  Songs:    128 tracks
  Albums:   12
  Artists:  8
  Size:     3.21 GB
  Duration: 48h 15m
  Mount:    /libraries/2
  Largest file in root: episode-001.mp3 (156.78 MB)
=== Library Inspection Complete ===
```

## License

GPL-3.0 - Same as Navidrome
