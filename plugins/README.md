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
- Cache service (in `plugins/host_cache.go`) for TTL-based plugin caching
- Artwork service (in `plugins/host_artwork.go`) for generating public artwork URLs
- SubsonicAPI service (in `plugins/host_subsonicapi.go`) for accessing Navidrome's Subsonic API

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
  rpc Patch(HttpRequest) returns (HttpResponse);
  rpc Head(HttpRequest) returns (HttpResponse);
  rpc Options(HttpRequest) returns (HttpResponse);
}
```

#### ConfigService

```protobuf
service ConfigService {
    rpc GetPluginConfig(GetPluginConfigRequest) returns (GetPluginConfigResponse);
}
```

The ConfigService allows plugins to access plugin-specific configuration. See the [config.proto](host/config/config.proto) file for the full API.

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

The SchedulerService provides a unified interface for scheduling both one-time and recurring tasks, as well as accessing current time information. See the [scheduler.proto](host/scheduler/scheduler.proto) file for the full API.

```protobuf
service SchedulerService {
   // One-time event scheduling
   rpc ScheduleOneTime(ScheduleOneTimeRequest) returns (ScheduleResponse);

   // Recurring event scheduling
   rpc ScheduleRecurring(ScheduleRecurringRequest) returns (ScheduleResponse);

   // Cancel any scheduled job
   rpc CancelSchedule(CancelRequest) returns (CancelResponse);

   // Get current time in multiple formats
   rpc TimeNow(TimeNowRequest) returns (TimeNowResponse);
}
```

**Key Features:**

- **One-time scheduling**: Schedule a callback to be executed once after a specified delay.
- **Recurring scheduling**: Schedule a callback to be executed repeatedly according to a cron expression.
- **Current time access**: Get the current time in standardized formats for time-based operations.

**TimeNow Function:**

The `TimeNow` function returns the current time in three formats:

```protobuf
message TimeNowResponse {
    string rfc3339_nano = 1;     // RFC3339 format with nanosecond precision
    int64 unix_milli = 2;        // Unix timestamp in milliseconds
    string local_time_zone = 3;  // Local timezone name (e.g., "UTC", "America/New_York")
}
```

This allows plugins to:

- Get high-precision timestamps for logging and event correlation
- Perform time-based calculations using Unix timestamps
- Handle timezone-aware operations by knowing the server's local timezone

Example usage:

```go
// Get current time information
timeResp, err := scheduler.TimeNow(ctx, &scheduler.TimeNowRequest{})
if err != nil {
    return err
}

// Use the different time formats
timestamp := timeResp.Rfc3339Nano     // "2024-01-15T10:30:45.123456789Z"
unixMs := timeResp.UnixMilli          // 1705312245123
timezone := timeResp.LocalTimeZone    // "UTC"
```

Plugins using this service must implement the `SchedulerCallback` interface:

```protobuf
service SchedulerCallback {
    rpc OnSchedulerCallback(SchedulerCallbackRequest) returns (SchedulerCallbackResponse);
}
```

The `IsRecurring` field in the request allows plugins to differentiate between one-time and recurring callbacks.

#### WebSocketService

The WebSocketService enables plugins to connect to and interact with WebSocket endpoints. See the [websocket.proto](host/websocket/websocket.proto) file for the full API.

```protobuf
service WebSocketService {
  // Connect to a WebSocket endpoint
  rpc Connect(ConnectRequest) returns (ConnectResponse);

  // Send a text message
  rpc SendText(SendTextRequest) returns (SendTextResponse);

  // Send binary data
  rpc SendBinary(SendBinaryRequest) returns (SendBinaryResponse);

  // Close a connection
  rpc Close(CloseRequest) returns (CloseResponse);
}
```

- **Connect**: Establish a WebSocket connection to a specified URL with optional headers
- **SendText**: Send text messages over an established connection
- **SendBinary**: Send binary data over an established connection
- **Close**: Close a WebSocket connection with optional close code and reason

Plugins using this service must implement the `WebSocketCallback` interface to handle incoming messages and connection events:

```protobuf
service WebSocketCallback {
  rpc OnTextMessage(OnTextMessageRequest) returns (OnTextMessageResponse);
  rpc OnBinaryMessage(OnBinaryMessageRequest) returns (OnBinaryMessageResponse);
  rpc OnError(OnErrorRequest) returns (OnErrorResponse);
  rpc OnClose(OnCloseRequest) returns (OnCloseResponse);
}
```

Example usage:

```go
// Connect to a WebSocket server
connectResp, err := websocket.Connect(ctx, &websocket.ConnectRequest{
    Url:          "wss://example.com/ws",
    Headers:      map[string]string{"Authorization": "Bearer token"},
    ConnectionId: "my-connection-id",
})
if err != nil {
    return err
}

// Send a text message
_, err = websocket.SendText(ctx, &websocket.SendTextRequest{
    ConnectionId: "my-connection-id",
    Message:      "Hello WebSocket",
})

// Send binary data
_, err = websocket.SendBinary(ctx, &websocket.SendBinaryRequest{
    ConnectionId: "my-connection-id",
    Data:         []byte{0x01, 0x02, 0x03},
})

