# Plugin Examples

This directory contains example plugins for Navidrome, intended for demonstration and reference purposes. These plugins are not used in automated tests.

## Contents

- `wikimedia/`: Example plugin that retrieves artist information from Wikidata.
- `coverartarchive/`: Example plugin that retrieves album cover images from the Cover Art Archive.

## Building

To build all example plugins, run:

```
make
```

Or to build a specific plugin:

```
make wikimedia
make coverartarchive
```

This will produce the corresponding `plugin.wasm` files in each plugin's directory.
