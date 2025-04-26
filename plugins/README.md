# Navidrome Plugin System

## Overview

Navidrome's plugin system is a WebAssembly (WASM) based extension mechanism that enables developers to expand Navidrome's functionality without modifying the core codebase. The plugin system supports several service types that can be implemented by plugins:

1. **Metadata Agent** - For fetching artist and album information, images, etc.
2. **Scrobbler** - For implementing scrobbling functionality with external services
3. **Timer Callback** - For executing code after a specified delay
4. **Lifecycle Management** - For plugin initialization and configuration

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

- `wasmMediaAgent` adapts `MetadataAgent` to the internal `agents.Interface`
- `wasmScrobblerPlugin` adapts `Scrobbler` to the internal `scrobbler.Scrobbler`
- `wasmTimerCallback` adapts `TimerCallback` for timer callbacks

### 4. Host Services

Navidrome provides host services that plugins can call to access functionality like HTTP requests and timers. These services are defined in `plugins/host/` and implemented in corresponding host files:

- HTTP service (in `plugins/host_http.go`) for making external requests
- Timer service (in `plugins/host_timer.go`) for scheduling delayed callbacks

## Plugin System Implementation

Navidrome's plugin system is built using the following key libraries:

### 1. WebAssembly Runtime (Wazero)

