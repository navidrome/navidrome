# Multi Service Plugin

This directory contains a test plugin for the Navidrome plugin system, implementing the MediaMetadataService interface, which provides both artist and album metadata capabilities.

## Overview

This test plugin is used for ensuring that the Navidrome plugin system correctly handles plugins that implement the MediaMetadataService interface. It provides mock implementations for all the required methods.

## Files

- `plugin.go`: Contains the plugin implementation
- `manifest.json`: Defines the plugin metadata and declares the implemented service

## Usage

This plugin is used primarily for testing. It returns hardcoded responses for various metadata requests:

- Artist metadata: MBID, URL, biography, similar artists, images, top songs
- Album metadata: album info, album images

To build the plugin:

```
cd plugins/testdata
make
```

The plugin is not meant for production use and only serves as a testing tool and example for plugin development.
