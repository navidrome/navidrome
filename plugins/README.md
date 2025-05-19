# Navidrome Plugin System

## Overview

Navidrome's plugin system is a WebAssembly (WASM) based extension mechanism that enables developers to expand Navidrome's functionality without modifying the core codebase. The plugin system supports several capabilities that can be implemented by plugins:

1. **MetadataAgent** - For fetching artist and album information, images, etc.
2. **Scrobbler** - For implementing scrobbling functionality with external services
3. **SchedulerCallback** - For executing code after a specified delay or on a recurring schedule
4. **WebSocketCallback** - For interacting with WebSocket endpoints and handling WebSocket events
5. **LifecycleManagement** - For plugin initialization and configuration (one-time `OnInit` only; not invoked per-request)

## Plugin Architecture

The plugin system is built on the following key components:

### 1. Plugin Manager

The `Manager` (implemented in `plugins/manager.go`) is the core component that:

- Scans for plugins in the configured plugins directory
- Loads and compiles plugins
- Provides access to loaded plugins through capability-specific interfaces

### 2. Plugin Protocol

Plugins communicate with Navidrome using Protocol Buffers (protobuf) over a WASM runtime. The protocol is defined in `plugins/api/api.proto` which specifies the capabilities and messages that plugins can implement.

### 3. Plugin Adapters

Adapters bridge between the plugin API and Navidrome's internal interfaces:

- `wasmMediaAgent` adapts `MetadataAgent` to the internal `agents.Interface`
- `wasmScrobblerPlugin` adapts `Scrobbler` to the internal `scrobbler.Scrobbler`
- `wasmSchedulerCallback` adapts `SchedulerCallback` to the internal `SchedulerCallback`

* **Plugin Instance Pooling**: Instances are managed in an internal pool (default 8 max, 1m TTL).
* **WASM Compilation & Caching**: Modules are pre-compiled concurrently (max 2) and cached in `[CacheFolder]/plugins`, reducing startup time. The compilation timeout can be configured via `DevPluginCompilationTimeout` in development.

### 4. Host Services

Navidrome provides host services that plugins can call to access functionality like HTTP requests and scheduling.
These services are defined in `plugins/host/` and implemented in corresponding host files:

- HTTP service (in `plugins/host_http.go`) for making external requests
- Scheduler service (in `plugins/host_scheduler.go`) for scheduling timed events
- Config service (in `plugins/host_config.go`) for accessing plugin-specific configuration
- WebSocket service (in `plugins/host_websocket.go`) for WebSocket communication

### Available Host Services

The following host services are available to plugins:

#### HttpService

```protobuf
// HTTP methods available to plugins
service HttpService {
  rpc Get(HttpRequest) returns (HttpResponse);
  rpc Post(HttpRequest) returns (HttpResponse);
  rpc Put(HttpRequest) returns (HttpResponse);
  rpc Delete(HttpRequest) returns (HttpResponse);
}
```

#### ConfigService

```protobuf
service ConfigService {
    rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
}
```

The ConfigService allows plugins to access Navidrome's configuration. See the [config.proto](host/config/config.proto) file for the full API.

#### ArtworkService

```protobuf
service ArtworkService {
    rpc GetArtistUrl(GetArtworkUrlRequest) returns (GetArtworkUrlResponse);
    rpc GetAlbumUrl(GetArtworkUrlRequest) returns (GetArtworkUrlResponse);
    rpc GetTrackUrl(GetArtworkUrlRequest) returns (GetArtworkUrlResponse);
}
```

Provides methods to get public URLs for artwork images:

- `GetArtistUrl(id string, size int) string`: Returns a public URL for an artist's artwork
- `GetAlbumUrl(id string, size int) string`: Returns a public URL for an album's artwork
- `GetTrackUrl(id string, size int) string`: Returns a public URL for a track's artwork

The `size` parameter is optional (use 0 for original size). The URLs returned are based on the server's ShareURL configuration.

Example:

```go
url := artwork.GetArtistUrl("123", 300) // Get artist artwork URL with size 300px
url := artwork.GetAlbumUrl("456", 0)    // Get album artwork URL in original size
```

#### CacheService

```protobuf
service CacheService {
    // Set a string value in the cache
    rpc SetString(SetStringRequest) returns (SetResponse);

    // Get a string value from the cache
    rpc GetString(GetRequest) returns (GetStringResponse);

    // Set an integer value in the cache
    rpc SetInt(SetIntRequest) returns (SetResponse);

    // Get an integer value from the cache
    rpc GetInt(GetRequest) returns (GetIntResponse);

    // Set a float value in the cache
    rpc SetFloat(SetFloatRequest) returns (SetResponse);

    // Get a float value from the cache
    rpc GetFloat(GetRequest) returns (GetFloatResponse);

    // Set a byte slice value in the cache
    rpc SetBytes(SetBytesRequest) returns (SetResponse);

    // Get a byte slice value from the cache
    rpc GetBytes(GetRequest) returns (GetBytesResponse);

    // Remove a value from the cache
    rpc Remove(RemoveRequest) returns (RemoveResponse);

    // Check if a key exists in the cache
    rpc Has(HasRequest) returns (HasResponse);
}
```

