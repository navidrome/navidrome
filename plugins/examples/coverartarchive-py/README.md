# Cover Art Archive Plugin (Python)

A Python example plugin that fetches album cover images from the [Cover Art Archive](https://coverartarchive.org/) API using the MusicBrainz Release MBID.

## Features

- Implements the `nd_get_album_images` method of the MetadataAgent plugin interface
- Returns front cover images for a given release MBID
- Returns `not found` if no MBID is provided or no images are found
- Demonstrates Python plugin development for Navidrome

## Prerequisites

- [extism-py](https://github.com/extism/python-pdk) - Python PDK compiler
  ```bash
  curl -Ls https://raw.githubusercontent.com/extism/python-pdk/main/install.sh | bash
  ```

> **Note:** `extism-py` requires [Binaryen](https://github.com/WebAssembly/binaryen/) (`wasm-merge`, `wasm-opt`) to be installed.

## Building

From the `plugins/examples` directory:

```bash
make coverartarchive-py.wasm
```

Or directly:

```bash
extism-py plugin/__init__.py -o coverartarchive-py.wasm
```

## Installation

1. Copy `coverartarchive-py.wasm` to your Navidrome plugins folder

2. Enable plugins in `navidrome.toml`:
   ```toml
   [Plugins]
   Enabled = true
   Folder = "/path/to/plugins"
   ```

3. Add to your agents list:
   ```toml
   Agents = "coverartarchive-py,spotify,lastfm"
   ```

## Testing

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

## How It Works

1. **Album Image Request (`nd_get_album_images`)**: Receives album metadata including the MusicBrainz Release MBID.

2. **API Query**: Fetches cover art metadata from `https://coverartarchive.org/release/{mbid}`.

3. **Response**: Returns the front cover image URL if found.

## API Reference

- [Cover Art Archive API](https://musicbrainz.org/doc/Cover_Art_Archive/API)
