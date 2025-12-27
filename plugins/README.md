# Navidrome Plugin System

Navidrome supports WebAssembly (Wasm) plugins for extending functionality. Plugins are loaded from the configured plugins folder and can provide additional metadata agents for fetching artist/album information.

## Configuration

Enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/plugins"   # Default: DataFolder/plugins

# Plugin-specific configuration (passed to plugins via Extism Config)
[PluginConfig.my-plugin]
api_key = "your-api-key"
custom_option = "value"
```

## Plugin Structure

A Navidrome plugin is a WebAssembly (`.wasm`) file that:

1. **Exports `nd_manifest`**: Returns a JSON manifest describing the plugin
2. **Exports capability functions**: Implements the functions for its declared capabilities

### Plugin Naming

Plugins are identified by their **filename** (without `.wasm` extension), not the manifest `name` field. This allows:
- Users to resolve name conflicts by renaming files
- Multiple instances of the same plugin with different names/configs
- Simple, predictable naming

Example: `my-musicbrainz.wasm` → plugin name is `my-musicbrainz`

### Plugin Manifest

Plugins must export an `nd_manifest` function that returns JSON:

```json
{
  "name": "My Plugin",
  "author": "Author Name",
  "version": "1.0.0",
  "description": "Plugin description",
  "website": "https://example.com",
  "permissions": {
    "http": {
      "reason": "Fetch metadata from external API",
      "allowedHosts": ["api.example.com", "*.musicbrainz.org"]
    }
  }
}
```

**Note**: Capabilities are auto-detected based on which functions the plugin exports. You don't need to declare them in the manifest.

## Capabilities

Capabilities are automatically detected by examining which functions a plugin exports. There's no need to declare capabilities in the manifest.

### MetadataAgent

Provides artist and album metadata. A plugin has this capability if it exports one or more of these functions:

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

### Scrobbler

Provides scrobbling (listening history) integration with external services. A plugin has this capability if it exports one or more of these functions:

| Function                     | Input                 | Output                  | Description                   |
|------------------------------|-----------------------|-------------------------|-------------------------------|
| `nd_scrobbler_is_authorized` | `{user_id, username}` | `{authorized}`          | Check if user is authorized   |
| `nd_scrobbler_now_playing`   | See NowPlaying Input  | `{error?, error_type?}` | Send now playing notification |
| `nd_scrobbler_scrobble`      | See Scrobble Input    | `{error?, error_type?}` | Submit a scrobble             |

#### NowPlaying Input

```json
{
  "user_id": "string",
  "username": "string",
  "track": {
    "id": "string",
    "title": "string",
    "album": "string",
    "artist": "string",
    "album_artist": "string",
    "duration": 180.5,
    "track_number": 1,
    "disc_number": 1,
    "mbz_recording_id": "string",
    "mbz_album_id": "string",
    "mbz_artist_id": "string",
    "mbz_release_group_id": "string",
    "mbz_album_artist_id": "string",
    "mbz_release_track_id": "string"
  },
  "position": 30
}
```

#### Scrobble Input

```json
{
  "user_id": "string",
  "username": "string",
  "track": { /* same as NowPlaying */ },
  "timestamp": 1703270400
}
```

#### Scrobbler Output

The output for `nd_scrobbler_now_playing` and `nd_scrobbler_scrobble` is **optional on success**. If there is no error, the plugin can return nothing (empty output).

On error, return:

```json
{
  "error": "error message",
  "error_type": "not_authorized|retry_later|unrecoverable"
}
```

**Error types:**
- `not_authorized`: User needs to re-authorize with the scrobbling service
- `retry_later`: Temporary failure, Navidrome will retry the scrobble later
- `unrecoverable`: Permanent failure, scrobble will be discarded

#### Example Scrobbler Plugin

```go
package main

import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

type AuthInput struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
}

type AuthOutput struct {
    Authorized bool `json:"authorized"`
}

type ScrobblerOutput struct {
    Error     string `json:"error,omitempty"`
    ErrorType string `json:"error_type,omitempty"`
}

//go:wasmexport nd_scrobbler_is_authorized
func ndScrobblerIsAuthorized() int32 {
    var input AuthInput
    if err := pdk.InputJSON(&input); err != nil {
        pdk.SetError(err)
        return 1
    }
    
    // Check if user is authorized with your scrobbling service
    // This could check a session key stored in plugin config
    sessionKey, hasKey := pdk.GetConfig("session_key_" + input.UserID)
    
    output := AuthOutput{Authorized: hasKey && sessionKey != ""}
    if err := pdk.OutputJSON(output); err != nil {
        pdk.SetError(err)
        return 1
    }
    return 0
}

