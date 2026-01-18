# Discord Rich Presence Plugin

This example plugin integrates Navidrome with Discord Rich Presence. It shows how a plugin can keep a real-time connection to an external service while remaining completely stateless. This plugin is based on the [Navicord](https://github.com/logixism/navicord) project, which provides similar functionality.

**⚠️ WARNING: This plugin is for demonstration purposes only. It relies on the user's Discord token being stored in the Navidrome configuration file, which is not secure and may be against Discord's terms of service. Use it at your own risk.**

## Overview

The plugin exposes three capabilities:

- **Scrobbler** – receives `NowPlaying` notifications from Navidrome
- **WebSocketCallback** – handles Discord gateway messages
- **SchedulerCallback** – used to clear presence and send periodic heartbeats

It relies on several host services declared in the manifest:

- `http` – queries Discord API endpoints
- `websocket` – maintains gateway connections
- `scheduler` – schedules heartbeats and presence cleanup
- `cache` – stores sequence numbers for heartbeats
- `artwork` – resolves track artwork URLs

## Architecture

The plugin registers capabilities using the PDK Register pattern:

```go
import (
    "github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
    "github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
    "github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)

type discordPlugin struct{}

func init() {
    scrobbler.Register(&discordPlugin{})
    scheduler.Register(&discordPlugin{})
    websocket.Register(&discordPlugin{})
}
```

The PDK generates the appropriate export wrappers automatically.

When `NowPlaying` is invoked the plugin:

1. Loads `clientid` and user tokens from the configuration (because plugins are stateless).
2. Connects to Discord using `WebSocketService` if no connection exists.
3. Sends the activity payload with track details and artwork.
4. Schedules a one-time callback to clear the presence after the track finishes.

Heartbeat messages are sent by a recurring scheduler job. Sequence numbers received from Discord are stored in `CacheService` to remain available across plugin instances.

The scheduler callback uses the `payload` field to route to the appropriate handler:
- `"heartbeat"` – sends a heartbeat to Discord (recurring)
- `"clear-activity"` – clears the presence and disconnects (one-time)

## Stateless Operation

Navidrome plugins are completely stateless – each method call instantiates a new plugin instance and discards it afterwards.

To work within this model the plugin stores no in-memory state. Connections are keyed by username inside the host services and any transient data (like Discord sequence numbers) is kept in the cache. Configuration is reloaded on every method call.

## Configuration

Configure in the Navidrome UI (Settings → Plugins → discord-rich-presence):

| Key           | Description                                | Example                        |
|---------------|-------------------------------------------|--------------------------------|
| `clientid`    | Your Discord application ID               | `123456789012345678`           |
| `user.<name>` | Discord token for the specified user      | `user.alice` = `token123`      |

Each user is configured as a separate key with the `user.` prefix.

## Building

From the `plugins/examples/` directory:

```sh
make discord-rich-presence.ndp
```

Or manually:

```sh
cd discord-rich-presence
tinygo build -target wasip1 -buildmode=c-shared -o plugin.wasm .
zip -j discord-rich-presence.ndp manifest.json plugin.wasm
```

## Installation

Place the resulting `discord-rich-presence.ndp` in your Navidrome plugins folder and enable plugins in your configuration:

```toml
[Plugins]
Enabled = true
Folder = "/path/to/plugins"
```

## Files

| File      | Description                                                      |
|-----------|------------------------------------------------------------------|
| `main.go` | Plugin entry point, capability registration, and implementations |
| `rpc.go`  | Discord gateway communication and RPC logic                      |
| `go.mod`  | Go module file                                                   |

## PDK

This plugin imports the Navidrome PDK subpackages directly:

```go
import (
    "github.com/navidrome/navidrome/plugins/pdk/go/host"
    "github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
    "github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
    "github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)
```

The `go.mod` file uses `replace` directives to point to the local packages for development.

## Host Services Used

| Service   | Purpose                                                          |
|-----------|------------------------------------------------------------------|
| Cache     | Store Discord sequence numbers and processed image URLs          |
| Scheduler | Schedule heartbeats (recurring) and activity clearing (one-time) |
| WebSocket | Maintain persistent connection to Discord gateway                |
| Artwork   | Get track artwork URLs for rich presence display                 |

## Implementation Details

See `main.go` and `rpc.go` for the complete implementation.
