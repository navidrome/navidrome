# Navidrome Plugin System

## Overview

Navidrome's plugin system is a WebAssembly (WASM) based extension mechanism that enables developers to expand Navidrome's functionality without modifying the core codebase. The plugin system supports several service types that can be implemented by plugins:

1. **Media Metadata Service** - For fetching artist and album information, images, etc.
2. **Scrobbler Service** - For implementing scrobbling functionality with external services

## Plugin Architecture

The plugin system is built on the following key components:

### 1. Plugin Manager

The `Manager` (implemented in `plugins/manager.go`) is the core component that:

- Scans for plugins in the configured plugins directory
- Loads and compiles plugins
- Provides access to loaded plugins through service-specific interfaces

### 2. Plugin Protocol

Plugins communicate with Navidrome using Protocol Buffers (protobuf) over a WASM runtime. The protocol is defined in `plugins/api/api.proto` which specifies the services and messages that plugins can implement.

### 3. Service Adapters

Adapters bridge between the plugin API and Navidrome's internal interfaces:

- `wasmMediaAgent` adapts `MediaMetadataService` to the internal `agents.Interface`
- `wasmScrobblerPlugin` adapts `ScrobblerService` to the internal `scrobbler.Scrobbler`

### 4. Host Services

Navidrome provides host services that plugins can call to access functionality like HTTP requests. These services are defined in `plugins/host/` and implemented in `plugins/host_http.go`.

## Configuration

Plugins are configured in Navidrome's main configuration via the `Plugins` section:

```toml
[Plugins]
# Enable or disable plugin support
Enabled = true

# Directory where plugins are stored (defaults to [DataFolder]/plugins)
Folder = "/path/to/plugins"
```

## Plugin Directory Structure

Each plugin must be located in its own directory under the plugins folder:

```
plugins/
├── my-plugin/
│   ├── plugin.wasm         # Compiled WebAssembly module
│   └── manifest.json       # Plugin manifest defining services
├── another-plugin/
│   ├── plugin.wasm
│   └── manifest.json
```

## Plugin Package Format (.ndp)

Navidrome Plugin Packages (.ndp) are ZIP archives that bundle all files needed for a plugin. They can be installed using the `navidrome plugin install` command.

### Package Structure

A valid .ndp file must contain:

```
plugin-name.ndp (ZIP file)
├── plugin.wasm         # Required: The compiled WebAssembly module
├── manifest.json       # Required: Plugin manifest with metadata
├── README.md           # Optional: Documentation
└── LICENSE             # Optional: License information
```

### Creating a Plugin Package

To create a plugin package:

1. Compile your plugin to WebAssembly (plugin.wasm)
2. Create a manifest.json file with required fields
3. Include any documentation files you want to bundle
4. Create a ZIP archive of all files
5. Rename the ZIP file to have a .ndp extension

### Installing a Plugin Package

Use the Navidrome CLI to install plugins:

```bash
navidrome plugin install /path/to/plugin-name.ndp
```

This will extract the plugin to a directory in your configured plugins folder.

## Plugin Management

Navidrome provides a command-line interface for managing plugins. To use these commands, the plugin system must be enabled in your configuration.

### Available Commands

```bash
# List all installed plugins
navidrome plugin list

# Show detailed information about a plugin package or installed plugin
navidrome plugin info plugin-name-or-package.ndp

# Install a plugin from a .ndp file
navidrome plugin install /path/to/plugin.ndp

# Remove an installed plugin
navidrome plugin remove plugin-name

# Update an existing plugin
navidrome plugin update /path/to/updated-plugin.ndp
```

### Plugin Security

Plugins are executed in a WebAssembly sandbox, but for additional security:

1. The plugins folder is configured with restricted permissions (0700) accessible only by the user running Navidrome
2. All plugin files are also restricted with appropriate permissions
3. Plugins can only access resources through the provided host functions

Always ensure you trust the source of any plugins you install, as they run with the same user permissions as Navidrome itself.

## Plugin Manifest

Every plugin must provide a `manifest.json` file that declares metadata and which services it implements:

```json
{
  "name": "my-awesome-plugin",
  "author": "Your Name",
  "version": "1.0.0",
  "description": "A plugin that does awesome things",
  "services": ["MediaMetadataService", "ScrobblerService"]
}
```

Required fields:

- `name`: The name of the plugin
- `services`: Array of service types the plugin implements
- `author`: The creator or organization behind the plugin
- `version`: Version identifier (recommended to follow semantic versioning)
- `description`: A brief description of what the plugin does

Currently supported service types:

- `MediaMetadataService` - For implementing media metadata providers
- `ScrobblerService` - For implementing scrobbling services

## Plugin Loading Process

1. The Plugin Manager scans the plugins directory and all subdirectories
2. For each subdirectory containing a `plugin.wasm` file and valid `manifest.json`, the manager:
   - Validates the manifest and checks for supported services
   - Pre-compiles the WASM module in the background
   - Registers the plugin and its services in the plugin registry
