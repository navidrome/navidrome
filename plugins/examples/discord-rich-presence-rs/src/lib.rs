//! Discord Rich Presence Plugin for Navidrome - Rust Implementation
//!
//! This plugin integrates Navidrome with Discord Rich Presence. It demonstrates how to:
//! - Use the nd-pdk crate for host service calls
//! - Implement the Scrobbler capability for now-playing updates
//! - Implement SchedulerCallback for heartbeat and activity clearing
//! - Implement WebSocketCallback for Discord gateway communication
//!
//! ## Configuration
//!
//! ```toml
//! [PluginConfig.discord-rich-presence-rs]
//! clientid = "YOUR_DISCORD_APPLICATION_ID"
//! "user.username1" = "discord_token1"
//! "user.username2" = "discord_token2"
//! ```
//!
//! **WARNING**: This plugin is for demonstration purposes only. Storing Discord tokens
//! in configuration files is not secure and may violate Discord's terms of service.

use extism_pdk::*;
use nd_pdk::host::{artwork, config, scheduler};
use nd_pdk::scrobbler::{
    Error as ScrobblerError, IsAuthorizedRequest, NowPlayingRequest,
    ScrobbleRequest, Scrobbler, SCROBBLER_ERROR_NOT_AUTHORIZED, SCROBBLER_ERROR_RETRY_LATER,
};
use nd_pdk::scheduler::{
    CallbackProvider, Error as SchedulerError, SchedulerCallbackRequest,
};
use nd_pdk::websocket::{
    BinaryMessageProvider, CloseProvider, Error as WebSocketError, ErrorProvider,
    OnBinaryMessageRequest, OnCloseRequest, OnErrorRequest, OnTextMessageRequest,
    TextMessageProvider,
};

mod rpc;

// Register capabilities using PDK macros
nd_pdk::register_scrobbler!(DiscordPlugin);
nd_pdk::register_scheduler_callback!(DiscordPlugin);
nd_pdk::register_websocket_text_message!(DiscordPlugin);
nd_pdk::register_websocket_binary_message!(DiscordPlugin);
nd_pdk::register_websocket_error!(DiscordPlugin);
nd_pdk::register_websocket_close!(DiscordPlugin);

// ============================================================================
// Constants
// ============================================================================

const CLIENT_ID_KEY: &str = "clientid";
const USER_KEY_PREFIX: &str = "user.";
const PAYLOAD_HEARTBEAT: &str = "heartbeat";
const PAYLOAD_CLEAR_ACTIVITY: &str = "clear-activity";

// ============================================================================
// Plugin Implementation
// ============================================================================

/// The Discord Rich Presence plugin type.
#[derive(Default)]
struct DiscordPlugin;

// ============================================================================
// Configuration
// ============================================================================

fn get_config() -> Result<(String, std::collections::HashMap<String, String>), Error> {
    let client_id = config::get(CLIENT_ID_KEY)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("missing clientid in configuration"))?;

    // Get all user keys with the "user." prefix
    let user_keys = config::keys(USER_KEY_PREFIX)?;

    let mut users = std::collections::HashMap::new();
    for key in user_keys {
        let username = key.strip_prefix(USER_KEY_PREFIX).unwrap_or(&key);
        if let Some(token) = config::get(&key)?.filter(|s| !s.is_empty()) {
            users.insert(username.to_string(), token);
        }
    }

    Ok((client_id, users))
}

fn get_image_url(track_id: &str) -> String {
    match artwork::get_track_url(track_id, 300) {
        Ok(url) => {
            if url.starts_with("http://localhost") {
                String::new()
            } else {
                url
            }
        }
        Err(e) => {
            warn!("Failed to get artwork URL: {:?}", e);
            String::new()
        }
    }
}

// ============================================================================
// Scrobbler Implementation
// ============================================================================

impl Scrobbler for DiscordPlugin {
    fn is_authorized(&self, req: IsAuthorizedRequest) -> Result<bool, ScrobblerError> {
        let (_, users) = match get_config() {
            Ok(config) => config,
            Err(e) => {
                error!("Failed to get config: {:?}", e);
                return Ok(false);
            }
        };

        let authorized = users.contains_key(&req.username);
        info!("IsAuthorized for user {}: {}", req.username, authorized);
        Ok(authorized)
    }

