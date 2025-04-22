# Wikimedia Artist Metadata Plugin

This is a WASM plugin for Navidrome that retrieves artist information from Wikidata/DBpedia using the Wikidata SPARQL endpoint.

## Implemented Methods

- `GetArtistBiography`: Returns the artist's English biography/description from Wikidata.
- `GetArtistURL`: Returns the artist's official website (if available) from Wikidata.
- `GetArtistImages`: Returns the artist's main image (Wikimedia Commons) from Wikidata.

All other methods (`GetArtistMBID`, `GetSimilarArtists`, `GetArtistTopSongs`) return a "not implemented" error, as this data is not available from Wikidata/DBpedia.

## How it Works

- The plugin uses the host-provided HTTP service (`HttpService`) to make SPARQL queries to the Wikidata endpoint.
- No network requests are made directly from the plugin; all HTTP is routed through the host.

## Building

To build the plugin to WASM:

```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm plugin.go
```

## Usage

Copy the resulting `plugin.wasm` to your Navidrome plugins folder under a `wikimedia` directory.

---

For more details, see the source code in `plugin.go`.