The CacheService provides a TTL-based cache for plugins. Each plugin gets its own isolated cache instance. By default, cached items expire after 24 hours unless a custom TTL is specified.

Key features:

- **Isolated Caches**: Each plugin has its own cache namespace, so different plugins can use the same key names without conflicts
- **Typed Values**: Store and retrieve values with their proper types (string, int64, float64, or byte slice)
- **Configurable TTL**: Set custom expiration times per item, or use the default 24-hour TTL
- **Type Safety**: The system handles type checking, returning "not exists" if there's a type mismatch

Example usage:

```go
// Store a string value with default TTL (24 hours)
cacheService.SetString(ctx, &cache.SetStringRequest{
    Key:   "user_preference",
    Value: "dark_mode",
})

// Store an integer with custom TTL (5 minutes)
cacheService.SetInt(ctx, &cache.SetIntRequest{
    Key:        "api_call_count",
    Value:      42,
    TtlSeconds: 300, // 5 minutes
})

// Retrieve a value
resp, err := cacheService.GetString(ctx, &cache.GetRequest{
    Key: "user_preference",
})
if err != nil {
    // Handle error
}
if resp.Exists {
    // Use resp.Value
} else {
    // Key doesn't exist or has expired
}

// Check if a key exists
hasResp, err := cacheService.Has(ctx, &cache.HasRequest{
    Key: "api_call_count",
})
if hasResp.Exists {
    // Key exists and hasn't expired
}

// Remove a value
cacheService.Remove(ctx, &cache.RemoveRequest{
    Key: "user_preference",
})
```

See the [cache.proto](host/cache/cache.proto) file for the full API definition.

#### SchedulerService

The SchedulerService provides a unified interface for scheduling both one-time and recurring tasks. See the [scheduler.proto](host/scheduler/scheduler.proto) file for the full API.

- **One-time scheduling**: Schedule a callback to be executed once after a specified delay.
- **Recurring scheduling**: Schedule a callback to be executed repeatedly according to a cron expression.

Plugins using this service must implement the `SchedulerCallback` interface:

```protobuf
service SchedulerCallback {
    rpc OnSchedulerCallback(SchedulerCallbackRequest) returns (SchedulerCallbackResponse);
}
```

The `IsRecurring` field in the request allows plugins to differentiate between one-time and recurring callbacks.

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

This framework significantly simplifies plugin development by handling the low-level details of WebAssembly communication, allowing plugin developers to focus on implementing capabilities interfaces.

### 3. Protocol Buffers (Protobuf)

