# Navidrome Plugin System

Navidrome supports WebAssembly (Wasm) plugins for extending functionality. Plugins run in a secure sandbox and can provide metadata agents, scrobblers, and other integrations through host services like scheduling, caching, WebSockets, and Subsonic API access.

The plugin system is built on **[Extism](https://extism.org/)**, a cross-language framework for building WebAssembly plugins. This means you can write plugins in any language that Extism supports (Go, Rust, Python, TypeScript, and more) using their Plugin Development Kits (PDKs).

**Essential Extism Resources:**
- [Extism Documentation](https://extism.org/docs/overview) – Core concepts and architecture
- [Plugin Development Kits (PDKs)](https://extism.org/docs/concepts/pdk) – Language-specific libraries for writing plugins
- [Go PDK](https://github.com/extism/go-pdk) – Recommended for Go plugins with TinyGo
- [Rust PDK](https://github.com/extism/rust-pdk) – For Rust plugins
- [Python PDK](https://github.com/extism/python-pdk) – Experimental Python support
- [JavaScript PDK](https://github.com/extism/js-pdk) – For TypeScript/JavaScript plugins

## Table of Contents

- [Quick Start](#quick-start)
- [Plugin Basics](#plugin-basics)
- [Capabilities](#capabilities)
  - [MetadataAgent](#metadataagent)
  - [Scrobbler](#scrobbler)
  - [Lifecycle](#lifecycle)
- [Host Services](#host-services)
  - [HTTP Requests](#http-requests)
  - [Scheduler](#scheduler)
  - [Cache](#cache)
  - [KVStore](#kvstore)
  - [WebSocket](#websocket)
  - [Library](#library)
  - [Artwork](#artwork)
  - [SubsonicAPI](#subsonicapi)
  - [Config](#config)
  - [Users](#users)
- [Configuration](#configuration)
- [Building Plugins](#building-plugins)
- [Examples](#examples)
- [Security](#security)

---

## Quick Start

### 1. Create a minimal plugin

Create `main.go`:

```go
package main

import "github.com/extism/go-pdk"

func main() {}

// Implement your capability functions here
```

Create `manifest.json`:

```json
{
    "name": "My Plugin",
    "author": "Your Name",
    "version": "1.0.0"
}
```

### 2. Build with TinyGo and package as .ndp

```bash
# Compile to WebAssembly
tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .

# Package as .ndp (zip archive)
zip -j my-plugin.ndp manifest.json plugin.wasm
```

### 3. Install

Copy `my-plugin.ndp` to your Navidrome plugins folder and enable plugins in your config:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/plugins"
```

---

## Plugin Basics

### What is a Plugin?

A Navidrome plugin is an `.ndp` package file (zip archive) containing:

1. **`manifest.json`** – Plugin metadata (name, author, version, permissions)
2. **`plugin.wasm`** – Compiled WebAssembly module with capability functions

### Plugin Package Structure

```
my-plugin.ndp (zip archive)
├── manifest.json    # Required: Plugin metadata
└── plugin.wasm      # Required: Compiled WebAssembly module
```

### Plugin Naming

Plugins are identified by their **filename** (without `.ndp` extension), not the manifest `name` field:

- `my-plugin.ndp` → plugin ID is `my-plugin`
- The manifest `name` is the display name shown in the UI

This allows users to have multiple instances of the same plugin with different configs by renaming the files.

### The Manifest

Every plugin must include a `manifest.json` file. Example:

```json
{
  "name": "My Plugin",
  "author": "Author Name",
  "version": "1.0.0",
  "description": "What this plugin does",
  "website": "https://example.com",
  "permissions": {
    "http": {
      "reason": "Fetch metadata from external API",
      "requiredHosts": ["api.example.com", "*.musicbrainz.org"]
    }
  }
}
```

**Required fields:** `name`, `author`, `version`

#### Experimental Features

Plugins can opt-in to experimental WebAssembly features that may change or be removed in future versions. Currently supported:

- **`threads`** – Enables WebAssembly threads support (for plugins compiled with multi-threading)

```json
{
  "name": "Threaded Plugin",
  "author": "Author Name",
  "version": "1.0.0",
  "experimental": {
    "threads": {
      "reason": "Required for concurrent audio processing"
    }
  }
}
```

> **Note:** Experimental features may have compatibility or performance implications. Use only when necessary.

---

## Capabilities

Capabilities define what your plugin can do. They're automatically detected based on which functions you export.

### MetadataAgent

Provides artist and album metadata. Export one or more of these functions:

| Function                  | Input                      | Output                           | Description          |
|---------------------------|----------------------------|----------------------------------|----------------------|
| `nd_get_artist_mbid`      | `{id, name}`               | `{mbid}`                         | Get MusicBrainz ID   |
| `nd_get_artist_url`       | `{id, name, mbid?}`        | `{url}`                          | Get artist URL       |
| `nd_get_artist_biography` | `{id, name, mbid?}`        | `{biography}`                    | Get artist biography |
| `nd_get_similar_artists`  | `{id, name, mbid?, limit}` | `{artists: [{name, mbid?}]}`     | Get similar artists  |
| `nd_get_artist_images`    | `{id, name, mbid?}`        | `{images: [{url, size}]}`        | Get artist images    |
| `nd_get_artist_top_songs` | `{id, name, mbid?, count}` | `{songs: [{name, mbid?}]}`       | Get top songs        |
| `nd_get_album_info`       | `{name, artist, mbid?}`    | `{name, mbid, description, url}` | Get album info       |
| `nd_get_album_images`     | `{name, artist, mbid?}`    | `{images: [{url, size}]}`        | Get album images     |

**Example:**

```go
type ArtistInput struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    MBID string `json:"mbid,omitempty"`
}

type BiographyOutput struct {
    Biography string `json:"biography"`
}

//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
    var input ArtistInput
    if err := pdk.InputJSON(&input); err != nil {
        pdk.SetError(err)
        return 1
    }

    // Fetch biography from your data source...
    output := BiographyOutput{Biography: "Artist biography..."}
    pdk.OutputJSON(output)
    return 0
}
```

To use the plugin as a metadata agent, add it to your config:

```toml
Agents = "lastfm,spotify,my-plugin"
```

### Scrobbler

Integrates with external scrobbling services. Export one or more of these functions:

| Function                     | Input                 | Output         | Description                 |
|------------------------------|-----------------------|----------------|-----------------------------|
| `nd_scrobbler_is_authorized` | `{username}`          | `bool`         | Check if user is authorized |
| `nd_scrobbler_now_playing`   | See below             | (none)         | Send now playing            |
| `nd_scrobbler_scrobble`      | See below             | (none)         | Submit a scrobble           |

> **Important:** Scrobbler plugins require the `users` permission in their manifest. Scrobble events are only sent for users assigned to the plugin through Navidrome's configuration. The `nd_scrobbler_is_authorized` function is called after the server-side user check passes.

**Manifest permission:**

```json
{
  "permissions": {
    "users": {
      "reason": "Receive scrobble events for users assigned to this plugin"
    }
  }
}
```

**NowPlaying/Scrobble Input:**

```json
{
  "username": "john",
  "track": {
    "id": "track-id",
    "title": "Song Title",
    "album": "Album Name",
    "artist": "Artist Name",
    "albumArtist": "Album Artist",
    "duration": 180.5,
    "trackNumber": 1,
    "discNumber": 1,
    "mbzRecordingId": "...",
    "mbzAlbumId": "...",
    "mbzArtistId": "..."
  },
  "timestamp": 1703270400
}
```

**Error Handling:**

On success, return `0`. On failure, use `pdk.SetError()` with one of these error types:

- `scrobbler(not_authorized)` – User needs to re-authorize
- `scrobbler(retry_later)` – Temporary failure, Navidrome will retry
- `scrobbler(unrecoverable)` – Permanent failure, scrobble discarded

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"

// Return error using predefined constants
return scrobbler.ScrobblerErrorNotAuthorized
return scrobbler.ScrobblerErrorRetryLater
return scrobbler.ScrobblerErrorUnrecoverable
```

### Lifecycle

Optional initialization callback. Export this function to run code when your plugin loads:

| Function     | Input | Output     | Description                    |
|--------------|-------|------------|--------------------------------|
| `nd_on_init` | `{}`  | `{error?}` | Called once after plugin loads |

Useful for initializing connections, scheduling recurring tasks, etc.

---

## Host Services

Host services let your plugin call back into Navidrome for advanced functionality. Each service requires declaring the permission in your manifest.

### HTTP Requests

Make HTTP requests using the Extism PDK's built-in HTTP support. See your [Extism PDK documentation](https://extism.org/docs/concepts/pdk) for more details on making requests.

**Manifest permission:**

```json
{
  "permissions": {
    "http": {
      "reason": "Fetch metadata from external API",
      "requiredHosts": ["api.example.com", "*.musicbrainz.org"]
    }
  }
}
```

**Usage:**

```go
req := pdk.NewHTTPRequest(pdk.MethodGet, "https://api.example.com/data")
req.SetHeader("Authorization", "Bearer " + apiKey)
resp := req.Send()

if resp.Status() == 200 {
    data := resp.Body()
    // Process response...
}
```

### Scheduler

Schedule one-time or recurring tasks. Your plugin must export `nd_scheduler_callback` to receive events.

**Manifest permission:**

```json
{
  "permissions": {
    "scheduler": {
      "reason": "Schedule periodic metadata refresh"
    }
  }
}
```

**Host functions:**

| Function                      | Parameters                               | Description                 |
|-------------------------------|------------------------------------------|-----------------------------|
| `scheduler_scheduleonetime`   | `delaySeconds, payload, scheduleId?`     | Schedule one-time callback  |
| `scheduler_schedulerecurring` | `cronExpression, payload, scheduleId?`   | Schedule recurring callback |
| `scheduler_cancelschedule`    | `scheduleId`                             | Cancel a scheduled task     |

**Callback function:**

```go
type SchedulerCallbackInput struct {
    ScheduleID  string `json:"scheduleId"`
    Payload     string `json:"payload"`
    IsRecurring bool   `json:"isRecurring"`
}

//go:wasmexport nd_scheduler_callback
func ndSchedulerCallback() int32 {
    var input SchedulerCallbackInput
    pdk.InputJSON(&input)

    // Handle the scheduled task based on payload
    pdk.Log(pdk.LogInfo, "Task fired: " + input.ScheduleID)
    return 0
}
```

**Scheduling tasks (using generated SDK):**

Add the generated SDK to your `go.mod`:

```
require github.com/navidrome/navidrome/plugins/pdk/go v0.0.0
replace github.com/navidrome/navidrome/plugins/pdk/go => ../../pdk/go
```

Then import and use:

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Schedule one-time task in 60 seconds
scheduleID, err := host.SchedulerScheduleOneTime(60, "my-payload", "")

// Schedule recurring task with cron expression (every hour)
scheduleID, err := host.SchedulerScheduleRecurring("0 * * * *", "hourly-task", "")

// Cancel a task
err := host.SchedulerCancelSchedule(scheduleID)
```

### Cache

Store and retrieve data in an in-memory TTL-based cache. Each plugin has its own isolated namespace.

**Manifest permission:**

```json
{
  "permissions": {
    "cache": {
      "reason": "Cache API responses to reduce external requests"
    }
  }
}
```

**Host functions:**

| Function          | Parameters                | Description           |
|-------------------|---------------------------|-----------------------|
| `cache_setstring` | `key, value, ttl_seconds` | Store a string        |
| `cache_getstring` | `key`                     | Get a string          |
| `cache_setint`    | `key, value, ttl_seconds` | Store an integer      |
| `cache_getint`    | `key`                     | Get an integer        |
| `cache_setfloat`  | `key, value, ttl_seconds` | Store a float         |
| `cache_getfloat`  | `key`                     | Get a float           |
| `cache_setbytes`  | `key, value, ttl_seconds` | Store bytes           |
| `cache_getbytes`  | `key`                     | Get bytes             |
| `cache_has`       | `key`                     | Check if key exists   |
| `cache_remove`    | `key`                     | Delete a cached value |

**TTL:** Pass `0` for the default (24 hours), or specify seconds.

**Usage (with generated SDK):**

Import the Go SDK (see [Scheduler](#scheduler) for `go.mod` setup):

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Cache a value for 1 hour
host.CacheSetString("api-response", responseData, 3600)

// Retrieve (check Exists before using Value)
result, err := host.CacheGetString("api-response")
if result.Exists {
    data := result.Value
}
```

> **Note:** Cache is in-memory only and cleared on server restart.

### KVStore

Persistent key-value storage that survives server restarts. Each plugin has its own isolated SQLite database.

**Manifest permission:**

```json
{
  "permissions": {
    "kvstore": {
      "reason": "Store OAuth tokens and plugin state",
      "maxSize": "1MB"
    }
  }
}
```

**Permission options:**
- `maxSize`: Maximum storage size (e.g., `"1MB"`, `"500KB"`). Default: 1MB

**Host functions:**

| Function                 | Parameters   | Description                       |
|--------------------------|--------------|-----------------------------------|
| `kvstore_set`            | `key, value` | Store a byte value                |
| `kvstore_get`            | `key`        | Retrieve a byte value             |
| `kvstore_delete`         | `key`        | Delete a value                    |
| `kvstore_has`            | `key`        | Check if key exists               |
| `kvstore_list`           | `prefix`     | List keys matching prefix         |
| `kvstore_getstorageused` | -            | Get current storage usage (bytes) |

**Key constraints:**
- Maximum key length: 256 bytes
- Keys must be valid UTF-8 strings

**Usage (with generated SDK):**

Import the Go SDK (see [Scheduler](#scheduler) for `go.mod` setup):

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Store a value (as raw bytes)
token := []byte(`{"access_token": "xyz", "refresh_token": "abc"}`)
_, err := host.KVStoreSet("oauth:spotify", token)

// Retrieve a value
result, err := host.KVStoreGet("oauth:spotify")
if result.Exists {
    var tokenData map[string]string
    json.Unmarshal(result.Value, &tokenData)
}

// List all keys with prefix
keysResult, err := host.KVStoreList("user:")
for _, key := range keysResult.Keys {
    // Process each key
}

// Check storage usage
usageResult, err := host.KVStoreGetStorageUsed()
fmt.Printf("Using %d bytes\n", usageResult.Bytes)

// Delete a value
host.KVStoreDelete("oauth:spotify")
```

> **Note:** Unlike Cache, KVStore data persists across server restarts. Storage is located at `${DataFolder}/plugins/${pluginID}/kvstore.db`.

### WebSocket

Establish persistent WebSocket connections to external services.

**Manifest permission:**

```json
{
  "permissions": {
    "websocket": {
      "reason": "Real-time connection to service",
      "requiredHosts": ["gateway.example.com", "*.discord.gg"]
    }
  }
}
```

**Host functions:**

| Function               | Parameters                      | Description       |
|------------------------|---------------------------------|-------------------|
| `websocket_connect`    | `url, headers?, connectionId?`  | Open a connection |
| `websocket_sendtext`   | `connectionId, message`         | Send text message |
| `websocket_sendbinary` | `connectionId, data`            | Send binary data  |
| `websocket_close`      | `connectionId, code?, reason?`  | Close connection  |

**Callback functions (export these to receive events):**

| Function                         | Input                           | Description                      |
|----------------------------------|---------------------------------|----------------------------------|
| `nd_websocket_on_text_message`   | `{connectionId, message}`       | Text message received            |
| `nd_websocket_on_binary_message` | `{connectionId, data}`          | Binary message received (base64) |
| `nd_websocket_on_error`          | `{connectionId, error}`         | Connection error                 |
| `nd_websocket_on_close`          | `{connectionId, code, reason}`  | Connection closed                |

### Library

Access music library metadata and optionally read files from library directories.

**Manifest permission:**

```json
{
  "permissions": {
    "library": {
      "reason": "Access library metadata for analysis",
      "filesystem": false
    }
  }
}
```

- `filesystem` – Set to `true` to enable read-only access to library directories (default: `false`)

**Host functions:**

| Function                   | Parameters | Returns                   |
|----------------------------|------------|---------------------------|
| `library_getlibrary`       | `id`       | Library metadata          |
| `library_getalllibraries`  | (none)     | Array of library metadata |

**Library metadata:**

```json
{
  "id": 1,
  "name": "My Music",
  "path": "/music/collection",
  "mountPoint": "/libraries/1",
  "lastScanAt": 1703270400,
  "totalSongs": 5000,
  "totalAlbums": 500,
  "totalArtists": 200,
  "totalSize": 50000000000,
  "totalDuration": 1500000.5
}
```

> **Note:** The `path` and `mountPoint` fields are only included when `filesystem: true` is set in the permission.

**Filesystem access:**

When `filesystem: true`, your plugin can read files from library directories via WASI filesystem APIs. Each library is mounted at `/libraries/<id>`:

```go
import "os"

// Read a file from library 1
content, err := os.ReadFile("/libraries/1/Artist/Album/track.mp3")

// List directory contents
entries, err := os.ReadDir("/libraries/1/Artist")
```

> **Security:** Filesystem access is read-only and restricted to configured library paths only. Plugins cannot access other parts of the host filesystem.

**Usage (with generated SDK):**

Import the Go SDK (see [Scheduler](#scheduler) for `go.mod` setup). The `Library` struct is provided by the SDK:

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Get a specific library
resp, err := host.LibraryGetLibrary(1)
if err != nil {
    // Handle error
}
library := resp.Result

// Get all libraries
resp, err := host.LibraryGetAllLibraries()
for _, lib := range resp.Result {
    // lib is of type host.Library
    fmt.Printf("Library: %s (%d songs)\n", lib.Name, lib.TotalSongs)
}
```

### Artwork

Generate public URLs for Navidrome artwork (albums, artists, tracks, playlists).

**Manifest permission:**

```json
{
  "permissions": {
    "artwork": {
      "reason": "Get artwork URLs for display"
    }
  }
}
```

**Host functions:**

| Function                 | Parameters | Returns     |
|--------------------------|------------|-------------|
| `artwork_getartisturl`   | `id, size` | Artwork URL |
| `artwork_getalbumurl`    | `id, size` | Artwork URL |
| `artwork_gettrackurl`    | `id, size` | Artwork URL |
| `artwork_getplaylisturl` | `id, size` | Artwork URL |

### SubsonicAPI

Call Navidrome's Subsonic API internally (no network round-trip).

**Manifest permission:**

```json
{
  "permissions": {
    "subsonicapi": {
      "reason": "Access library data"
    },
    "users": {
      "reason": "Access user information for SubsonicAPI authorization"
    }
  }
}
```

> **Important:** The `subsonicapi` permission requires the `users` permission. User access is controlled through the plugin's database configuration, not the manifest. Configure which users can use the plugin through the Navidrome UI or API.

**Host function:**

| Function           | Parameters | Returns       |
|--------------------|------------|---------------|
| `subsonicapi_call` | `uri`      | JSON response |

**Usage:**

```go
// The URI must include the 'u' parameter with the username
response, err := SubsonicAPICall("getAlbumList2?type=random&size=10&u=username")
```

### Config

Access plugin configuration values programmatically. Unlike `pdk.GetConfig()` which only retrieves individual values, this service can list all available configuration keys—useful for discovering dynamic configuration (e.g., user-to-token mappings).

> **Note:** This service is always available and does not require a manifest permission.

**Host functions:**

| Function        | Parameters | Returns                     |
|-----------------|------------|-----------------------------|
| `config_get`    | `key`      | `value, exists`             |
| `config_getint` | `key`      | `value, exists`             |
| `config_keys`   | `prefix`   | Array of matching key names |

**Usage (with generated SDK):**

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Get a string configuration value
value, exists := host.ConfigGet("api_key")
if exists {
    // Use the value
}

// Get an integer configuration value
count, exists := host.ConfigGetInt("max_retries")

// List all keys with a prefix (useful for user-specific config)
keys := host.ConfigKeys("user:")
for _, key := range keys {
    // key might be "user:john", "user:jane", etc.
}

// List all configuration keys
allKeys := host.ConfigKeys("")
```

### Users

Access user information for the users that the plugin has been granted access to. This is useful for plugins that need to associate data with specific users or display user information.

**Manifest permission:**

```json
{
  "permissions": {
    "users": {
      "reason": "Display user information in status updates"
    }
  }
}
```

**Important:** Before enabling a plugin that requires the `users` permission, an administrator must configure which users the plugin can access. This can be done in two ways:

1. **Allow all users** – Enable the "Allow all users" toggle in the plugin settings
2. **Select specific users** – Choose individual users from the user list

If neither option is configured, the plugin cannot be enabled.

**Host functions:**

| Function         | Parameters | Returns               |
|------------------|------------|-----------------------|
| `users_getusers` | –          | Array of User objects |

**User object fields:**

| Field      | Type    | Description                    |
|------------|---------|--------------------------------|
| `userName` | string  | The user's unique username     |
| `name`     | string  | The user's display name        |
| `isAdmin`  | boolean | Whether the user is an admin   |

> **Security:** Sensitive fields like passwords, email addresses, and internal IDs are never exposed to plugins.

**Usage (with generated SDK):**

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"

// Get all users the plugin has access to
users, err := host.UsersGetUsers()
if err != nil {
    pdk.Log(pdk.LogError, "Failed to get users: " + err.Error())
    return
}

for _, user := range users {
    pdk.Log(pdk.LogInfo, "User: " + user.UserName + " (" + user.Name + ")")
    if user.IsAdmin {
        pdk.Log(pdk.LogInfo, "  - Administrator")
    }
}
```

**Rust example:**

```rust
use nd_pdk_host::users::get_users;

let users = get_users()?;
for user in users {
    println!("User: {} ({})", user.user_name, user.name);
}
```

**Python example:**

```python
from host.nd_host_users import users_get_users

users = users_get_users()
for user in users:
    print(f"User: {user['userName']} ({user['name']})")
```

---

## Configuration

### Server Configuration

Enable plugins in `navidrome.toml`:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/plugins"   # Default: DataFolder/plugins
AutoReload = true             # Auto-reload on file changes (dev mode)
LogLevel = "debug"            # Plugin-specific log level
CacheSize = "200MB"           # Compilation cache size limit
```

### Plugin Configuration

Plugin configuration is managed through the Navidrome web UI. Navigate to the Plugins page, select a plugin, and edit its configuration as key-value pairs.

Access configuration values in your plugin:

```go
apiKey, ok := pdk.GetConfig("api_key")
if !ok {
    pdk.SetErrorString("api_key configuration is required")
    return 1
}
```

---

## Building Plugins

### Supported Languages

Plugins can be written in any language that Extism supports. Each language has its own PDK (Plugin Development Kit) that provides the APIs for I/O, logging, configuration, and HTTP requests. See the [Extism PDK documentation](https://extism.org/docs/concepts/pdk) for details.

We recommend:

- **Go** – Best experience with [TinyGo](https://tinygo.org/) and the [Go PDK](https://github.com/extism/go-pdk)
- **Rust** – Excellent performance with the [Rust PDK](https://github.com/extism/rust-pdk)
- **Python** – Experimental support via [extism-py](https://github.com/extism/python-pdk)
- **TypeScript** – Experimental support via [extism-js](https://github.com/extism/js-pdk)

### Go with TinyGo (Recommended)

```bash
# Install TinyGo: https://tinygo.org/getting-started/install/

# Build WebAssembly module
tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .

# Package as .ndp
zip -j my-plugin.ndp manifest.json plugin.wasm
```

#### Using Go PDK Packages

Navidrome provides type-safe Go packages for each capability in `plugins/pdk/go/`. Instead of manually exporting functions with `//go:wasmexport`, use the `Register()` pattern:

```go
package main

import (
    "github.com/navidrome/navidrome/plugins/pdk/go/metadata"
)

type myPlugin struct{}

func (p *myPlugin) GetArtistBiography(input metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
    return &metadata.ArtistBiographyResponse{Biography: "Biography text..."}, nil
}

func init() {
    metadata.Register(&myPlugin{})
}

func main() {}
```

Add to your `go.mod`:

```
require github.com/navidrome/navidrome v0.0.0
replace github.com/navidrome/navidrome => ../../..
```

Available capability packages:

| Package     | Import Path                | Description                          |
|-------------|----------------------------|--------------------------------------|
| `metadata`  | `plugins/pdk/go/metadata`  | Artist/album metadata providers      |
| `scrobbler` | `plugins/pdk/go/scrobbler` | Scrobbling services                  |
| `lifecycle` | `plugins/pdk/go/lifecycle` | Plugin initialization                |
| `scheduler` | `plugins/pdk/go/scheduler` | Scheduled task callbacks             |
| `websocket` | `plugins/pdk/go/websocket` | WebSocket event handlers             |
| `host`      | `plugins/pdk/go/host`      | Host service SDK (HTTP, cache, etc.) |

See the example plugins in [examples/](examples/) for complete usage patterns.

### Rust

```bash
# Build WebAssembly module
cargo build --release --target wasm32-wasip1

# Package as .ndp
zip -j my-plugin.ndp manifest.json target/wasm32-wasip1/release/plugin.wasm
```

#### Using Rust PDK

The Rust PDK provides generated type-safe wrappers for both capabilities and host services:

```toml
# Cargo.toml
[dependencies]
nd-pdk = { path = "../../pdk/rust/nd-pdk" }
extism-pdk = "1.2"
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

**Implementing capabilities with traits and macros:**

```rust
use nd_pdk::scrobbler::{Scrobbler, IsAuthorizedRequest, Error};
use nd_pdk::register_scrobbler;

#[derive(Default)]
struct MyPlugin;

impl Scrobbler for MyPlugin {
    fn is_authorized(&self, req: IsAuthorizedRequest) -> Result<bool, Error> {
        Ok(true)
    }
    fn now_playing(&self, req: NowPlayingRequest) -> Result<(), Error> { Ok(()) }
    fn scrobble(&self, req: ScrobbleRequest) -> Result<(), Error> { Ok(()) }
}

register_scrobbler!(MyPlugin);  // Generates all WASM exports
```

**Using host services:**

```rust
use nd_pdk::host::{cache, scheduler, library};

// Cache a value for 1 hour
cache::set_string("my_key", "my_value", 3600)?;

// Schedule a recurring task
scheduler::schedule_recurring("@every 5m", "payload", "task_id")?;

// Access library metadata
let libs = library::get_all_libraries()?;
```

See [pdk/rust/README.md](pdk/rust/README.md) for detailed documentation and examples.

### Python (with extism-py)

```bash
# Build WebAssembly module (requires extism-py installed)
extism-py plugin.wasm -o plugin.wasm *.py

# Package as .ndp
zip -j my-plugin.ndp manifest.json plugin.wasm
```

### Using XTP CLI (Scaffolding)

Bootstrap a new plugin from a schema:

```bash
# Install XTP CLI: https://docs.xtp.dylibso.com/docs/cli

# Create a metadata agent plugin
xtp plugin init \
  --schema-file plugins/capabilities/metadata_agent.yaml \
  --template go \
  --path ./my-agent \
  --name my-agent

# Build and package
cd my-agent && xtp plugin build
zip -j my-agent.ndp manifest.json dist/plugin.wasm
```

See [capabilities/README.md](capabilities/README.md) for available schemas and scaffolding examples.

### Using Host Service SDKs

Generated SDKs for calling host services are in `plugins/pdk/go/`, `plugins/pdk/python/` and `plugins/pdk/rust`.

**For Go plugins:** Import the SDK as a Go module:

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/host"
```

Add to your `go.mod`:

```
require github.com/navidrome/navidrome/plugins/pdk/go v0.0.0
replace github.com/navidrome/navidrome/plugins/pdk/go => ../../pdk/go
```

See [pdk/go/README.md](pdk/go/README.md) for detailed documentation.

**For Python plugins:** Copy functions from `nd_host_*.py` into your `__init__.py` (see comments in those files for extism-py limitations).

**Recommendations:**

- **Go:** Best overall experience with excellent stdlib support and familiar syntax for most developers. Recommended if you're already in the Go ecosystem.
- **Rust:** Best for performance-critical plugins or when leveraging Rust's ecosystem. Produces smallest binaries with excellent type safety.
- **Python:** Best for rapid prototyping or simple plugins. Note that extism-py has limitations compared to compiled languages.

---

## Examples

See [examples/](examples/) for complete working plugins:

| Plugin                                                         | Language | Capabilities  | Host Services                              | Description                    |
|----------------------------------------------------------------|----------|---------------|--------------------------------------------|--------------------------------|
| [minimal](examples/minimal/)                                   | Go       | MetadataAgent | –                                          | Basic structure example        |
| [wikimedia](examples/wikimedia/)                               | Go       | MetadataAgent | HTTP                                       | Wikidata/Wikipedia integration |
| [coverartarchive-py](examples/coverartarchive-py/)             | Python   | MetadataAgent | HTTP                                       | Cover Art Archive              |
| [webhook-rs](examples/webhook-rs/)                             | Rust     | Scrobbler     | HTTP                                       | HTTP webhooks                  |
| [nowplaying-py](examples/nowplaying-py/)                       | Python   | Lifecycle     | Scheduler, SubsonicAPI                     | Periodic now-playing logger    |
| [library-inspector](examples/library-inspector-rs/)               | Rust     | Lifecycle     | Library, Scheduler                         | Periodic library stats logging |
| [crypto-ticker](examples/crypto-ticker/)                       | Go       | Lifecycle     | WebSocket, Scheduler                       | Real-time crypto prices demo   |
| [discord-rich-presence](examples/discord-rich-presence/)       | Go       | Scrobbler     | HTTP, WebSocket, Cache, Scheduler, Artwork | Discord integration            |
| [discord-rich-presence-rs](examples/discord-rich-presence-rs/) | Rust     | Scrobbler     | HTTP, WebSocket, Cache, Scheduler, Artwork | Discord integration (Rust)     |

---


## Security

Plugins run in a secure WebAssembly sandbox provided by [Extism](https://extism.org/) and the [Wazero](https://wazero.io/) runtime:

1. **Host Allowlisting** – Only explicitly allowed hosts are accessible via HTTP/WebSocket
2. **Limited File System** – Plugins can only access library directories when explicitly granted the `library.filesystem` permission, and access is read-only
3. **No Network Listeners** – Plugins cannot bind ports
4. **Config Isolation** – Plugins only receive their own config section
5. **Memory Limits** – Controlled by the WebAssembly runtime
6. **User-Scoped Authorization** – Plugins with `subsonicapi` or `scrobbler` capabilities can only access/receive events for users assigned to them through Navidrome's configuration. The `users` permission is required for these features.
7. **Users Permission** – Plugins requesting user access must be explicitly configured with allowed users; sensitive data (passwords, emails) is never exposed


---

## Runtime Management

### Auto-Reload

With `AutoReload = true`, Navidrome watches the plugins folder and automatically detects when `.ndp` files are added, modified, or removed. When a plugin file changes, the plugin is disabled and its metadata is re-read from the archive.

If the `AutoReload` setting is disabled, Navidrome needs to be restarted to pick up plugin changes.

### Enabling/Disabling Plugins

Plugins can be enabled/disabled via the Navidrome UI. The plugin state is persisted in the database.

### Important Notes

- **In-flight requests** – When reloading, existing requests complete before the new version takes over
- **Config changes** – Changes to the plugin configuration in the UI are applied immediately
- **Cache persistence** – The in-memory cache is cleared when a plugin is unloaded