// Close the connection when done
_, err = websocket.Close(ctx, &websocket.CloseRequest{
    ConnectionId: "my-connection-id",
    Code:         1000, // Normal closure
    Reason:       "Done",
})
```

#### SubsonicAPIService

```protobuf
service SubsonicAPIService {
    rpc Call(CallRequest) returns (CallResponse);
}
```

The SubsonicAPIService provides plugins with access to Navidrome's Subsonic API endpoints. This allows plugins to query and interact with Navidrome's music library data using the same API that external Subsonic clients use.

Key features:

- **Library Access**: Query artists, albums, tracks, playlists, and other music library data
- **Search Functionality**: Search across the music library using various criteria
- **Metadata Retrieval**: Get detailed information about music items including ratings, play counts, etc.
- **Authentication Handled**: The service automatically handles authentication using internal auth context
- **JSON Responses**: All responses are returned as JSON strings for easy parsing

**Important Security Notes:**

- Plugins must specify a username via the `u` parameter in the URL - this determines which user's library view and permissions apply
- The service uses internal authentication, so plugins don't need to provide passwords or API keys
- All Subsonic API security and access controls apply based on the specified user

Example usage:

```go
// Get ping response to test connectivity
resp, err := subsonicAPI.Call(ctx, &subsonicapi.CallRequest{
    Url: "/rest/ping?u=admin",
})
if err != nil {
    return err
}
// resp.Json contains the JSON response

// Search for artists
resp, err = subsonicAPI.Call(ctx, &subsonicapi.CallRequest{
    Url: "/rest/search3?u=admin&query=Beatles&artistCount=10",
})

// Get album details
resp, err = subsonicAPI.Call(ctx, &subsonicapi.CallRequest{
    Url: "/rest/getAlbum?u=admin&id=123",
})

// Check for errors
if resp.Error != "" {
    // Handle error - could be missing parameters, invalid user, etc.
    log.Printf("SubsonicAPI error: %s", resp.Error)
}
```

**Common URL Patterns:**

- `/rest/ping?u=USERNAME` - Test API connectivity
- `/rest/search3?u=USERNAME&query=TERM` - Search library
- `/rest/getArtists?u=USERNAME` - Get all artists
- `/rest/getAlbum?u=USERNAME&id=ID` - Get album details
- `/rest/getPlaylists?u=USERNAME` - Get user playlists

**Required Parameters:**

- `u` (username): Required for all requests - determines user context and permissions
- `f=json`: Recommended to get JSON responses (easier to parse than XML)

The service accepts standard Subsonic API endpoints and parameters. Refer to the [Subsonic API documentation](http://www.subsonic.org/pages/api.jsp) for complete endpoint details, but note that authentication parameters (`p`, `t`, `s`, `c`, `v`) are handled automatically.

See the [subsonicapi.proto](host/subsonicapi/subsonicapi.proto) file for the full API definition.

## Plugin Permission System

Navidrome implements a permission-based security system that controls which host services plugins can access. This system enforces security at load-time by only making authorized services available to plugins in their WebAssembly runtime environment.

### How Permissions Work

The permission system follows a **secure-by-default** approach:

1. **Default Behavior**: Plugins have access to **no host services** unless explicitly declared
2. **Load-time Enforcement**: Only services listed in a plugin's permissions are loaded into its WASM runtime
3. **Runtime Security**: Unauthorized services are completely unavailable - attempts to call them result in "function not exported" errors

This design ensures that even if malicious code tries to access unauthorized services, the calls will fail because the functions simply don't exist in the plugin's runtime environment.

### Permission Syntax

Permissions are declared in the plugin's `manifest.json` file using the `permissions` field as an object:

```json
{
  "name": "my-plugin",
  "author": "Plugin Developer",
  "version": "1.0.0",
  "description": "A plugin that fetches data and caches results",
  "website": "https://github.com/plugindeveloper/my-plugin",
  "capabilities": ["MetadataAgent"],
  "permissions": {
    "http": {
      "reason": "To fetch metadata from external APIs",
      "allowedUrls": {
        "https://api.musicbrainz.org": ["GET"],
        "https://coverartarchive.org": ["GET"]
      },
      "allowLocalNetwork": false
    },
    "cache": {
      "reason": "To cache API responses and reduce rate limiting"
    },
    "subsonicapi": {
      "reason": "To query music library for artist and album information",
      "allowedUsernames": ["metadata-user"],
      "allowAdmins": false
    }
  }
}
```

Each permission is represented as a key in the permissions object. The value must be an object containing a `reason` field that explains why the permission is needed.

**Important**: Some permissions require additional configuration fields:

- **`http`**: Requires `allowedUrls` object mapping URL patterns to allowed HTTP methods, and optional `allowLocalNetwork` boolean
- **`websocket`**: Requires `allowedUrls` array of WebSocket URL patterns, and optional `allowLocalNetwork` boolean
- **`subsonicapi`**: Requires `reason` field, with optional `allowedUsernames` array and `allowAdmins` boolean for fine-grained access control
- **`config`**, **`cache`**, **`scheduler`**, **`artwork`**: Only require the `reason` field

**Security Benefits of Required Reasons:**

- **Transparency**: Users can see exactly what each plugin will do with its permissions
- **Security Auditing**: Makes it easier to identify suspicious or overly broad permission requests
- **Developer Accountability**: Forces plugin authors to justify each permission they request
- **Trust Building**: Clear explanations help users make informed decisions about plugin installation

If no permissions are needed, use an empty permissions object: `"permissions": {}`.

### Available Permissions

The following permission keys correspond to host services:

| Permission    | Host Service       | Description                                        | Required Fields                                       |
| ------------- | ------------------ | -------------------------------------------------- | ----------------------------------------------------- |
| `http`        | HttpService        | Make HTTP requests (GET, POST, PUT, DELETE, etc..) | `reason`, `allowedUrls`                               |
| `websocket`   | WebSocketService   | Connect to and communicate via WebSockets          | `reason`, `allowedUrls`                               |
| `cache`       | CacheService       | Store and retrieve cached data with TTL            | `reason`                                              |
| `config`      | ConfigService      | Access Navidrome configuration values              | `reason`                                              |
| `scheduler`   | SchedulerService   | Schedule one-time and recurring tasks              | `reason`                                              |
| `artwork`     | ArtworkService     | Generate public URLs for artwork images            | `reason`                                              |
| `subsonicapi` | SubsonicAPIService | Access Navidrome's Subsonic API endpoints          | `reason`, optional: `allowedUsernames`, `allowAdmins` |

#### HTTP Permission Structure

HTTP permissions require explicit URL whitelisting for security:

```json
{
  "http": {
    "reason": "To fetch artist data from MusicBrainz and album covers from Cover Art Archive",
    "allowedUrls": {
      "https://musicbrainz.org/ws/2/*": ["GET"],
      "https://coverartarchive.org/*": ["GET"],
      "https://api.example.com/submit": ["POST"]
    },
    "allowLocalNetwork": false
  }
}
```

**Fields:**

- `reason` (required): Explanation of why HTTP access is needed
- `allowedUrls` (required): Object mapping URL patterns to allowed HTTP methods
- `allowLocalNetwork` (optional, default false): Whether to allow requests to localhost/private IPs

**URL Pattern Matching:**

- Exact URLs: `"https://api.example.com/endpoint": ["GET"]`
- Wildcard paths: `"https://api.example.com/*": ["GET", "POST"]`
- Subdomain wildcards: `"https://*.example.com": ["GET"]`

**Important**: Redirect destinations must also be included in `allowedUrls` if you want to follow redirects.

#### WebSocket Permission Structure

WebSocket permissions require explicit URL whitelisting:

```json
{
  "websocket": {
    "reason": "To connect to Discord gateway for real-time Rich Presence updates",
    "allowedUrls": ["wss://gateway.discord.gg", "wss://*.discord.gg"],
    "allowLocalNetwork": false
  }
}
```

**Fields:**

- `reason` (required): Explanation of why WebSocket access is needed
- `allowedUrls` (required): Array of WebSocket URL patterns (must start with `ws://` or `wss://`)
- `allowLocalNetwork` (optional, default false): Whether to allow connections to localhost/private IPs