The plugin system uses [Wazero](https://github.com/tetratelabs/wazero), a WebAssembly runtime written in pure Go. Wazero was chosen for several reasons:

- **No CGO dependency**: Unlike other WebAssembly runtimes, Wazero is implemented in pure Go, which simplifies cross-compilation and deployment.
- **Performance**: It provides efficient compilation and caching of WebAssembly modules.
- **Security**: Wazero enforces strict sandboxing, which is important for running third-party plugin code safely.

The plugin manager uses Wazero to:

- Compile and cache WebAssembly modules
- Create isolated runtime environments for each plugin
- Instantiate plugin modules when they're called
- Provide host functions that plugins can call

### 2. Go-plugin Framework

Navidrome builds on [go-plugin](https://github.com/knqyf263/go-plugin), a Go plugin system over WebAssembly that provides:

- **Code generation**: Custom Protocol Buffer compiler plugin (`protoc-gen-go-plugin`) that generates Go code for both the host and WebAssembly plugins
- **Host function system**: Framework for exposing host functionality to plugins safely
- **Interface versioning**: Built-in mechanism for handling API compatibility between the host and plugins
- **Type conversion**: Utilities for marshaling and unmarshaling data between Go and WebAssembly

This framework significantly simplifies plugin development by handling the low-level details of WebAssembly communication, allowing plugin developers to focus on implementing service interfaces.

### 3. Protocol Buffers (Protobuf)

[Protocol Buffers](https://developers.google.com/protocol-buffers) serve as the interface definition language for the plugin system. Navidrome uses:

- **protoc-gen-go-plugin**: A custom protobuf compiler plugin that generates Go code for both the Navidrome host and WebAssembly plugins
- Protobuf messages for structured data exchange between the host and plugins

The protobuf definitions are located in:

- `plugins/api/api.proto`: Core plugin service interfaces
- `plugins/host/http/http.proto`: HTTP service interface
- `plugins/host/timer/timer.proto`: Timer service interface

### 4. Integration Architecture

The plugin system integrates these libraries through several key components:

- **Plugin Manager**: Manages the lifecycle of plugins, from discovery to loading
- **Compilation Cache**: Improves performance by caching compiled WebAssembly modules
- **Host Function Bridge**: Exposes Navidrome functionality to plugins through WebAssembly imports
- **Service Adapters**: Convert between the plugin API and Navidrome's internal interfaces

Each plugin method call:

1. Creates a new isolated plugin instance using Wazero
2. Executes the method in the sandboxed environment
3. Converts data between Go and WebAssembly formats using the protobuf-generated code
4. Cleans up the instance after the call completes

This stateless design ensures that plugins remain isolated and can't interfere with Navidrome's core functionality or each other.

## Configuration

Plugins are configured in Navidrome's main configuration via the `Plugins` section:

```toml
[Plugins]
# Enable or disable plugin support
Enabled = true

# Directory where plugins are stored (defaults to [DataFolder]/plugins)
Folder = "/path/to/plugins"
```

### Plugin-specific Configuration

You can also provide plugin-specific configuration using the `PluginConfig` section. Each plugin can have its own configuration map:

```toml
[PluginConfig.my-plugin-name]
api_key = "your-api-key"
user_id = "your-user-id"
enable_feature = "true"

[PluginConfig.another-plugin]
server_url = "https://example.com/api"
timeout = "30"
```

These configuration values are passed to plugins during initialization through the `OnInit` method in the `InitService` interface.
Plugins that implement the `InitService` will receive their configuration as a map of string keys and values.

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

# Reload a plugin without restarting Navidrome
navidrome plugin refresh plugin-name

# Create a symlink to a plugin development folder
navidrome plugin dev /path/to/dev/folder
```

### Plugin Development

The `dev` and `refresh` commands are particularly useful for plugin development:

#### Development Workflow

1. Create a plugin development folder with required files (`manifest.json` and `plugin.wasm`)
2. Run `navidrome plugin dev /path/to/your/plugin` to create a symlink in the plugins directory
3. Make changes to your plugin code
4. Recompile the WebAssembly module
5. Run `navidrome plugin refresh your-plugin-name` to reload the plugin without restarting Navidrome

The `dev` command creates a symlink from your development folder to the plugins directory, allowing you to edit the plugin files directly in your development environment without copying them to the plugins directory after each change.

The refresh process:

- Reloads the plugin manifest
- Recompiles the WebAssembly module
- Updates the plugin registration
- Makes the updated plugin immediately available to Navidrome

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
  "services": [
    "MetadataAgent",
    "Scrobbler",
    "TimerCallback",
    "LifecycleManagement"
  ]
}
```

Required fields:

- `name`: The name of the plugin
- `services`: Array of service types the plugin implements
- `author`: The creator or organization behind the plugin
- `version`: Version identifier (recommended to follow semantic versioning)
- `description`: A brief description of what the plugin does

Currently supported service types:

- `MetadataAgent` - For implementing media metadata providers
- `Scrobbler` - For implementing scrobbling services
- `TimerCallback` - For implementing plugins that use the timer service
- `LifecycleManagement` - For handling plugin initialization and configuration

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

#### Metadata Agent

This service fetches metadata about artists and albums. Implement this interface to add support for fetching data from external sources.

```protobuf
service MetadataAgent {
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

#### Scrobbler

This service enables scrobbling to external services. Implement this interface to add support for custom scrobblers.

```protobuf
service Scrobbler {
  rpc IsAuthorized(ScrobblerIsAuthorizedRequest) returns (ScrobblerIsAuthorizedResponse);
  rpc NowPlaying(ScrobblerNowPlayingRequest) returns (ScrobblerNowPlayingResponse);
  rpc Scrobble(ScrobblerScrobbleRequest) returns (ScrobblerScrobbleResponse);
}
```

#### Timer Callback

This service allows plugins to receive timer callbacks after a specified delay. Implement this interface to add support for delayed operations.

```protobuf
service TimerCallback {
  rpc OnTimerCallback(TimerCallbackRequest) returns (TimerCallbackResponse);
}
```

### Host Functions

Plugins can access host functionality through the host interface:

```protobuf
// HTTP methods available to plugins
service Http {
  rpc Get(HttpRequest) returns (HttpResponse);
  rpc Post(HttpRequest) returns (HttpResponse);
  rpc Put(HttpRequest) returns (HttpResponse);
  rpc Delete(HttpRequest) returns (HttpResponse);
}

// Timer methods available to plugins
service TimerService {
  rpc RegisterTimer(TimerRequest) returns (TimerResponse);
  rpc CancelTimer(CancelTimerRequest) returns (CancelTimerResponse);
}
```

The Timer service allows plugins to:

- Register timers that will trigger a callback after a specified delay
- Cancel previously registered timers
- Receive callbacks through the `OnTimerCallback` method when timers expire

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

### Using Plugin Configuration

Since plugins are stateless, you can use the `LifecycleManagement` interface to read configuration when your plugin is loaded and perform any necessary setup:

```go
func (p *myPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
    // Access plugin configuration
    apiKey := req.Config["api_key"]
    if apiKey == "" {
        return &api.InitResponse{Error: "Missing API key in configuration"}, nil
    }

    // Validate configuration
    serverURL := req.Config["server_url"]
    if serverURL == "" {
        serverURL = "https://default-api.example.com" // Use default if not specified
    }

    // Perform initialization tasks (e.g., validate API key)
    httpClient := &http.HttpServiceClient{}
    resp, err := httpClient.Get(ctx, &http.HttpRequest{
        Url: serverURL + "/validate?key=" + apiKey,
    })
    if err != nil {
        return &api.InitResponse{Error: "Failed to validate API key: " + err.Error()}, nil
    }

    if resp.StatusCode != 200 {
        return &api.InitResponse{Error: "Invalid API key"}, nil
    }

    return &api.InitResponse{}, nil
}
```

Remember, the `OnInit` method is called only once when the plugin is loaded. It cannot store any state that needs to persist between method calls. It's primarily useful for:

1. Validating required configuration
2. Checking API credentials
3. Verifying connectivity to external services
4. Initializing any external resources

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
