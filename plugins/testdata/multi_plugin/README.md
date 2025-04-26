# Multi Service Plugin

This directory contains a test plugin for the Navidrome plugin system, implementing the MetadataAgent interface, which provides both artist and album metadata capabilities, as well as the TimerCallback interface for handling scheduled callbacks.

## Overview

This test plugin is used for ensuring that the Navidrome plugin system correctly handles plugins that implement multiple service interfaces. It provides test implementations for all the required methods, including the timer callback functionality.

## Files

- `plugin.go`: Contains the plugin implementation
- `manifest.json`: Defines the plugin metadata and declares the implemented services

## Services

### Metadata Agent

This plugin returns hardcoded responses for various metadata requests:

- Artist metadata: MBID, URL, biography, similar artists, images, top songs
- Album metadata: album info, album images

### Timer Callback

The plugin also implements the TimerCallback interface, which allows it to:

1. Register timers with the host system
2. Receive callbacks when timers expire
3. Process timer-related payloads

This demonstrates how plugins can implement scheduled or delayed operations even with the stateless plugin architecture.

## Usage

This plugin is used primarily for testing. It provides examples of how to implement various service interfaces in the Navidrome plugin system.

To build the plugin:

```
cd plugins/testdata
make
```

The plugin is not meant for production use and only serves as a testing tool and example for plugin development.