#### SubsonicAPI Permission Structure

SubsonicAPI permissions control which users plugins can access Navidrome's Subsonic API as, providing fine-grained security controls:

```json
{
  "subsonicapi": {
    "reason": "To query music library data for recommendation engine",
    "allowedUsernames": ["plugin-user", "readonly-user"],
    "allowAdmins": false
  }
}
```

**Fields:**

- `reason` (required): Explanation of why SubsonicAPI access is needed
- `allowedUsernames` (optional): Array of specific usernames the plugin is allowed to use. If empty or omitted, any username can be used
- `allowAdmins` (optional, default false): Whether the plugin can make API calls using admin user accounts

**Security Model:**

The SubsonicAPI service enforces strict user-based access controls:

- **Username Validation**: The plugin must provide a valid `u` (username) parameter in all API calls
- **User Context**: All API responses are filtered based on the specified user's permissions and library access
- **Admin Protection**: By default, plugins cannot use admin accounts for API calls to prevent privilege escalation
- **Username Restrictions**: When `allowedUsernames` is specified, only those users can be used

**Common Permission Patterns:**

```jsonc
// Allow any non-admin user (most permissive)
{
  "subsonicapi": {
    "reason": "To search music library for metadata enhancement",
    "allowAdmins": false
  }
}

// Allow only specific users (most secure)
{
  "subsonicapi": {
    "reason": "To access playlists for synchronization with external service",
    "allowedUsernames": ["sync-user"],
    "allowAdmins": false
  }
}

// Allow admin users (use with caution)
{
  "subsonicapi": {
    "reason": "To perform administrative tasks like library statistics",
    "allowAdmins": true
  }
}

// Restrict to specific users but allow admins
{
  "subsonicapi": {
    "reason": "To backup playlists for authorized users only",
    "allowedUsernames": ["backup-admin", "user1", "user2"],
    "allowAdmins": true
  }
}
```

**Important Notes:**

- Username matching is case-insensitive
- If `allowedUsernames` is empty or omitted, any username can be used (subject to `allowAdmins` setting)
- Admin restriction (`allowAdmins: false`) is checked after username validation
- Invalid or non-existent usernames will result in API call errors

### Permission Validation

The plugin system validates permissions during loading:

1. **Schema Validation**: The manifest is validated against the JSON schema
2. **Permission Recognition**: Unknown permission keys are silently accepted for forward compatibility
3. **Service Loading**: Only services with corresponding permissions are made available to the plugin

### Security Model

The permission system provides multiple layers of security:

#### 1. Principle of Least Privilege

- Plugins start with zero permissions
- Only explicitly requested services are available
- No way to escalate privileges at runtime

