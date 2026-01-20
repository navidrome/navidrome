# Webhook Scrobbler Plugin (Rust)

A Navidrome plugin written in Rust that sends HTTP webhook notifications when tracks are scrobbled. This is useful for integrating with external services like home automation systems, Discord bots, monitoring tools, or any service that can receive HTTP requests.

## Features

- Sends HTTP GET requests to configured URLs on every scrobble event
- Includes track metadata (title, artist, album, username, timestamp) as query parameters
- Supports multiple webhook URLs (comma-separated)
- All users are automatically authorized (no external service authentication required)
- Now playing events are ignored (webhooks fire only on completed scrobbles)

## Prerequisites

- [Rust](https://rustup.rs/) toolchain
- WebAssembly target: `rustup target add wasm32-unknown-unknown`

## Building

From the `plugins/examples` directory:

```bash
make webhook-rs.ndp
```

Or build directly with cargo:

```bash
cd webhook-rs
cargo build --release
zip -j webhook-rs.ndp manifest.json target/wasm32-unknown-unknown/release/webhook_rs.wasm
```

## Installation

Copy `webhook-rs.ndp` to your Navidrome plugins folder (configured via `Plugins.Folder` in your config).

## Configuration

Configure in the Navidrome UI (Settings → Plugins → webhook-rs):

| Key    | Description                          | Example                                                   |
|--------|--------------------------------------|-----------------------------------------------------------|
| `urls` | Comma-separated list of webhook URLs | `https://example.com/hook1,https://example.com/hook2`     |

## Webhook Request Format

When a scrobble occurs, the plugin sends an HTTP GET request to each configured URL with the following query parameters:

| Parameter   | Description                                   |
|-------------|-----------------------------------------------|
| `title`     | Track title                                   |
| `artist`    | Track artist                                  |
| `album`     | Album name                                    |
| `user`      | Username who scrobbled                        |
| `timestamp` | Unix timestamp when the track started playing |

Example request:
```
GET https://example.com/webhook?title=Song%20Name&artist=Artist%20Name&album=Album%20Name&user=john&timestamp=1703270400
```

## Use Cases

- **Home Automation**: Trigger lights or displays when music starts playing
- **Discord/Slack Notifications**: Post currently playing tracks to a channel
- **Logging/Analytics**: Track listening history in an external system
- **IFTTT/Zapier Integration**: Connect to thousands of services via webhook triggers

## Development

The plugin is built using the [Extism Rust PDK](https://github.com/extism/rust-pdk). Key exports:

- `nd_manifest` - Returns plugin metadata and permissions
- `nd_scrobbler_is_authorized` - Always returns `true` (all users authorized)
- `nd_scrobbler_now_playing` - No-op (returns success without action)
- `nd_scrobbler_scrobble` - Sends webhooks to configured URLs
