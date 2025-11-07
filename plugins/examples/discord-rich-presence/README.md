# Discord Rich Presence Plugin

This example plugin integrates Navidrome with Discord Rich Presence. It shows how a plugin can keep a real-time
connection to an external service while remaining completely stateless. This plugin is based on the 
[Navicord](https://github.com/logixism/navicord) project, which provides a similar functionality.

**NOTE: This plugin is for demonstration purposes only. It relies on the user's Discord token being stored in the 
Navidrome configuration file, which is not secure, and may be against Discord's terms of service. 
Use it at your own risk.**

## Overview

The plugin exposes three capabilities:

- **Scrobbler** – receives `NowPlaying` notifications from Navidrome
- **WebSocketCallback** – handles Discord gateway messages
- **SchedulerCallback** – used to clear presence and send periodic heartbeats

It relies on several host services declared in `manifest.json`:

- `http` – queries Discord API endpoints
- `websocket` – maintains gateway connections
- `scheduler` – schedules heartbeats and presence cleanup
- `cache` – stores sequence numbers for heartbeats
- `config` – retrieves the plugin configuration on each call
- `artwork` – resolves track artwork URLs

## Architecture

Each call from Navidrome creates a new plugin instance. The `init` function registers the capabilities and obtains the
scheduler service:

```go
api.RegisterScrobbler(plugin)
api.RegisterWebSocketCallback(plugin.rpc)
plugin.sched = api.RegisterNamedSchedulerCallback("close-activity", plugin)
plugin.rpc.sched = api.RegisterNamedSchedulerCallback("heartbeat", plugin.rpc)
```

When `NowPlaying` is invoked the plugin:

1. Loads `clientid` and user tokens from the configuration (because plugins are stateless).
2. Connects to Discord using `WebSocketService` if no connection exists.
3. Sends the activity payload with track details and artwork.
4. Schedules a one‑time callback to clear the presence after the track finishes.

Heartbeat messages are sent by a recurring scheduler job. Sequence numbers received from Discord are stored in
`CacheService` to remain available across plugin instances.

The `OnSchedulerCallback` method clears the presence and closes the connection when the scheduled time is reached.

```go
// The plugin is stateless, we need to load the configuration every time
clientID, users, err := d.getConfig(ctx)
```

## Configuration

Add the following to `navidrome.toml` and adjust for your tokens:

```toml
[PluginConfig.discord-rich-presence]
ClientID = "123456789012345678"
Users = "alice:token123,bob:token456"
```

- `clientid` is your Discord application ID
- `users` is a comma‑separated list of `username:token` pairs used for authorization

## Building

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm ./discord-rich-presence/...
```

Place the resulting `plugin.wasm` and `manifest.json` in a `discord-rich-presence` folder under your Navidrome plugins
directory.

## Stateless Operation

Navidrome plugins are completely stateless – each method call instantiates a new plugin instance and discards it
afterwards.

To work within this model the plugin stores no in-memory state. Connections are keyed by user name inside the host
services and any transient data (like Discord sequence numbers) is kept in the cache. Configuration is reloaded on every
method call.

For more implementation details see `plugin.go` and `rpc.go`.