#### 2. Load-time Enforcement

- Unauthorized services are not loaded into the WASM runtime
- No performance overhead for permission checks during execution
- Impossible to bypass restrictions through code manipulation

#### 3. Service Isolation

- Each plugin gets its own isolated service instances
- Plugins cannot interfere with each other's service usage
- Host services are sandboxed within the WASM environment

### Best Practices for Plugin Developers

#### Request Minimal Permissions

```jsonc
// Good: No permissions if none needed
{
  "permissions": {}
}

// Good: Only request what you need with clear reasoning
{
  "permissions": {
    "http": {
      "reason": "To fetch artist biography from MusicBrainz database",
      "allowedUrls": {
        "https://musicbrainz.org/ws/2/artist/*": ["GET"]
      },
      "allowLocalNetwork": false
    }
  }
}

// Avoid: Requesting unnecessary permissions
{
  "permissions": {
    "http": {
      "reason": "To fetch data",
      "allowedUrls": {
        "https://*": ["*"]
      },
      "allowLocalNetwork": true
    },
    "cache": {
      "reason": "For caching"
    },
    "scheduler": {
      "reason": "For scheduling"
    },
    "websocket": {
      "reason": "For real-time updates",
      "allowedUrls": ["wss://*"],
      "allowLocalNetwork": true
    }
  }
}
```

#### Write Clear Permission Reasons

Provide specific, descriptive reasons for each permission that explain exactly what the plugin does. Good reasons should:

- Specify **what data** will be accessed/fetched
- Mention **which external services** will be contacted (if applicable)
- Explain **why** the permission is necessary for the plugin's functionality
- Use clear, non-technical language that users can understand

```jsonc
// Good: Specific and informative
{
  "http": {
    "reason": "To fetch album reviews from AllMusic API and artist biographies from MusicBrainz",
    "allowedUrls": {
      "https://www.allmusic.com/api/*": ["GET"],
      "https://musicbrainz.org/ws/2/*": ["GET"]
    },
    "allowLocalNetwork": false
  },
  "cache": {
    "reason": "To cache API responses for 24 hours to respect rate limits and improve performance"
  }
}

// Bad: Vague and unhelpful
{
  "http": {
    "reason": "To make requests",
    "allowedUrls": {
      "https://*": ["*"]
    },
    "allowLocalNetwork": true
  },
  "cache": {
    "reason": "For caching"
  }
}
```

#### Handle Missing Permissions Gracefully

Your plugin should provide clear error messages when permissions are missing:

```go
func (p *Plugin) GetArtistInfo(ctx context.Context, req *api.ArtistInfoRequest) (*api.ArtistInfoResponse, error) {
    // This will fail with "function not exported" if http permission is missing
    resp, err := p.httpClient.Get(ctx, &http.HttpRequest{Url: apiURL})
    if err != nil {
        // Check if it's a permission error
        if strings.Contains(err.Error(), "not exported") {
            return &api.ArtistInfoResponse{
                Error: "Plugin requires 'http' permission (reason: 'To fetch artist metadata from external APIs') - please add to manifest.json",
            }, nil
        }
        return &api.ArtistInfoResponse{Error: err.Error()}, nil
    }
    // ... process response
}
```

### Troubleshooting Permissions

#### Common Error Messages

**"function not exported in module env"**

- Cause: Plugin trying to call a service without proper permission
- Solution: Add the required permission to your manifest.json

**"manifest validation failed" or "missing required field"**

- Cause: Plugin manifest is missing required fields (e.g., `allowedUrls` for HTTP/WebSocket permissions)
- Solution: Ensure your manifest includes all required fields for each permission type

**Permission silently ignored**

- Cause: Using a permission key not recognized by current Navidrome version
- Effect: The unknown permission is silently ignored (no error or warning)
- Solution: This is actually normal behavior for forward compatibility

#### Debugging Permission Issues

1. **Check the manifest**: Ensure required permissions are spelled correctly and present
2. **Verify required fields**: Check that HTTP and WebSocket permissions include `allowedUrls` and other required fields
3. **Review logs**: Check for plugin loading errors, manifest validation errors, and WASM runtime errors
4. **Test incrementally**: Add permissions one at a time to identify which services your plugin needs
5. **Verify service names**: Ensure permission keys match exactly: `http`, `cache`, `config`, `scheduler`, `websocket`, `artwork`, `subsonicapi`
6. **Validate manifest**: Use a JSON schema validator to check your manifest against the schema

### Future Considerations

The permission system is designed for extensibility:

- **Unknown permissions** are allowed in manifests for forward compatibility
- **New services** can be added with corresponding permission keys
- **Permission scoping** could be added in the future (e.g., read-only vs. read-write access)

This ensures that plugins developed today will continue to work as the system evolves, while maintaining strong security boundaries.

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
- `plugins/host/cache/cache.proto`: Cache service interface
- `plugins/host/artwork/artwork.proto`: Artwork service interface
- `plugins/host/subsonicapi/subsonicapi.proto`: SubsonicAPI service interface

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

You can also provide plugin-specific configuration using the `PluginConfig` section. Each plugin can have its own configuration map using the **folder name** as the key:

