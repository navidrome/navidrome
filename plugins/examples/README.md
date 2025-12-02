# Plugin Examples

This directory contains example plugins for Navidrome, intended for demonstration and reference purposes. These plugins are not used in automated tests.

## Contents

- `wikimedia/`: Retrieves artist information from Wikidata.
- `coverartarchive/`: Fetches album cover images from the Cover Art Archive.
- `crypto-ticker/`: Uses websockets to log real-time cryptocurrency prices.
- `discord-rich-presence/`: Integrates with Discord Rich Presence to display currently playing tracks on Discord profiles.
- `subsonicapi-demo/`: Demonstrates interaction with Navidrome's Subsonic API from a plugin.

## Building

To build all example plugins, run:

```
make
```

Or to build a specific plugin:

```
make wikimedia
make coverartarchive
make crypto-ticker
make discord-rich-presence
make subsonicapi-demo
```

This will produce the corresponding `plugin.wasm` files in each plugin's directory.
