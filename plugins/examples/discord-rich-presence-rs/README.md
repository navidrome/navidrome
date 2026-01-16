# Discord Rich Presence Plugin (Rust)

A Navidrome plugin that displays your currently playing track on Discord using Rich Presence. This is the Rust implementation demonstrating how to use the `nd-pdk` library.

## ⚠️ Warning

This plugin is for **demonstration purposes only**. It requires storing your Discord token in the Navidrome configuration file, which:

1. Is not secure (tokens should never be stored in plain text)
2. May violate Discord's Terms of Service

**Use at your own risk.**

## Features

- Shows currently playing track on Discord Rich Presence
- Displays album artwork
- Shows track progress with start/end timestamps
- Automatically clears presence when track finishes
- Supports multiple users

## Capabilities

This plugin implements multiple capabilities to demonstrate the nd-pdk library:

- **Scrobbler**: Receives now-playing events from Navidrome
- **SchedulerCallback**: Handles heartbeat and activity clearing timers
- **WebSocketCallback**: Communicates with Discord gateway (text, binary, error, and close handlers)

## Configuration

Configure in the Navidrome UI (Settings → Plugins → discord-rich-presence):

| Key           | Description                          | Example                   |
|---------------|--------------------------------------|---------------------------|
| `clientid`    | Your Discord application ID          | `123456789012345678`      |
| `user.<name>` | Discord token for the specified user | `user.alice` = `token123` |

Each user is configured as a separate key with the `user.` prefix.


### Getting Configuration Values

1. **Client ID**: Create a Discord Application at https://discord.com/developers/applications and copy the Application ID

2. **Discord Token**: This requires extracting your user token from Discord (not recommended for security reasons)

3. **Multiple Users**: Add multiple user keys:
   ```properties
   user.user1 = "token1"
   user.user2 = "token2"
   ```

## Building

```bash
# From the plugins/examples directory
make discord-rich-presence-rs.ndp

# This creates discord-rich-presence-rs.ndp containing:
# - manifest.json
# - plugin.wasm
```

## Installation

1. Build the plugin using the command above
2. Copy the `.ndp` file to your Navidrome plugins directory
3. Enable and configure the plugin in the Navidrome UI (Settings → Plugins)
4. Restart Navidrome if needed

## Using nd-pdk Library

This plugin demonstrates how to use the Rust plugin development kit:

```rust
use nd_pdk::host::{artwork, cache, scheduler, websocket};
use std::collections::HashMap;

// Get artwork URL
let url = artwork::get_track_url(track_id, 300)?;

// Cache operations
cache::set_string("key", "value", 3600)?;
if let Some(value) = cache::get_string("key")? {
    // Use the cached value
}

// Schedule tasks
scheduler::schedule_one_time(60, "payload", "task-id")?;
scheduler::schedule_recurring("@every 30s", "heartbeat", "heartbeat-task")?;

// WebSocket operations
let conn_id = websocket::connect("wss://example.com/socket", HashMap::new(), "my-conn")?;
websocket::send_text(&conn_id, "Hello")?;
```

## License

GPL-3.0