```toml
[PluginConfig.my-plugin-folder]
api_key = "your-api-key"
user_id = "your-user-id"
enable_feature = "true"

[PluginConfig.another-plugin-folder]
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

**Note**: Plugin identification has changed! Navidrome now uses the **folder name** as the unique identifier for plugins, not the `name` field in `manifest.json`. This means:

- **Multiple plugins can have the same `name` in their manifest**, as long as they are in different folders
- **Plugin loading and commands use the folder name**, not the manifest name
- **Folder names must be unique** across all plugins in your plugins directory

This change allows you to have multiple versions or variants of the same plugin (e.g., `lastfm-official`, `lastfm-custom`, `lastfm-dev`) that all have the same manifest name but coexist peacefully.

### Example: Multiple Plugin Variants

```
plugins/
├── lastfm-official/
│   ├── plugin.wasm
│   └── manifest.json         # {"name": "LastFM Agent", ...}
├── lastfm-custom/
│   ├── plugin.wasm
│   └── manifest.json         # {"name": "LastFM Agent", ...}
└── lastfm-dev/
    ├── plugin.wasm
    └── manifest.json         # {"name": "LastFM Agent", ...}
```

All three plugins can have the same `"name": "LastFM Agent"` in their manifest, but they are identified and loaded by their folder names:

```bash
# Load specific variants
navidrome plugin refresh lastfm-official
navidrome plugin refresh lastfm-custom
navidrome plugin refresh lastfm-dev

# Configure each variant separately
[PluginConfig.lastfm-official]
api_key = "production-key"

[PluginConfig.lastfm-dev]
api_key = "development-key"
```

### Using Symlinks for Plugin Variants

Symlinks provide a powerful way to create multiple configurations for the same plugin without duplicating files. When you create a symlink to a plugin directory, Navidrome treats the symlink as a separate plugin with its own configuration.

**Example: Discord Rich Presence with Multiple Configurations**

```bash
# Create symlinks for different environments
cd /path/to/navidrome/plugins
ln -s /path/to/discord-rich-presence-plugin drp-prod
ln -s /path/to/discord-rich-presence-plugin drp-dev
ln -s /path/to/discord-rich-presence-plugin drp-test
```

Directory structure:

```
plugins/
├── drp-prod -> /path/to/discord-rich-presence-plugin/
├── drp-dev -> /path/to/discord-rich-presence-plugin/
├── drp-test -> /path/to/discord-rich-presence-plugin/
```

Each symlink can have its own configuration:

```toml
[PluginConfig.drp-prod]
clientid = "production-client-id"
users = "admin:prod-token"

[PluginConfig.drp-dev]
clientid = "development-client-id"
users = "admin:dev-token,testuser:test-token"

[PluginConfig.drp-test]
clientid = "test-client-id"
users = "testuser:test-token"
```

**Key Benefits:**

- **Single Source**: One plugin implementation serves multiple use cases
- **Independent Configuration**: Each symlink has its own configuration namespace
- **Development Workflow**: Easy to test different configurations without code changes
- **Resource Sharing**: All symlinks share the same compiled WASM binary

**Important Notes:**

- The **symlink name** (not the target folder name) is used as the plugin ID
- Configuration keys use the symlink name: `PluginConfig.<symlink-name>`
- Each symlink appears as a separate plugin in `navidrome plugin list`
- CLI commands use the symlink name: `navidrome plugin refresh drp-dev`

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

# Remove an installed plugin (use folder name)
navidrome plugin remove plugin-folder-name

# Update an existing plugin
navidrome plugin update /path/to/updated-plugin.ndp

# Reload a plugin without restarting Navidrome (use folder name)
navidrome plugin refresh plugin-folder-name

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
5. Run `navidrome plugin refresh your-plugin-folder-name` to reload the plugin without restarting Navidrome

The `dev` command creates a symlink from your development folder to the plugins directory, allowing you to edit the plugin files directly in your development environment without copying them to the plugins directory after each change.

The refresh process:

- Reloads the plugin manifest
- Recompiles the WebAssembly module
- Updates the plugin registration
- Makes the updated plugin immediately available to Navidrome

### Plugin Security

Navidrome provides multiple layers of security for plugin execution:

1. **WebAssembly Sandbox**: Plugins run in isolated WebAssembly environments with no direct system access
2. **Permission System**: Plugins can only access host services they explicitly request in their manifest (see [Plugin Permission System](#plugin-permission-system))
3. **File System Security**: The plugins folder is configured with restricted permissions (0700) accessible only by the user running Navidrome
4. **Resource Isolation**: Each plugin instance is isolated and cannot interfere with other plugins or core Navidrome functionality

The permission system ensures that plugins follow the principle of least privilege - they start with no access to host services and must explicitly declare what they need. This prevents malicious or poorly written plugins from accessing unauthorized functionality.

Always ensure you trust the source of any plugins you install, and review their requested permissions before installation.

## Plugin Manifest

**Capability Names Are Case-Sensitive**: Entries in the `capabilities` array must exactly match one of the supported capabilities: `MetadataAgent`, `Scrobbler`, `SchedulerCallback`, `WebSocketCallback`, or `LifecycleManagement`.
**Manifest Validation**: The `manifest.json` is validated against the embedded JSON schema (`plugins/schema/manifest.schema.json`). Invalid manifests will be rejected during plugin discovery.

Every plugin must provide a `manifest.json` file that declares metadata, capabilities, and permissions:

```json
{
  "name": "my-awesome-plugin",
  "author": "Your Name",
  "version": "1.0.0",
  "description": "A plugin that does awesome things",
  "website": "https://github.com/yourname/my-awesome-plugin",
  "capabilities": [
    "MetadataAgent",
    "Scrobbler",
    "SchedulerCallback",
    "WebSocketCallback",
    "LifecycleManagement"
  ],
  "permissions": {
    "http": {
      "reason": "To fetch metadata from external music APIs"
    },
    "cache": {
      "reason": "To cache API responses and reduce rate limiting"
    },
    "config": {
      "reason": "To read API keys and service configuration"
    },
    "scheduler": {
      "reason": "To schedule periodic data refresh tasks"
    }
  }
}
```

Required fields:

- `name`: Display name of the plugin (used for documentation/display purposes; folder name is used for identification)
- `author`: The creator or organization behind the plugin
- `version`: Version identifier (recommended to follow semantic versioning)
- `description`: A brief description of what the plugin does
- `website`: Website URL for the plugin documentation, source code, or homepage (must be a valid URI)
- `capabilities`: Array of capability types the plugin implements
- `permissions`: Object mapping host service names to their configurations (use empty object `{}` for no permissions)

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
   - Registers the plugin using the **folder name** as the unique identifier in the plugin registry
3. Plugins can be loaded on-demand by folder name or all at once, depending on the manager's method calls

## Writing a Plugin

### Requirements

1. Your plugin must be compiled to WebAssembly (WASM)
2. Your plugin must implement at least one of the capability interfaces defined in `api.proto`
3. Your plugin must be placed in its own directory with a proper `manifest.json`

### Plugin Registration Functions

The plugin API provides several registration functions that plugins can call during initialization to register capabilities and obtain host services. These functions should typically be called in your plugin's `init()` function.

#### Standard Registration Functions

```go
func RegisterMetadataAgent(agent MetadataAgent)
func RegisterScrobbler(scrobbler Scrobbler)
func RegisterSchedulerCallback(callback SchedulerCallback)
func RegisterLifecycleManagement(lifecycle LifecycleManagement)
func RegisterWebSocketCallback(callback WebSocketCallback)
```

These functions register plugins for the standard capability interfaces:

- **RegisterMetadataAgent**: Register a plugin that provides artist/album metadata and images
- **RegisterScrobbler**: Register a plugin that handles scrobbling to external services
- **RegisterSchedulerCallback**: Register a plugin that handles scheduled callbacks (single callback per plugin)
- **RegisterLifecycleManagement**: Register a plugin that handles initialization and configuration
- **RegisterWebSocketCallback**: Register a plugin that handles WebSocket events

**Basic Usage Example:**

```go
type MyPlugin struct {
    // plugin implementation
}

