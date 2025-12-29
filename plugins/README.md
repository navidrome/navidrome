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
      "allowedHosts": ["api.example.com", "*.musicbrainz.org"]
    }
  }
}
```

**Required fields:** `name`, `author`, `version`

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

| Function                     | Input                 | Output                  | Description                 |
|------------------------------|-----------------------|-------------------------|-----------------------------|
| `nd_scrobbler_is_authorized` | `{user_id, username}` | `{authorized}`          | Check if user is authorized |
| `nd_scrobbler_now_playing`   | See below             | `{error?, error_type?}` | Send now playing            |
| `nd_scrobbler_scrobble`      | See below             | `{error?, error_type?}` | Submit a scrobble           |

**NowPlaying/Scrobble Input:**

```json
{
  "user_id": "abc123",
  "username": "john",
  "track": {
    "id": "track-id",
    "title": "Song Title",
    "album": "Album Name",
    "artist": "Artist Name",
    "album_artist": "Album Artist",
    "duration": 180.5,
    "track_number": 1,
    "disc_number": 1,
    "mbz_recording_id": "...",
    "mbz_album_id": "...",
    "mbz_artist_id": "..."
  },
  "timestamp": 1703270400
}
```

**Error Output (on failure):**

```json
{
  "error": "error message",
  "error_type": "not_authorized|retry_later|unrecoverable"
}
```

- `not_authorized` – User needs to re-authorize
- `retry_later` – Temporary failure, Navidrome will retry
- `unrecoverable` – Permanent failure, scrobble discarded

On success, return empty JSON `{}` or omit output entirely.

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
      "allowedHosts": ["api.example.com", "*.musicbrainz.org"]
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
| `scheduler_scheduleonetime`   | `delay_seconds, payload, schedule_id?`   | Schedule one-time callback  |
| `scheduler_schedulerecurring` | `cron_expression, payload, schedule_id?` | Schedule recurring callback |
| `scheduler_cancelschedule`    | `schedule_id`                            | Cancel a scheduled task     |

**Callback function:**

```go
type SchedulerCallbackInput struct {
    ScheduleID  string `json:"schedule_id"`
    Payload     string `json:"payload"`
    IsRecurring bool   `json:"is_recurring"`
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

Copy `plugins/host/go/nd_host_scheduler.go` to your plugin and use:

```go
// Schedule one-time task in 60 seconds
scheduleID, err := SchedulerScheduleOneTime(60, "my-payload", "")

// Schedule recurring task with cron expression (every hour)
scheduleID, err := SchedulerScheduleRecurring("0 * * * *", "hourly-task", "")

// Cancel a task
err := SchedulerCancelSchedule(scheduleID)
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

Copy `plugins/host/go/nd_host_cache.go` to your plugin:

```go
// Cache a value for 1 hour
CacheSetString("api-response", responseData, 3600)

// Retrieve (check Exists before using Value)
result, err := CacheGetString("api-response")
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

Copy `plugins/host/go/nd_host_kvstore.go` to your plugin:

```go
// Store a value (as raw bytes)
token := []byte(`{"access_token": "xyz", "refresh_token": "abc"}`)
_, err := KVStoreSet("oauth:spotify", token)

// Retrieve a value
result, err := KVStoreGet("oauth:spotify")
if result.Exists {
    var tokenData map[string]string
    json.Unmarshal(result.Value, &tokenData)
}

// List all keys with prefix
keysResult, err := KVStoreList("user:")
for _, key := range keysResult.Keys {
    // Process each key
}

// Check storage usage
usageResult, err := KVStoreGetStorageUsed()
fmt.Printf("Using %d bytes\n", usageResult.Bytes)

// Delete a value
KVStoreDelete("oauth:spotify")
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
      "allowedHosts": ["gateway.example.com", "*.discord.gg"]
    }
  }
}
```

**Host functions:**

| Function               | Parameters                      | Description       |
|------------------------|---------------------------------|-------------------|
| `websocket_connect`    | `url, headers?, connection_id?` | Open a connection |
| `websocket_sendtext`   | `connection_id, message`        | Send text message |
| `websocket_sendbinary` | `connection_id, data`           | Send binary data  |
| `websocket_close`      | `connection_id, code?, reason?` | Close connection  |

**Callback functions (export these to receive events):**

| Function                         | Input                           | Description                      |
|----------------------------------|---------------------------------|----------------------------------|
| `nd_websocket_on_text_message`   | `{connection_id, message}`      | Text message received            |
| `nd_websocket_on_binary_message` | `{connection_id, data}`         | Binary message received (base64) |
| `nd_websocket_on_error`          | `{connection_id, error}`        | Connection error                 |
| `nd_websocket_on_close`          | `{connection_id, code, reason}` | Connection closed                |

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

Copy `plugins/host/go/nd_host_library.go` to your plugin. You'll also need to add the `Library` struct definition:

```go
// Library represents a music library with metadata.
type Library struct {
    ID            int32   `json:"id"`
    Name          string  `json:"name"`
    Path          string  `json:"path,omitempty"`
    MountPoint    string  `json:"mountPoint,omitempty"`
    LastScanAt    int64   `json:"lastScanAt"`
    TotalSongs    int32   `json:"totalSongs"`
    TotalAlbums   int32   `json:"totalAlbums"`
    TotalArtists  int32   `json:"totalArtists"`
    TotalSize     int64   `json:"totalSize"`
    TotalDuration float64 `json:"totalDuration"`
}

// Get a specific library
resp, err := LibraryGetLibrary(1)
if err != nil {
    // Handle error
}
library := resp.Result

// Get all libraries
resp, err := LibraryGetAllLibraries()
for _, lib := range resp.Result {
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
      "reason": "Access library data",
      "allowedUsernames": ["user1", "user2"],
      "allowAdmins": false
    }
  }
}
```

- `allowedUsernames` – Restrict which users the plugin can act as (empty = any user)
- `allowAdmins` – Whether plugin can call API as admin users (default: false)

**Host function:**

| Function           | Parameters | Returns       |
|--------------------|------------|---------------|
| `subsonicapi_call` | `uri`      | JSON response |

**Usage:**

```go
// The URI must include the 'u' parameter with the username
response, err := SubsonicAPICall("getAlbumList2?type=random&size=10&u=username")
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

### Rust

```bash
# Build WebAssembly module
cargo build --release --target wasm32-unknown-unknown

# Package as .ndp
zip -j my-plugin.ndp manifest.json target/wasm32-unknown-unknown/release/plugin.wasm
```

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
  --schema-file plugins/schemas/metadata_agent.yaml \
  --template go \
  --path ./my-agent \
  --name my-agent

# Build and package
cd my-agent && xtp plugin build
zip -j my-agent.ndp manifest.json dist/plugin.wasm
```

See [schemas/README.md](schemas/README.md) for available schemas.

### Using Host Service SDKs

Generated SDKs for calling host services are in `plugins/host/go/` and `plugins/host/python/`.

**For Go plugins:** Copy the needed `nd_host_*.go` file to your plugin directory.

**For Python plugins:** Copy functions from `nd_host_*.py` into your `__init__.py` (see comments in those files for extism-py limitations).

---

## Examples

See [examples/](examples/) for complete working plugins:

| Plugin                                                   | Language | Capabilities            | Host Services                               | Description                    |
|----------------------------------------------------------|----------|-------------------------|---------------------------------------------|--------------------------------|
| [minimal](examples/minimal/)                             | Go       | MetadataAgent           | –                                           | Basic structure example        |
| [wikimedia](examples/wikimedia/)                         | Go       | MetadataAgent           | HTTP                                        | Wikidata/Wikipedia integration |
| [coverartarchive-py](examples/coverartarchive-py/)       | Python   | MetadataAgent           | HTTP                                        | Cover Art Archive              |
| [webhook-rs](examples/webhook-rs/)                       | Rust     | Scrobbler               | HTTP                                        | HTTP webhooks                  |
| [nowplaying-py](examples/nowplaying-py/)                 | Python   | Lifecycle               | Scheduler, SubsonicAPI                      | Periodic now-playing logger    |
| [library-inspector](examples/library-inspector/)         | Rust     | Lifecycle               | Library, Scheduler                          | Periodic library stats logging |
| [crypto-ticker](examples/crypto-ticker/)                 | Go       | Lifecycle               | WebSocket, Scheduler                        | Real-time crypto prices demo   |
| [discord-rich-presence](examples/discord-rich-presence/) | Go       | Scrobbler               | HTTP, WebSocket, Cache, Scheduler, Artwork  | Discord integration            |

---

## Security

Plugins run in a secure WebAssembly sandbox provided by [Extism](https://extism.org/) and the [Wazero](https://wazero.io/) runtime:

1. **Host Allowlisting** – Only explicitly allowed hosts are accessible via HTTP/WebSocket
2. **Limited File System** – Plugins can only access library directories when explicitly granted the `library.filesystem` permission, and access is read-only
3. **No Network Listeners** – Plugins cannot bind ports
4. **Config Isolation** – Plugins only receive their own config section
5. **Memory Limits** – Controlled by the WebAssembly runtime
6. **SubsonicAPI Restrictions** – Configurable user/admin access controls


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