//go:wasmexport nd_scrobbler_scrobble
func ndScrobblerScrobble() int32 {
    // Read input, send to external service...
    
    output := ScrobblerOutput{ErrorType: "none"}
    if err := pdk.OutputJSON(output); err != nil {
        pdk.SetError(err)
        return 1
    }
    return 0
}

func main() {}
```

Scrobbler plugins are automatically discovered and used by Navidrome's PlayTracker alongside built-in scrobblers (Last.fm, ListenBrainz).

### Scheduler

Allows plugins to schedule one-time or recurring tasks. Plugins that use the scheduler host service must export a callback function to receive scheduled events.

| Function                | Input                                        | Output            | Description                        |
|-------------------------|----------------------------------------------|-------------------|------------------------------------||
| `nd_scheduler_callback` | `{schedule_id, payload, is_recurring}`       | `{error?}`        | Called when a scheduled task fires |

#### Scheduler Callback Input

```json
{
  "schedule_id": "string",
  "payload": "string",
  "is_recurring": true
}
```

- `schedule_id`: The unique identifier for the scheduled task
- `payload`: Data passed when the task was scheduled
- `is_recurring`: `true` for recurring schedules, `false` for one-time

#### Scheduler Callback Output

The output is optional on success. On error, return:

```json
{
  "error": "error message"
}
```

#### Using the Scheduler Host Service

To schedule tasks, plugins call these host functions (provided by Navidrome):

| Host Function                 | Parameters                              | Description                   |
|-------------------------------|-----------------------------------------|-------------------------------|
| `scheduler_scheduleonetime`   | `delay_seconds, payload, schedule_id`   | Schedule a one-time callback  |
| `scheduler_schedulerecurring` | `cron_expression, payload, schedule_id` | Schedule a recurring callback |
| `scheduler_cancelschedule`    | `schedule_id`                           | Cancel a scheduled task       |

#### Manifest Permissions

Plugins using the scheduler must declare the permission in their manifest:

```json
{
  "permissions": {
    "scheduler": {
      "reason": "Schedule periodic metadata refresh"
    }
  }
}
```

#### Example Scheduler Plugin

```go
package main

import (
    "github.com/extism/go-pdk"
)

type SchedulerCallbackInput struct {
    ScheduleId  string `json:"schedule_id"`
    Payload     string `json:"payload"`
    IsRecurring bool   `json:"is_recurring"`
}

type SchedulerCallbackOutput struct {
    Error *string `json:"error,omitempty"`
}

//go:wasmexport nd_scheduler_callback
func ndSchedulerCallback() int32 {
    var input SchedulerCallbackInput
    if err := pdk.InputJSON(&input); err != nil {
        pdk.SetError(err)
        return 1
    }
    
    // Handle the scheduled task based on payload
    pdk.Log(pdk.LogInfo, "Task fired: " + input.ScheduleId)
    
    // Return success (empty output)
    output := SchedulerCallbackOutput{}
    if err := pdk.OutputJSON(output); err != nil {
        pdk.SetError(err)
        return 1
    }
    return 0
}

func main() {}
```

To schedule a task from your plugin, use the generated SDK functions (see `plugins/host/go/nd_host_scheduler.go`).

### Cache

Allows plugins to store and retrieve data in an in-memory TTL-based cache. This is useful for caching API responses, storing session tokens, or persisting state across plugin invocations.

**Important:** The cache is in-memory only and will be lost on server restart. Plugins should handle cache misses gracefully.

#### Using the Cache Host Service

To use the cache, plugins call these host functions (provided by Navidrome):

| Host Function        | Parameters                     | Description                                    |
|----------------------|--------------------------------|------------------------------------------------|
| `cache_setstring`    | `key, value, ttl_seconds`      | Store a string value                           |
| `cache_getstring`    | `key`                          | Retrieve a string value                        |
| `cache_setint`       | `key, value, ttl_seconds`      | Store an integer value                         |
| `cache_getint`       | `key`                          | Retrieve an integer value                      |
| `cache_setfloat`     | `key, value, ttl_seconds`      | Store a float value                            |
| `cache_getfloat`     | `key`                          | Retrieve a float value                         |
| `cache_setbytes`     | `key, value, ttl_seconds`      | Store a byte slice                             |
| `cache_getbytes`     | `key`                          | Retrieve a byte slice                          |
| `cache_has`          | `key`                          | Check if a key exists                          |
| `cache_remove`       | `key`                          | Delete a cached value                          |

**TTL (Time-to-Live):** Pass `0` to use the default TTL of 24 hours, or specify seconds.

**Key Isolation:** Each plugin's cache keys are automatically namespaced, so different plugins can use the same key names without conflicts.

#### Get Response Format

Get operations return a JSON response:

```json
{
  "value": "...",
  "exists": true,
  "error": ""
}
```

- `value`: The cached value (type matches the operation: string, int64, float64, or base64-encoded bytes)
- `exists`: `true` if the key was found and the type matched, `false` otherwise
- `error`: Error message if something went wrong

#### Manifest Permissions

Plugins using the cache must declare the permission in their manifest:

```json
{
  "permissions": {
    "cache": {
      "reason": "Cache API responses to reduce external requests"
    }
  }
}
```

#### Example Cache Usage

```go
package main