func init() {
    plugin := &MyPlugin{}

    // Register capabilities your plugin implements
    api.RegisterScrobbler(plugin)
    api.RegisterLifecycleManagement(plugin)
}
```

#### RegisterNamedSchedulerCallback

```go
func RegisterNamedSchedulerCallback(name string, cb SchedulerCallback) scheduler.SchedulerService
```

This function registers a named scheduler callback and returns a scheduler service instance. Named callbacks allow a single plugin to register multiple scheduler callbacks for different purposes, each with its own identifier.

**Parameters:**

- `name` (string): A unique identifier for this scheduler callback within the plugin. This name is used to route scheduled events to the correct callback handler.
- `cb` (SchedulerCallback): An object that implements the `SchedulerCallback` interface

**Returns:**

- `scheduler.SchedulerService`: A scheduler service instance that can be used to schedule one-time or recurring tasks for this specific callback

**Usage Example** (from Discord Rich Presence plugin):

```go
func init() {
    // Register multiple named scheduler callbacks for different purposes
    plugin.sched = api.RegisterNamedSchedulerCallback("close-activity", plugin)
    plugin.rpc.sched = api.RegisterNamedSchedulerCallback("heartbeat", plugin.rpc)
}

// The plugin implements SchedulerCallback to handle "close-activity" events
func (d *DiscordRPPlugin) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
    log.Printf("Removing presence for user %s", req.ScheduleId)
    // Handle close-activity scheduling events
    return nil, d.rpc.clearActivity(ctx, req.ScheduleId)
}

// The rpc component implements SchedulerCallback to handle "heartbeat" events
func (r *discordRPC) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
    // Handle heartbeat scheduling events
    return nil, r.sendHeartbeat(ctx, req.ScheduleId)
}

// Use the returned scheduler service to schedule tasks
func (d *DiscordRPPlugin) NowPlaying(ctx context.Context, request *api.ScrobblerNowPlayingRequest) (*api.ScrobblerNowPlayingResponse, error) {
    // Schedule a one-time callback to clear activity when track ends
    _, err = d.sched.ScheduleOneTime(ctx, &scheduler.ScheduleOneTimeRequest{
        ScheduleId:   request.Username,
        DelaySeconds: request.Track.Length - request.Track.Position + 5,
    })
    return nil, err
}

