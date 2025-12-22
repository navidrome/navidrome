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