3. Plugins can be loaded on-demand or all at once, depending on the manager's method calls

## Writing a Plugin

### Requirements

1. Your plugin must be compiled to WebAssembly (WASM)
2. Your plugin must implement at least one of the service interfaces defined in `api.proto`
3. Your plugin must be placed in its own directory with a proper `manifest.json`

### Service Interfaces

#### Media Metadata Service

This service fetches metadata about artists and albums. Implement this interface to add support for fetching data from external sources.

```protobuf
service MediaMetadataService {
  // Artist metadata methods
  rpc GetArtistMBID(ArtistMBIDRequest) returns (ArtistMBIDResponse);
  rpc GetArtistURL(ArtistURLRequest) returns (ArtistURLResponse);
  rpc GetArtistBiography(ArtistBiographyRequest) returns (ArtistBiographyResponse);
  rpc GetSimilarArtists(ArtistSimilarRequest) returns (ArtistSimilarResponse);
  rpc GetArtistImages(ArtistImageRequest) returns (ArtistImageResponse);
  rpc GetArtistTopSongs(ArtistTopSongsRequest) returns (ArtistTopSongsResponse);

  // Album metadata methods
  rpc GetAlbumInfo(AlbumInfoRequest) returns (AlbumInfoResponse);
  rpc GetAlbumImages(AlbumImagesRequest) returns (AlbumImagesResponse);
}
```

#### Scrobbler Service

This service enables scrobbling to external services. Implement this interface to add support for custom scrobblers.

```protobuf
service ScrobblerService {
  rpc IsAuthorized(ScrobblerIsAuthorizedRequest) returns (ScrobblerIsAuthorizedResponse);
  rpc NowPlaying(ScrobblerNowPlayingRequest) returns (ScrobblerNowPlayingResponse);
  rpc Scrobble(ScrobblerScrobbleRequest) returns (ScrobblerScrobbleResponse);
}
```

### Host Functions

Plugins can access host functionality through the host interface, which currently provides HTTP request capabilities:

```protobuf
// HTTP methods available to plugins
service Http {
  rpc Get(HttpRequest) returns (HttpResponse);
  rpc Post(HttpRequest) returns (HttpResponse);
  rpc Put(HttpRequest) returns (HttpResponse);
  rpc Delete(HttpRequest) returns (HttpResponse);
}
```

### Error Handling

Plugins should return standardized errors when certain conditions occur:

```go
ErrNotFound       = errors.New("plugin:not_found")       // When a requested resource isn't found
ErrNotImplemented = errors.New("plugin:not_implemented") // For unimplemented methods
```

However, plugins can also return arbitrary errors when needed, which will be propagated to the caller.

## Plugin Lifecycle and Statelessness

**Important**: Navidrome plugins are stateless. Each method call creates a new plugin instance which is destroyed afterward. This has several important implications:

1. **No in-memory persistence**: Plugins cannot store state between method calls in memory
2. **Each call is isolated**: Variables, configurations, and runtime state don't persist between calls
3. **No shared resources**: Each plugin instance has its own memory space

This stateless design is crucial for security and stability:

- Memory leaks in one call won't affect subsequent operations
- A crashed plugin instance won't bring down the entire system
- Resource usage is more predictable and contained

When developing plugins, keep these guidelines in mind:

- Don't try to cache data in memory between calls
- Don't store authentication tokens or session data in variables
- If persistence is needed, use external storage or the host's HTTP interface
- Performance optimizations should focus on efficient per-call execution

## Caching

The plugin system implements a compilation cache to improve performance:

1. Compiled WASM modules are cached in `[CacheFolder]/plugins`
2. This reduces startup time for plugins that have already been compiled

## Best Practices

1. **Resource Management**:

   - The host handles HTTP response cleanup, so no need to close response objects
   - Keep plugin instances lightweight as they are created and destroyed frequently

2. **Error Handling**:

   - Use the standard error types when appropriate
   - Return descriptive error messages for debugging
   - Custom errors are supported and will be propagated to the caller

3. **Performance**:

   - Remember plugins are stateless, so don't rely on in-memory caching
   - Use efficient algorithms that work well in single-call scenarios

4. **Security**:
   - Validate inputs to prevent injection attacks
   - Don't store sensitive credentials in the plugin code

## Limitations

1. WASM plugins have limited access to system resources
2. Plugin compilation has an initial overhead on first load
3. New plugin service types require changes to the core codebase
4. Stateless nature prevents certain optimizations

## Troubleshooting

1. **Plugin not detected**:

   - Ensure `plugin.wasm` and `manifest.json` exist in the plugin directory
   - Check that the manifest contains valid service names

2. **Compilation errors**:

   - Check logs for WASM compilation errors
   - Verify the plugin is compatible with the current API version

3. **Runtime errors**:
   - Look for error messages in the Navidrome logs
   - Add debug logging to your plugin