func (r *discordRPC) connect(ctx context.Context, username string, token string) error {
    // Schedule recurring heartbeats for Discord connection
    _, err := r.sched.ScheduleRecurring(ctx, &scheduler.ScheduleRecurringRequest{
        CronExpression: "@every 41s",
        ScheduleId:     username,
    })
    return err
}
```

**Key Benefits:**

- **Multiple Schedulers**: A single plugin can have multiple named scheduler callbacks for different purposes (e.g., "heartbeat", "cleanup", "refresh")
- **Isolated Scheduling**: Each named callback gets its own scheduler service, allowing independent scheduling management
- **Clear Separation**: Different callback handlers can be implemented on different objects within your plugin
- **Flexible Routing**: The scheduler automatically routes callbacks to the correct handler based on the registration name

**Important Notes:**

- The `name` parameter must be unique within your plugin, but can be the same across different plugins
- The returned scheduler service is specifically tied to the named callback you registered
- Scheduled events will call the `OnSchedulerCallback` method on the object you provided during registration
- You must implement the `SchedulerCallback` interface on the object you register

#### RegisterSchedulerCallback vs RegisterNamedSchedulerCallback

- **Use `RegisterSchedulerCallback`** when your plugin only needs a single scheduler callback
- **Use `RegisterNamedSchedulerCallback`** when your plugin needs multiple scheduler callbacks for different purposes (like the Discord plugin's "heartbeat" and "close-activity" callbacks)

The named version allows better organization and separation of concerns when you have complex scheduling requirements.

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
// Define a connection ID first
connectionID := "my-connection-id"

// Connect to a WebSocket server
connectResp, err := websocket.Connect(ctx, &websocket.ConnectRequest{
    Url:          "wss://example.com/ws",
    Headers:      map[string]string{"Authorization": "Bearer token"},
    ConnectionId: connectionID,
})
if err != nil {
    return err
}

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

## Host Services

Navidrome provides several host services that plugins can use to interact with external systems and access functionality. Plugins must declare permissions for each service they want to use in their `manifest.json`.

### HTTP Service

The HTTP service allows plugins to make HTTP requests to external APIs and services. To use this service, declare the `http` permission in your manifest.

#### Basic Usage

```json
{
  "permissions": {
    "http": {
      "reason": "To fetch artist metadata from external music APIs"
    }
  }
}
```

#### Granular Permissions

For enhanced security, you can specify granular HTTP permissions that restrict which URLs and HTTP methods your plugin can access:

```json
{
  "permissions": {
    "http": {
      "reason": "To fetch album reviews from AllMusic and artist data from MusicBrainz",
      "allowedUrls": {
        "https://api.allmusic.com": ["GET", "POST"],
        "https://*.musicbrainz.org": ["GET"],
        "https://coverartarchive.org": ["GET"],
        "*": ["GET"]
      },
      "allowLocalNetwork": false
    }
  }
}
```

**Permission Fields:**

- `reason` (required): Clear explanation of why HTTP access is needed
- `allowedUrls` (required): Map of URL patterns to allowed HTTP methods

  - Must contain at least one URL pattern
  - For unrestricted access, use: `{"*": ["*"]}`
  - Keys can be exact URLs, wildcard patterns, or `*` for any URL
  - Values are arrays of HTTP methods: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, or `*` for any method
  - **Important**: Redirect destinations must also be included in this list. If a URL redirects to another URL not in `allowedUrls`, the redirect will be blocked.

- `allowLocalNetwork` (optional, default: `false`): Whether to allow requests to localhost/private IPs

**URL Pattern Matching:**

- Exact URLs: `https://api.example.com`
- Wildcard subdomains: `https://*.example.com` (matches any subdomain)
- Wildcard paths: `https://example.com/api/*` (matches any path under /api/)
- Global wildcard: `*` (matches any URL - use with caution)

**Examples:**

```json
// Allow only GET requests to specific APIs
{
  "allowedUrls": {
    "https://api.last.fm": ["GET"],
    "https://ws.audioscrobbler.com": ["GET"]
  }
}

// Allow any method to a trusted domain, GET everywhere else
{
  "allowedUrls": {
    "https://my-trusted-api.com": ["*"],
    "*": ["GET"]
  }
}

// Handle redirects by including redirect destinations
{
  "allowedUrls": {
    "https://short.ly/api123": ["GET"],      // Original URL
    "https://api.actual-service.com": ["GET"] // Redirect destination
  }
}

// Strict permissions for a secure plugin (blocks redirects by not including redirect destinations)
{
  "allowedUrls": {
    "https://api.musicbrainz.org/ws/2": ["GET"]
  },
  "allowLocalNetwork": false
}
```

#### Security Considerations

The HTTP service implements several security features:

1. **Local Network Protection**: By default, requests to localhost and private IP ranges are blocked
2. **URL Filtering**: Only URLs matching `allowedUrls` patterns are allowed
3. **Method Restrictions**: HTTP methods are validated against the allowed list for each URL pattern
4. **Redirect Security**:
   - Redirect destinations must also match `allowedUrls` patterns and methods
   - Maximum of 5 redirects per request to prevent redirect loops
   - To block all redirects, simply don't include any redirect destinations in `allowedUrls`

**Private IP Ranges Blocked (when `allowLocalNetwork: false`):**

- IPv4: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`, `169.254.0.0/16`
- IPv6: `::1`, `fe80::/10`, `fc00::/7`
- Hostnames: `localhost`

#### Making HTTP Requests

```go
import "github.com/navidrome/navidrome/plugins/host/http"

// GET request
resp, err := httpClient.Get(ctx, &http.HttpRequest{
    Url: "https://api.example.com/data",
    Headers: map[string]string{
        "Authorization": "Bearer " + token,
        "User-Agent": "MyPlugin/1.0",
    },
    TimeoutMs: 5000,
})

