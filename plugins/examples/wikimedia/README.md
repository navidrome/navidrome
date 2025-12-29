# Wikimedia Plugin for Navidrome

A Navidrome plugin that fetches artist metadata from Wikidata, DBpedia, and Wikipedia.

## Generating the Plugin

This plugin was generated using the XTP CLI:

```bash
xtp plugin init \
  --schema-file plugins/schemas/metadata_agent.yaml \
  --template go \
  --path ./wikimedia \
  --name wikimedia-plugin
```

## Features

- **Artist URL**: Fetches Wikipedia URL for an artist using Wikidata (by MBID or name), DBpedia, or falls back to a Wikipedia search URL
- **Artist Biography**: Fetches the introductory text from the artist's Wikipedia page
- **Artist Images**: Fetches artist images from Wikidata

## Building

### Using TinyGo

```bash
tinygo build -target wasip1 -buildmode=c-shared -o plugin.wasm .
zip -j wikimedia.ndp manifest.json plugin.wasm
```

### Using the Makefile

From the `plugins/examples` directory:

```bash
make wikimedia.ndp
```

### Using XTP CLI

```bash
xtp plugin build
zip -j wikimedia.ndp manifest.json dist/plugin.wasm
```

## Installation

Copy the `.ndp` file to your Navidrome plugins folder:

```bash
cp wikimedia.ndp /path/to/navidrome/plugins/
```

Then enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/navidrome/plugins"
```

Add the plugin to your agents list:

```toml
Agents = "lastfm,spotify,wikimedia"
```

## Testing with Extism CLI

Install the [Extism CLI](https://extism.org/docs/install):

```bash
brew install extism/tap/extism  # macOS
# or see https://extism.org/docs/install for other platforms
```

Extract the wasm file from the package and test:

```bash
# Extract wasm from package
unzip -p wikimedia.ndp plugin.wasm > wikimedia.wasm

# Test artist URL lookup with MBID (The Beatles)
extism call wikimedia.wasm nd_get_artist_url --wasi \
  --input '{"id":"1","name":"The Beatles","mbid":"b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"}' \
  --allow-host "query.wikidata.org"
```

Expected output:
```json
{"url":"https://en.wikipedia.org/wiki/The_Beatles"}
```

### Test artist biography

```bash
extism call wikimedia.wasm nd_get_artist_biography --wasi \
  --input '{"id":"1","name":"The Beatles","mbid":"b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"}' \
  --allow-host "query.wikidata.org" \
  --allow-host "en.wikipedia.org"
```

### Test artist images

```bash
extism call wikimedia.wasm nd_get_artist_images --wasi \
  --input '{"id":"1","name":"The Beatles","mbid":"b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"}' \
  --allow-host "query.wikidata.org"
```

Expected output:
```json
{"images":[{"url":"http://commons.wikimedia.org/wiki/Special:FilePath/Beatles%20ad%201965%20just%20the%20beatles%20crop.jpg","size":0}]}
```

## Project Structure

```
wikimedia/
├── main.go       # Plugin implementation with Wikimedia API logic
├── pdk.gen.go    # Generated types and export wrappers (DO NOT EDIT)
├── go.mod        # Go module file
├── go.sum        # Go module checksums
├── prepare.sh    # Build preparation script
└── xtp.toml      # XTP plugin configuration
```

## API Endpoints Used

| Service   | Endpoint                             | Purpose                                                   |
|-----------|--------------------------------------|-----------------------------------------------------------|
| Wikidata  | `https://query.wikidata.org/sparql`  | SPARQL queries for Wikipedia URLs and images              |
| DBpedia   | `https://dbpedia.org/sparql`         | Fallback SPARQL queries for Wikipedia URLs and short bios |
| Wikipedia | `https://en.wikipedia.org/w/api.php` | MediaWiki API for article extracts                        |

## Implemented Functions

| Function                  | Description                                   |
|---------------------------|-----------------------------------------------|
| `nd_manifest`             | Returns plugin manifest with HTTP permissions |
| `nd_get_artist_url`       | Returns Wikipedia URL for an artist           |
| `nd_get_artist_biography` | Returns artist biography from Wikipedia       |
| `nd_get_artist_images`    | Returns artist image URLs from Wikidata       |