    fn now_playing(&self, req: NowPlayingRequest) -> Result<(), ScrobblerError> {
        info!(
            "Setting presence for user {}, track: {}",
            req.username, req.track.title
        );

        // Load configuration
        let (client_id, users) = get_config()
            .map_err(|e| ScrobblerError::new(format!("{}: failed to get config: {:?}", SCROBBLER_ERROR_RETRY_LATER, e)))?;

        // Check authorization
        let user_token = users.get(&req.username).cloned().ok_or_else(|| {
            ScrobblerError::new(format!(
                "{}: user '{}' not authorized",
                SCROBBLER_ERROR_NOT_AUTHORIZED, req.username
            ))
        })?;

        // Connect to Discord
        rpc::connect(&req.username, &user_token)
            .map_err(|e| ScrobblerError::new(format!(
                "{}: failed to connect to Discord: {:?}",
                SCROBBLER_ERROR_RETRY_LATER, e
            )))?;

        // Cancel any existing completion schedule
        let _ = scheduler::cancel_schedule(&format!("{}-clear", req.username));

        // Calculate timestamps
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_secs() as i64)
            .unwrap_or(0);
        let start_time = (now - req.position as i64) * 1000;
        let end_time = start_time + (req.track.duration as i64) * 1000;

        // Send activity update
        rpc::send_activity(
            &client_id,
            &req.username,
            &user_token,
            rpc::Activity {
                application: client_id.clone(),
                name: "Navidrome".to_string(),
                activity_type: 2, // Listening
                details: req.track.title.clone(),
                state: req.track.artist.clone(),
                timestamps: rpc::ActivityTimestamps {
                    start: start_time,
                    end: end_time,
                },
                assets: rpc::ActivityAssets {
                    large_image: get_image_url(&req.track.id),
                    large_text: req.track.album.clone(),
                },
            },
        )
        .map_err(|e| ScrobblerError::new(format!(
            "{}: failed to send activity: {:?}",
            SCROBBLER_ERROR_RETRY_LATER, e
        )))?;

        // Schedule a timer to clear the activity after the track completes
        let remaining_seconds = (req.track.duration as i32) - req.position + 5;
        if let Err(e) = scheduler::schedule_one_time(
            remaining_seconds,
            PAYLOAD_CLEAR_ACTIVITY,
            &format!("{}-clear", req.username),
        ) {
            warn!("Failed to schedule completion timer: {:?}", e);
        }

        Ok(())
    }

    fn scrobble(&self, _req: ScrobbleRequest) -> Result<(), ScrobblerError> {
        // Discord Rich Presence doesn't need scrobble events - success
        Ok(())
    }
}

// ============================================================================
// Scheduler Callback Implementation
// ============================================================================

impl CallbackProvider for DiscordPlugin {
    fn on_callback(&self, req: SchedulerCallbackRequest) -> Result<(), SchedulerError> {
        match req.payload.as_str() {
            PAYLOAD_HEARTBEAT => {
                // Heartbeat callback - schedule_id is the username
                if let Err(e) = rpc::handle_heartbeat_callback(&req.schedule_id) {
                    // On heartbeat failure, clean up the connection (like the original Go plugin)
                    // The next NowPlaying call will reconnect if needed
                    warn!("Heartbeat failed for user {}, cleaning up connection: {:?}", req.schedule_id, e);
                    rpc::cleanup_connection(&req.schedule_id);
                    return Err(SchedulerError::new(format!("heartbeat failed, connection cleaned up: {}", e)));
                }
            }
            PAYLOAD_CLEAR_ACTIVITY => {
                // Clear activity callback - schedule_id is "username-clear"
                let username = req.schedule_id.trim_end_matches("-clear");
                info!("Removing presence for user {}", username);
                rpc::handle_clear_activity_callback(username)
                    .map_err(|e| SchedulerError::new(e.to_string()))?;
                info!("Disconnecting user {}", username);
                rpc::disconnect(username)
                    .map_err(|e| SchedulerError::new(e.to_string()))?;
            }
            _ => {
                warn!("Unknown scheduler callback payload: {}", req.payload);
            }
        }

        Ok(())
    }
}

// ============================================================================
// WebSocket Callback Implementations
// ============================================================================

impl TextMessageProvider for DiscordPlugin {
    fn on_text_message(&self, req: OnTextMessageRequest) -> Result<(), WebSocketError> {
        rpc::handle_websocket_message(&req.connection_id, &req.message)
            .map_err(|e| WebSocketError::new(e.to_string()))?;
        Ok(())
    }
}

impl BinaryMessageProvider for DiscordPlugin {
    fn on_binary_message(&self, _req: OnBinaryMessageRequest) -> Result<(), WebSocketError> {
        // Binary messages are not expected from Discord
        Ok(())
    }
}

impl ErrorProvider for DiscordPlugin {
    fn on_error(&self, req: OnErrorRequest) -> Result<(), WebSocketError> {
        warn!(
            "WebSocket error for connection '{}': {}",
            req.connection_id, req.error
        );
        // Clean up all state associated with this connection since it's likely broken
        rpc::handle_connection_close(&req.connection_id);
        Ok(())
    }
}

impl CloseProvider for DiscordPlugin {
    fn on_close(&self, req: OnCloseRequest) -> Result<(), WebSocketError> {
        info!(
            "WebSocket connection '{}' closed with code {}: {}",
            req.connection_id, req.code, req.reason
        );
        // Clean up all state associated with this connection
        rpc::handle_connection_close(&req.connection_id);
        Ok(())
    }
}