[Protocol Buffers](https://developers.google.com/protocol-buffers) serve as the interface definition language for the plugin system. Navidrome uses:

- **protoc-gen-go-plugin**: A custom protobuf compiler plugin that generates Go code for both the Navidrome host and WebAssembly plugins
- Protobuf messages for structured data exchange between the host and plugins

The protobuf definitions are located in:

- `plugins/api/api.proto`: Core plugin capability interfaces
- `plugins/host/http/http.proto`: HTTP service interface
- `plugins/host/scheduler/scheduler.proto`: Scheduler service interface
- `plugins/host/config/config.proto`: Config service interface
- `plugins/host/websocket/websocket.proto`: WebSocket service interface

### 4. Integration Architecture

The plugin system integrates these libraries through several key components:

- **Plugin Manager**: Manages the lifecycle of plugins, from discovery to loading
- **Compilation Cache**: Improves performance by caching compiled WebAssembly modules
- **Host Function Bridge**: Exposes Navidrome functionality to plugins through WebAssembly imports
- **Capability Adapters**: Convert between the plugin API and Navidrome's internal interfaces

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

By default, the plugins folder is created under `[DataFolder]/plugins` with restrictive permissions (`0700`) to limit access to the Navidrome user.

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

These configuration values are passed to plugins during initialization through the `OnInit` method in the `LifecycleManagement` capability.
Plugins that implement the `LifecycleManagement` capability will receive their configuration as a map of string keys and values.

## Plugin Directory Structure

Each plugin must be located in its own directory under the plugins folder:

```
plugins/
├── my-plugin/
│   ├── plugin.wasm         # Compiled WebAssembly module
│   └── manifest.json       # Plugin manifest defining metadata and capabilities
├── another-plugin/
│   ├── plugin.wasm
│   └── manifest.json
```

**Note**: The folder name does not need to match the `name` field in `manifest.json`. Navidrome registers plugins by the manifest `name`, which must be unique across all plugins.

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

**Capability Names Are Case-Sensitive**: Entries in the `capabilities` array must exactly match one of the supported capabilities: `MetadataAgent`, `Scrobbler`, `SchedulerCallback`, `WebSocketCallback`, or `LifecycleManagement`.
**Manifest Validation**: The `manifest.json` is validated against the embedded JSON schema (`plugins/schema/manifest.schema.json`). Invalid manifests will be rejected during plugin discovery.

Every plugin must provide a `manifest.json` file that declares metadata and which capabilities it implements:

```json
{
  "name": "my-awesome-plugin",
  "author": "Your Name",
  "version": "1.0.0",
  "description": "A plugin that does awesome things",
  "capabilities": [
    "MetadataAgent",
    "Scrobbler",
    "SchedulerCallback",
    "WebSocketCallback",
    "LifecycleManagement"
  ]
}
```

Required fields:

- `name`: The name of the plugin
- `capabilities`: Array of capability types the plugin implements
- `author`: The creator or organization behind the plugin
- `version`: Version identifier (recommended to follow semantic versioning)
- `description`: A brief description of what the plugin does

Currently supported capabilities:

- `MetadataAgent` - For implementing media metadata providers
- `Scrobbler` - For implementing scrobbling plugins
- `SchedulerCallback` - For implementing timed callbacks
- `WebSocketCallback` - For interacting with WebSocket endpoints and handling WebSocket events
- `LifecycleManagement` - For handling plugin initialization and configuration

## Plugin Loading Process

1. The Plugin Manager scans the plugins directory and all subdirectories
2. For each subdirectory containing a `plugin.wasm` file and valid `manifest.json`, the manager:
   - Validates the manifest and checks for supported capabilities
   - Pre-compiles the WASM module in the background
   - Registers the plugin and its capabilities in the plugin registry
3. Plugins can be loaded on-demand or all at once, depending on the manager's method calls

## Writing a Plugin

### Requirements

1. Your plugin must be compiled to WebAssembly (WASM)
2. Your plugin must implement at least one of the capability interfaces defined in `api.proto`
3. Your plugin must be placed in its own directory with a proper `manifest.json`

### Capability Interfaces

#### Metadata Agent

A capability fetches metadata about artists and albums. Implement this interface to add support for fetching data from external sources.

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

This capability enables scrobbling to external services. Implement this interface to add support for custom scrobblers.

```protobuf
service Scrobbler {
  rpc IsAuthorized(ScrobblerIsAuthorizedRequest) returns (ScrobblerIsAuthorizedResponse);
  rpc NowPlaying(ScrobblerNowPlayingRequest) returns (ScrobblerNowPlayingResponse);
  rpc Scrobble(ScrobblerScrobbleRequest) returns (ScrobblerScrobbleResponse);
}
```

#### Scheduler Callback

This capability allows plugins to receive one-time or recurring scheduled callbacks. Implement this interface to add
support for scheduled tasks. See the [SchedulerService](#scheduler-service) for more information.

```protobuf
service SchedulerCallback {
  rpc OnSchedulerCallback(SchedulerCallbackRequest) returns (SchedulerCallbackResponse);
}
```

#### WebSocket Callback

This capability allows plugins to interact with WebSocket endpoints and handle WebSocket events. Implement this interface to add support for WebSocket-based communication.

```protobuf
service WebSocketCallback {
  // Called when a text message is received
  rpc OnTextMessage(OnTextMessageRequest) returns (OnTextMessageResponse);

  // Called when a binary message is received
  rpc OnBinaryMessage(OnBinaryMessageRequest) returns (OnBinaryMessageResponse);

  // Called when an error occurs
  rpc OnError(OnErrorRequest) returns (OnErrorResponse);

  // Called when the connection is closed
  rpc OnClose(OnCloseRequest) returns (OnCloseResponse);
}
```

Plugins can use the WebSocket host service to connect to WebSocket endpoints, send messages, and handle responses:

```go
// Connect to a WebSocket server
connectResp, err := websocket.Connect(ctx, &websocket.ConnectRequest{
    Url:        "wss://example.com/ws",
    Headers:    map[string]string{"Authorization": "Bearer token"},
    ConnectionId: connectionID,
})
if err != nil {
    return err
}

// Store the connection ID for later use
connectionID := "mu-connection-id"

// Send a text message
_, err = websocket.SendText(ctx, &websocket.SendTextRequest{
    ConnectionId: connectionID,
    Message:      "Hello WebSocket",
})

// Close the connection when done
_, err = websocket.Close(ctx, &websocket.CloseRequest{
    ConnectionId: connectionID,
    Code:         1000, // Normal closure
    Reason:       "Done",
})
```

### Host Functions

Plugins can access host functionality through the host interface:

#### HttpService

```protobuf
// HTTP methods available to plugins
service HttpService {
  rpc Get(HttpRequest) returns (HttpResponse);
  rpc Post(HttpRequest) returns (HttpResponse);
  rpc Put(HttpRequest) returns (HttpResponse);
  rpc Delete(HttpRequest) returns (HttpResponse);
}
```

#### ConfigService

```protobuf
service ConfigService {
    rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
}
```

The ConfigService allows plugins to access Navidrome's configuration. See the [config.proto](host/config/config.proto) file for the full API.

#### CacheService

```protobuf
service CacheService {
    // Set a string value in the cache
    rpc SetString(SetStringRequest) returns (SetResponse);

    // Get a string value from the cache
    rpc GetString(GetRequest) returns (GetStringResponse);

    // Set an integer value in the cache
    rpc SetInt(SetIntRequest) returns (SetResponse);

    // Get an integer value from the cache
    rpc GetInt(GetRequest) returns (GetIntResponse);

    // Set a float value in the cache
    rpc SetFloat(SetFloatRequest) returns (SetResponse);

    // Get a float value from the cache
    rpc GetFloat(GetRequest) returns (GetFloatResponse);

    // Set a byte slice value in the cache
    rpc SetBytes(SetBytesRequest) returns (SetResponse);

    // Get a byte slice value from the cache
    rpc GetBytes(GetRequest) returns (GetBytesResponse);

    // Remove a value from the cache
    rpc Remove(RemoveRequest) returns (RemoveResponse);

    // Check if a key exists in the cache
    rpc Has(HasRequest) returns (HasResponse);
}
```

The CacheService provides a TTL-based cache for plugins. Each plugin gets its own isolated cache instance. By default, cached items expire after 24 hours unless a custom TTL is specified.

Key features:

- **Isolated Caches**: Each plugin has its own cache namespace, so different plugins can use the same key names without conflicts
- **Typed Values**: Store and retrieve values with their proper types (string, int64, float64, or byte slice)
- **Configurable TTL**: Set custom expiration times per item, or use the default 24-hour TTL
- **Type Safety**: The system handles type checking, returning "not exists" if there's a type mismatch

Example usage:

```go
// Store a string value with default TTL (24 hours)
cacheService.SetString(ctx, &cache.SetStringRequest{
    Key:   "user_preference",
    Value: "dark_mode",
})

// Store an integer with custom TTL (5 minutes)
cacheService.SetInt(ctx, &cache.SetIntRequest{
    Key:        "api_call_count",
    Value:      42,
    TtlSeconds: 300, // 5 minutes
})

// Retrieve a value
resp, err := cacheService.GetString(ctx, &cache.GetRequest{
    Key: "user_preference",
})
if err != nil {
    // Handle error
}
if resp.Exists {
    // Use resp.Value
} else {
    // Key doesn't exist or has expired
}

// Check if a key exists
hasResp, err := cacheService.Has(ctx, &cache.HasRequest{
    Key: "api_call_count",
})
if hasResp.Exists {
    // Key exists and hasn't expired
}

// Remove a value
cacheService.Remove(ctx, &cache.RemoveRequest{
    Key: "user_preference",
})
```

See the [cache.proto](host/cache/cache.proto) file for the full API definition.

#### SchedulerService

The SchedulerService provides a unified interface for scheduling both one-time and recurring tasks. See the [scheduler.proto](host/scheduler/scheduler.proto) file for the full API.

- **One-time scheduling**: Schedule a callback to be executed once after a specified delay.
- **Recurring scheduling**: Schedule a callback to be executed repeatedly according to a cron expression.

Plugins using this service must implement the `SchedulerCallback` interface:

```protobuf
service SchedulerCallback {
    rpc OnSchedulerCallback(SchedulerCallbackRequest) returns (SchedulerCallbackResponse);
}
```

The `IsRecurring` field in the request allows plugins to differentiate between one-time and recurring callbacks.

### Error Handling

Plugins should use the standard error values (`plugin:not_found`, `plugin:not_implemented`) to indicate resource-not-found and unimplemented-method scenarios. All other errors will be propagated directly to the caller. Ensure your capability methods return errors via the response message `error` fields rather than panicking or relying on transport errors.

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

**Initialization Semantics**: The `OnInit` call is invoked once when the plugin is first loaded for a given version. If the plugin's version changes, initialization will run again for that new version.

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
3. New plugin capabilities types require changes to the core codebase
4. Stateless nature prevents certain optimizations

## Troubleshooting

1. **Plugin not detected**:

   - Ensure `plugin.wasm` and `manifest.json` exist in the plugin directory
   - Check that the manifest contains valid capabilities names

2. **Compilation errors**:

   - Check logs for WASM compilation errors
   - Verify the plugin is compatible with the current API version

3. **Runtime errors**:
   - Look for error messages in the Navidrome logs
   - Add debug logging to your plugin
