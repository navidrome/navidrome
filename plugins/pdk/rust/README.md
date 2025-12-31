# Navidrome Plugin Development Kit for Rust

This directory contains the Rust PDK crates for building Navidrome plugins.

## Crate Structure

```
plugins/pdk/rust/
├── nd-pdk/              # Umbrella crate - use this as your dependency
├── nd-pdk-host/         # Host function wrappers (call Navidrome services)
└── nd-pdk-capabilities/ # Capability traits and types (generated)
```

## Usage

Add the `nd-pdk` crate as a dependency in your plugin's `Cargo.toml`:

```toml
[package]
name = "my-plugin"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
nd-pdk = { path = "../../pdk/rust/nd-pdk" }
extism-pdk = "1.2"
```

### Implementing a Scrobbler (Required-All Pattern)

The Scrobbler capability requires all methods to be implemented:

```rust
use nd_pdk::scrobbler::{
    Error, IsAuthorizedRequest,
    NowPlayingRequest, ScrobbleRequest, Scrobbler,
};

// Register WASM exports for all Scrobbler methods
nd_pdk::register_scrobbler!(MyPlugin);

#[derive(Default)]
struct MyPlugin;

impl Scrobbler for MyPlugin {
    fn is_authorized(&self, req: IsAuthorizedRequest) -> Result<bool, Error> {
        Ok(true)
    }

    fn now_playing(&self, req: NowPlayingRequest) -> Result<(), Error> {
        // Handle now playing notification
        Ok(())
    }

    fn scrobble(&self, req: ScrobbleRequest) -> Result<(), Error> {
        // Submit scrobble
        Ok(())
    }
}
```

### Implementing Metadata Agent (Optional Pattern)

The MetadataAgent capability allows implementing individual methods:

```rust
use nd_pdk::metadata::{
    ArtistBiographyProvider, GetArtistBiographyRequest, ArtistBiography, Error,
};

// Register only the methods you implement
nd_pdk::register_artist_biography!(MyPlugin);

#[derive(Default)]
struct MyPlugin;

impl ArtistBiographyProvider for MyPlugin {
    fn get_artist_biography(&self, req: GetArtistBiographyRequest) 
        -> Result<ArtistBiography, Error> 
    {
        // Return artist biography
        Ok(ArtistBiography {
            biography: "Artist bio text...".into(),
            ..Default::default()
        })
    }
}
```

### Using Host Services

Access Navidrome services via the host module:

```rust
use nd_pdk::host::{artwork, scheduler, library};

// Get artwork URL for a track
let url = artwork::get_track_url("track-id", 300)?;

// Schedule a one-time callback
scheduler::schedule_one_time(60, "my-payload", "schedule-id")?;

// Get library information
let libs = library::get_all()?;
```

## Available Capabilities

| Capability  | Pattern      | Description                                         |
|-------------|--------------|-----------------------------------------------------|
| `scrobbler` | Required-all | Submit listening history to external services       |
| `metadata`  | Optional     | Provide artist/album metadata from external sources |
| `lifecycle` | Optional     | Handle plugin initialization                        |
| `scheduler` | Optional     | Receive scheduled callbacks                         |
| `websocket` | Optional     | Handle WebSocket messages                           |

## Building

Rust plugins must be compiled to WASM using the `wasm32-wasip1` target:

```bash
cargo build --release --target wasm32-wasip1
```

The resulting `.wasm` file can be packaged into an `.ndp` plugin package.

## Examples

See the example plugins for complete implementations:

- [webhook-rs](../../examples/webhook-rs/) - Simple scrobbler using the PDK
- [discord-rich-presence-rs](../../examples/discord-rich-presence-rs/) - Complex plugin with multiple capabilities
- [library-inspector-rs](../../examples/library-inspector-rs/) - Host service demonstration

## Code Generation

The capability modules in `nd-pdk-capabilities` are auto-generated from the Go capability definitions. To regenerate after capability changes:

```bash
make gen
```

This generates both Go and Rust PDK code.
