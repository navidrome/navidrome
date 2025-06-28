# Plugin Test Data

This directory contains test data and mock implementations used for testing the Navidrome plugin system.

## Contents

Each of these directories contains the source code for a simple Go plugin that implements a specific agent interface 
(or multiple interfaces in the case of `multi_plugin`). These are compiled into WASM modules using the 
`Makefile` and used in integration tests for the plugin adapters (e.g., `adapter_media_agent_test.go`).

Running `make` within this directory will build all test plugins.

## Usage

The primary use of this directory is during the development and testing phase. The `Makefile` is used to build the 
necessary WASM plugin binaries. The tests within the `plugins` package (and potentially other packages that interact 
with plugins) then utilize these compiled plugins and other test fixtures found here.