// POST request with body
resp, err := httpClient.Post(ctx, &http.HttpRequest{
    Url: "https://api.example.com/submit",
    Headers: map[string]string{
        "Content-Type": "application/json",
    },
    Body: []byte(`{"key": "value"}`),
    TimeoutMs: 10000,
})

// Handle response
if err != nil {
    return &api.Response{Error: "HTTP request failed: " + err.Error()}, nil
}

if resp.Error != "" {
    return &api.Response{Error: "HTTP error: " + resp.Error}, nil
}

if resp.Status != 200 {
    return &api.Response{Error: fmt.Sprintf("HTTP %d: %s", resp.Status, string(resp.Body))}, nil
}

// Use response data
data := resp.Body
headers := resp.Headers
```

### Other Host Services

#### Config Service

Access plugin-specific configuration:

```json
{
  "permissions": {
    "config": {
      "reason": "To read API keys and service endpoints from plugin configuration"
    }
  }
}
```

#### Cache Service

Store and retrieve data to improve performance:

```json
{
  "permissions": {
    "cache": {
      "reason": "To cache API responses and reduce external service calls"
    }
  }
}
```

#### Scheduler Service

Schedule recurring or one-time tasks:

```json
{
  "permissions": {
    "scheduler": {
      "reason": "To schedule periodic metadata refresh and cleanup tasks"
    }
  }
}
```

#### WebSocket Service

Connect to WebSocket endpoints:

```json
{
  "permissions": {
    "websocket": {
      "reason": "To connect to real-time music service APIs for live data",
      "allowedUrls": [
        "wss://api.musicservice.com/ws",
        "wss://realtime.example.com"
      ],
      "allowLocalNetwork": false
    }
  }
}
```

#### Artwork Service

Generate public URLs for artwork:

```json
{
  "permissions": {
    "artwork": {
      "reason": "To generate public URLs for album and artist images"
    }
  }
}
```

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

## Caching

The plugin system implements a compilation cache to improve performance:

1. Compiled WASM modules are cached in `[CacheFolder]/plugins`
2. This reduces startup time for plugins that have already been compiled
3. The cache has a automatic cleanup mechanism to remove old modules.
   - when the cache folder exceeds `Plugins.CacheSize` (default 100MB),
     the oldest modules are removed

### WASM Loading Optimization

To improve performance during plugin instance creation, the system implements an optimization that avoids repeated file reads and compilation:

1. **Precompilation**: During plugin discovery, WASM files are read and compiled in the background, with both the MD5 hash of the file bytes and compiled modules cached in memory.

2. **Optimized Runtime**: After precompilation completes, plugins use an `optimizedRuntime` wrapper that overrides `CompileModule` to detect when the same WASM bytes are being compiled by comparing MD5 hashes.

3. **Cache Hit**: When the generated plugin code calls `os.ReadFile()` and `CompileModule()`, the optimization calculates the MD5 hash of the incoming bytes and compares it with the cached hash. If they match, it returns the pre-compiled module directly.

4. **Performance Benefit**: This eliminates repeated compilation while using minimal memory (16 bytes per plugin for the MD5 hash vs potentially MB of WASM bytes), significantly improving plugin instance creation speed while maintaining full compatibility with the generated API code.

5. **Memory Efficiency**: By storing only MD5 hashes instead of full WASM bytes, the optimization scales efficiently regardless of plugin size or count.

The optimization is transparent to plugin developers and automatically activates when plugins are successfully precompiled.

## Best Practices

1. **Resource Management**:

   - The host handles HTTP response cleanup, so no need to close response objects
   - Keep plugin instances lightweight as they are created and destroyed frequently

2. **Error Handling**:

   - Use the standard error types when appropriate
   - Return descriptive error messages for debugging
   - Custom errors are supported and will be propagated to the caller

3. **Performance**:

   - Remember plugins are stateless, so don't rely on local variables for caching. Use the CacheService for caching data.
   - Use efficient algorithms that work well in single-call scenarios

4. **Security**:
   - Only request permissions you actually need (see [Plugin Permission System](#plugin-permission-system))
   - Validate inputs to prevent injection attacks
   - Don't store sensitive credentials in the plugin code
   - Use configuration for API keys and sensitive data

## Limitations

1. WASM plugins have limited access to system resources
2. Plugin compilation has an initial overhead on first load, as it needs to be compiled to WebAssembly
   - Subsequent calls are faster due to caching
3. New plugin capabilities types require changes to the core codebase
4. Stateless nature prevents certain optimizations

## Troubleshooting

1. **Plugin not detected**:

   - Ensure `plugin.wasm` and `manifest.json` exist in the plugin directory
   - Check that the manifest contains valid capabilities names
   - Verify the manifest schema is valid (see [Plugin Permission System](#plugin-permission-system))

2. **Permission errors**:

   - **"function not exported in module env"**: Plugin trying to use a service without proper permission
   - Check that required permissions are declared in `manifest.json`
   - See [Troubleshooting Permissions](#troubleshooting-permissions) for detailed guidance

3. **Compilation errors**:

   - Check logs for WASM compilation errors
   - Verify the plugin is compatible with the current API version

4. **Runtime errors**:
   - Look for error messages in the Navidrome logs
   - Add debug logging to your plugin
   - Check if the error is permission-related before debugging plugin logic
