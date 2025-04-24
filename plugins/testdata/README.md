# Plugin Test Data

This directory contains test data and mock implementations used for testing the Navidrome plugin system.

## Contents

- **`fake_*_agent/` directories**: Each of these directories (`fake_multi_agent`, `fake_artist_agent`, `fake_album_agent`) contains the source code for a simple Go plugin that implements a specific agent interface (or multiple interfaces in the case of `fake_multi_agent`). These are compiled into WASM modules using the `Makefile` and used in integration tests for the plugin adapters (e.g., `adapter_media_agent_test.go`).
- **`fake_scrobbler/` directory**: Contains the source code for a mock scrobbler plugin, used for testing scrobbling functionality.
- **`coverartarchive/` directory**: Likely contains cached responses or test fixtures related to the Cover Art Archive integration tests.
- **`wikimedia/` directory**: Likely contains cached responses or test fixtures related to Wikimedia/Wikipedia integration tests.
- **`Makefile`**: Contains build rules for compiling the test plugins located in the subdirectories. Running `make` within this directory will build all test plugins.

## Usage

The primary use of this directory is during the development and testing phase. The `Makefile` is used to build the necessary WASM plugin binaries. The tests within the `plugins` package (and potentially other packages that interact with plugins) then utilize these compiled plugins and other test fixtures found here.