import (
    "github.com/extism/go-pdk"
)

// Import the generated cache SDK (from plugins/host/go/nd_host_cache.go)

func fetchWithCache(key string) (string, error) {
    // Try to get from cache first
    resp, err := CacheGetString(key)
    if err != nil {
        return "", err
    }
    if resp.Exists {
        return resp.Value, nil
    }
    
    // Cache miss - fetch from external API
    value := fetchFromAPI()
    
    // Cache for 1 hour (3600 seconds)
    CacheSetString(key, value, 3600)
    
    return value, nil
}
```

To use the cache from your plugin, copy the generated SDK file `plugins/host/go/nd_host_cache.go` to your plugin directory.

## Developing Plugins

Plugins can be written in any language that compiles to WebAssembly. We recommend using the [Extism PDK](https://extism.org/docs/category/write-a-plug-in) for your language.

### Go Example

```go
package main

import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

type Manifest struct {
    Name    string `json:"name"`
    Author  string `json:"author"`
    Version string `json:"version"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
    manifest := Manifest{
        Name:    "My Plugin",
        Author:  "Me",
        Version: "1.0.0",
    }
    out, _ := json.Marshal(manifest)
    pdk.Output(out)
    return 0
}

type ArtistInput struct {
    ID   string `json:"id"`
    Name string `json:"name"`
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
    if err := pdk.OutputJSON(output); err != nil {
        pdk.SetError(err)
        return 1
    }
    return 0
}

func main() {}
```

Build with TinyGo:
```bash
tinygo build -o my-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
```

### Using HTTP

Plugins can make HTTP requests using the Extism PDK. The host controls which hosts are allowed via the `permissions.http.allowedHosts` manifest field.

```go
//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
    var input ArtistInput
    pdk.InputJSON(&input)
    
    req := pdk.NewHTTPRequest(pdk.MethodGet, 
        "https://api.example.com/artist/" + input.Name)
    resp := req.Send()
    
    // Process response...
    pdk.Output(resp.Body())
    return 0
}
```

### Using Configuration

Plugins can read configuration values passed from `navidrome.toml`:

```go
apiKey, ok := pdk.GetConfig("api_key")
if !ok {
    pdk.SetErrorString("api_key configuration is required")
    return 1
}
```

## Runtime Loading

Navidrome supports loading, unloading, and reloading plugins at runtime without restarting the server.

### Auto-Reload (File Watcher)

Enable automatic plugin reloading when files change:

```toml
[Plugins]
Enabled = true
AutoReload = true   # Default: false
```

When enabled, Navidrome watches the plugins folder and automatically:
- **Loads** new `.wasm` files when they are created
- **Reloads** plugins when their `.wasm` file is modified  
- **Unloads** plugins when their `.wasm` file is removed

This is especially useful during plugin development - just rebuild your plugin and it will be automatically reloaded.

### Programmatic API

The plugin Manager exposes methods for runtime plugin management:

```go
manager := plugins.GetManager()

// Load a new plugin (file must exist at <plugins_folder>/<name>.wasm)
err := manager.LoadPlugin("my-plugin")

// Unload a running plugin
err := manager.UnloadPlugin("my-plugin")

// Reload a plugin (unload + load)
err := manager.ReloadPlugin("my-plugin")
```

### Notes on Runtime Loading

- **In-flight requests**: When a plugin is unloaded, existing plugin instances continue working until their request completes. New requests use the reloaded version.
- **Config changes**: Plugin configuration (`PluginConfig.<name>`) is read at load time. Changes require a reload.
- **Failed reloads**: If loading fails after unloading, the plugin remains unloaded. Check logs for errors.

## Host Services (Internal Development)

This section is for Navidrome developers who want to add new host services that plugins can call.

### Overview

Host services allow plugins to call back into Navidrome for functionality like Subsonic API access, scheduling, and other internal services. The `hostgen` tool generates Extism host function wrappers from annotated Go interfaces, automating the boilerplate of memory management, JSON marshalling, and error handling.

### Adding a New Host Service

1. **Create an annotated interface** in `plugins/host/`:

```go
// MyService provides some functionality to plugins.
//nd:hostservice name=MyService permission=myservice
type MyService interface {
    // DoSomething performs an action.
    //nd:hostfunc
    DoSomething(ctx context.Context, input string) (output string, err error)
}
```

2. **Run the generator**:

```bash
make gen
# Or directly:
go run ./plugins/cmd/hostgen -input=./plugins/host -output=./plugins/host
```

3. **Implement the interface** and wire it up in `plugins/manager.go`.

### Annotation Format

#### Service-level (`//nd:hostservice`)

Marks an interface as a host service:
- `name=<ServiceName>` - Service identifier used in generated code
- `permission=<key>` - Manifest permission key (e.g., "subsonicapi", "scheduler")

#### Method-level (`//nd:hostfunc`)

Marks a method for host function wrapper generation:
- `name=<CustomName>` - (Optional) Override the export name

### Method Signature Requirements

- First parameter must be `context.Context`
- Last return value must be `error`
- All parameter types must be JSON-serializable
- Supported types: primitives, structs, slices, maps

### Generated Code

The generator creates `<servicename>_gen.go` with:
- Request/response structs for each method
- `Register<Service>HostFunctions()` - Returns Extism host functions to register
- Helper functions for memory operations and error handling

Example generated function name: `subsonicapi_call` for `SubsonicAPIService.Call`

### Important: Annotation Placement

**The annotation line must immediately precede the type/method declaration without an empty comment line between them.**

✅ **Correct** (annotation directly before type):
```go
// MyService provides functionality.
// More documentation here.
//nd:hostservice name=MyService permission=myservice
type MyService interface { ... }
```

❌ **Incorrect** (empty comment line separates annotation):
```go
// MyService provides functionality.
//
//nd:hostservice name=MyService permission=myservice
type MyService interface { ... }
```

This is due to how Go's AST parser groups comments. An empty `//` line creates a new comment group, causing the annotation to be separated from the type's doc comment.

### Troubleshooting

#### "No host services found" when running generator

1. **Check annotation placement**: Ensure `//nd:hostservice` is on the line immediately before the `type` declaration (no blank `//` line between doc text and annotation).

2. **Check file naming**: The generator skips files ending in `_gen.go` or `_test.go`.

3. **Check interface syntax**: The type must be an interface, not a struct.

4. **Run with verbose flag**: Use `-v` to see what the generator is finding:
   ```bash
   go run ./plugins/cmd/hostgen -input=./plugins/host -output=./plugins/host -v
   ```

#### Generated code doesn't compile

1. **Check method signatures**: First parameter must be `context.Context`, last return must be `error`.

2. **Check parameter types**: All types must be JSON-serializable. Avoid channels, functions, and unexported types.

3. **Review raw output**: Use `-dry-run` to see the generated code without writing files.

#### Methods not being generated

1. **Check `//nd:hostfunc` annotation**: It must be in the method's doc comment, immediately before the method signature.

2. **Check method visibility**: Only methods with names (not embedded interfaces) are processed.

## Security

Plugins run in a secure WebAssembly sandbox with these restrictions:

1. **Host Allowlisting**: Only hosts listed in `permissions.http.allowedHosts` are accessible
2. **No File System Access**: Plugins cannot access the file system
3. **No Network Listeners**: Plugins cannot bind ports or create servers
4. **Config Isolation**: Plugins receive only their own config section
5. **Memory Limits**: Configurable via Extism

## Using Plugins with Agents

To use a plugin as a metadata agent, add it to the `Agents` configuration:

```toml
Agents = "lastfm,spotify,my-plugin"  # my-plugin.wasm must be in the plugins folder
```

Plugins are tried in the order specified, just like built-in agents.
