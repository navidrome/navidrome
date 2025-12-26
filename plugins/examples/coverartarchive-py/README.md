# Cover Art Archive Plugin (Python)

This plugin provides album cover images for Navidrome by querying the [Cover Art Archive](https://coverartarchive.org/) API using the MusicBrainz Release MBID.

**This is a Python example** demonstrating that Navidrome's plugin system supports multiple programming languages.

## Features

- Implements the `nd_get_album_images` method of the MetadataAgent plugin interface
- Returns front cover images for a given release MBID
- Returns `not found` if no MBID is provided or no images are found

## Prerequisites

1. **extism-py** - Python to WASM compiler

   Install using the official script:
   ```bash
   curl -Ls https://raw.githubusercontent.com/extism/python-pdk/main/install.sh | bash
   ```

   Or download from [extism/python-pdk releases](https://github.com/extism/python-pdk/releases).

2. **Extism CLI** (optional, for testing)

   ```bash
   # macOS
   brew install extism/tap/extism

   # Or see https://extism.org/docs/install
   ```

## How to Build

```bash
make build
```

Or manually:

```bash
extism-py plugin/__init__.py -o coverartarchive-py.wasm
```

This produces `coverartarchive-py.wasm` in this directory.

## Testing with Extism CLI

Test the manifest:

```bash
extism call coverartarchive-py.wasm nd_manifest --wasi
```

Test album image retrieval (using Portishead's "Dummy" MBID):

```bash
extism call coverartarchive-py.wasm nd_get_album_images --wasi \
  --input '{"name":"Dummy","artist":"Portishead","mbid":"76df3287-6cda-33eb-8e9a-044b5e15ffdd"}' \
  --allow-host "coverartarchive.org" --allow-host "archive.org"
```

Run all tests:

```bash
make test
```

## Installation in Navidrome

1. Build the plugin:
   ```bash
   make build
   ```

2. Copy to your Navidrome plugins folder:
   ```bash
   cp coverartarchive-py.wasm /path/to/navidrome/plugins/
   ```

3. Enable plugins in `navidrome.toml`:
   ```toml
   [Plugins]
   Enabled = true
   Folder = "/path/to/navidrome/plugins"
   ```

4. Add to your agents list:
   ```toml
   Agents = "coverartarchive-py,spotify,lastfm"
   ```

## API Reference

- [Cover Art Archive API](https://musicbrainz.org/doc/Cover_Art_Archive/API)
- Endpoint used: `https://coverartarchive.org/release/{mbid}`
