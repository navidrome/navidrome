//! Discord Rich Presence Plugin for Navidrome - Rust Implementation
//!
//! This plugin integrates Navidrome with Discord Rich Presence. It demonstrates how to:
//! - Use the generated nd-host wrappers for host service calls
//! - Implement the Scrobbler capability for now-playing updates
//! - Implement SchedulerCallback for heartbeat and activity clearing
//! - Implement WebSocketCallback for Discord gateway communication
//!
//! ## Configuration
//!
//! ```toml
//! [PluginConfig.discord-rich-presence-rs]
//! clientid = "YOUR_DISCORD_APPLICATION_ID"
//! users = "username1:discord_token1,username2:discord_token2"
//! ```
//!
//! **WARNING**: This plugin is for demonstration purposes only. Storing Discord tokens
//! in configuration files is not secure and may violate Discord's terms of service.

use extism_pdk::*;
use nd_host::{artwork, scheduler};
use serde::{Deserialize, Serialize};

mod rpc;

// ============================================================================
// Constants
// ============================================================================

const CLIENT_ID_KEY: &str = "clientid";
const USERS_KEY: &str = "users";
const PAYLOAD_HEARTBEAT: &str = "heartbeat";
const PAYLOAD_CLEAR_ACTIVITY: &str = "clear-activity";

// ============================================================================
// Configuration
// ============================================================================

fn get_config() -> Result<(String, std::collections::HashMap<String, String>), Error> {
    let client_id = config::get(CLIENT_ID_KEY)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("missing clientid in configuration"))?;

    let users_config = config::get(USERS_KEY)?
        .filter(|s| !s.is_empty())
        .unwrap_or_default();

    let mut users = std::collections::HashMap::new();
    for user in users_config.split(',') {
        let parts: Vec<&str> = user.split(':').collect();
        if parts.len() == 2 {
            users.insert(parts[0].trim().to_string(), parts[1].trim().to_string());
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
// Scrobbler Types
// ============================================================================

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct AuthInput {
    #[allow(dead_code)]
    user_id: String,
    username: String,
}

#[derive(Serialize)]
struct AuthOutput {
    authorized: bool,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct TrackInfo {
    id: String,
    title: String,
    album: String,
    artist: String,
    album_artist: String,
    duration: f32,
    track_number: i32,
    disc_number: i32,
    #[serde(default)]
    mbz_recording_id: Option<String>,
    #[serde(default)]
    mbz_album_id: Option<String>,
    #[serde(default)]
    mbz_artist_id: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct NowPlayingInput {
    user_id: String,
    username: String,
    track: TrackInfo,
    position: i32,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct ScrobbleInput {
    user_id: String,
    username: String,
    track: TrackInfo,
    timestamp: i64,
}

#[derive(Serialize, Default)]
#[serde(rename_all = "camelCase")]
struct ScrobblerOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error_type: Option<String>,
}

const ERROR_TYPE_NOT_AUTHORIZED: &str = "not_authorized";
const ERROR_TYPE_RETRY_LATER: &str = "retry_later";

// ============================================================================
// Scheduler Callback Types
// ============================================================================

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct SchedulerCallbackInput {
    schedule_id: String,
    payload: String,
    is_recurring: bool,
}

#[derive(Serialize, Default)]
struct SchedulerCallbackOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

// ============================================================================
// WebSocket Callback Types
// ============================================================================

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct OnTextMessageInput {
    connection_id: String,
    message: String,
}

#[derive(Serialize, Default)]
struct OnTextMessageOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct OnBinaryMessageInput {
    connection_id: String,
    message: Vec<u8>,
}

#[derive(Serialize, Default)]
struct OnBinaryMessageOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct OnErrorInput {
    connection_id: String,
    error: String,
}

#[derive(Serialize, Default)]
struct OnErrorOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct OnCloseInput {
    connection_id: String,
    code: i32,
    reason: String,
}

#[derive(Serialize, Default)]
struct OnCloseOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

// ============================================================================
// Scrobbler Plugin Exports
// ============================================================================

/// Checks if a user is authorized for Discord Rich Presence.
#[plugin_fn]
pub fn nd_scrobbler_is_authorized(Json(input): Json<AuthInput>) -> FnResult<Json<AuthOutput>> {
    let (_, users) = match get_config() {
        Ok(config) => config,
        Err(e) => {
            error!("Failed to get config: {:?}", e);
            return Ok(Json(AuthOutput { authorized: false }));
        }
    };

    let authorized = users.contains_key(&input.username);
    info!(
        "IsAuthorized for user {}: {}",
        input.username, authorized
    );
    Ok(Json(AuthOutput { authorized }))
}

/// Sends a now playing notification to Discord.
#[plugin_fn]
pub fn nd_scrobbler_now_playing(
    Json(input): Json<NowPlayingInput>,
) -> FnResult<Json<ScrobblerOutput>> {
    info!(
        "Setting presence for user {}, track: {}",
        input.username, input.track.title
    );

    // Load configuration
    let (client_id, users) = match get_config() {
        Ok(config) => config,
        Err(e) => {
            let err_msg = format!("failed to get config: {:?}", e);
            return Ok(Json(ScrobblerOutput {
                error: Some(err_msg),
                error_type: Some(ERROR_TYPE_RETRY_LATER.to_string()),
            }));
        }
    };

    // Check authorization
    let user_token = match users.get(&input.username) {
        Some(token) => token.clone(),
        None => {
            let err_msg = format!("user '{}' not authorized", input.username);
            return Ok(Json(ScrobblerOutput {
                error: Some(err_msg),
                error_type: Some(ERROR_TYPE_NOT_AUTHORIZED.to_string()),
            }));
        }
    };

    // Connect to Discord
    if let Err(e) = rpc::connect(&input.username, &user_token) {
        let err_msg = format!("failed to connect to Discord: {:?}", e);
        return Ok(Json(ScrobblerOutput {
            error: Some(err_msg),
            error_type: Some(ERROR_TYPE_RETRY_LATER.to_string()),
        }));
    }

    // Cancel any existing completion schedule
    let _ = scheduler::cancel_schedule(&format!("{}-clear", input.username));

    // Calculate timestamps
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0);
    let start_time = (now - input.position as i64) * 1000;
    let end_time = start_time + (input.track.duration as i64) * 1000;

    // Send activity update
    if let Err(e) = rpc::send_activity(
        &client_id,
        &input.username,
        &user_token,
        rpc::Activity {
            application: client_id.clone(),
            name: "Navidrome".to_string(),
            activity_type: 2, // Listening
            details: input.track.title.clone(),
            state: input.track.artist.clone(),
            timestamps: rpc::ActivityTimestamps {
                start: start_time,
                end: end_time,
            },
            assets: rpc::ActivityAssets {
                large_image: get_image_url(&input.track.id),
                large_text: input.track.album.clone(),
            },
        },
    ) {
        let err_msg = format!("failed to send activity: {:?}", e);
        return Ok(Json(ScrobblerOutput {
            error: Some(err_msg),
            error_type: Some(ERROR_TYPE_RETRY_LATER.to_string()),
        }));
    }

    // Schedule a timer to clear the activity after the track completes
    let remaining_seconds = (input.track.duration as i32) - input.position + 5;
    if let Err(e) = scheduler::schedule_one_time(
        remaining_seconds,
        PAYLOAD_CLEAR_ACTIVITY,
        &format!("{}-clear", input.username),
    ) {
        warn!("Failed to schedule completion timer: {:?}", e);
    }

    Ok(Json(ScrobblerOutput::default()))
}

/// Handles scrobble requests (no-op for Discord Rich Presence).
#[plugin_fn]
pub fn nd_scrobbler_scrobble(_input: Json<ScrobbleInput>) -> FnResult<Json<ScrobblerOutput>> {
    // Discord Rich Presence doesn't need scrobble events
    Ok(Json(ScrobblerOutput::default()))
}

// ============================================================================
// Scheduler Callback Export
// ============================================================================

/// Handles scheduler callbacks for heartbeat and activity clearing.
#[plugin_fn]
pub fn nd_scheduler_callback(
    Json(input): Json<SchedulerCallbackInput>,
) -> FnResult<Json<SchedulerCallbackOutput>> {

    match input.payload.as_str() {
        PAYLOAD_HEARTBEAT => {
            // Heartbeat callback - schedule_id is the username
            if let Err(e) = rpc::handle_heartbeat_callback(&input.schedule_id) {
                return Ok(Json(SchedulerCallbackOutput {
                    error: Some(e.to_string()),
                }));
            }
        }
        PAYLOAD_CLEAR_ACTIVITY => {
            // Clear activity callback - schedule_id is "username-clear"
            let username = input.schedule_id.trim_end_matches("-clear");
            if let Err(e) = rpc::handle_clear_activity_callback(username) {
                return Ok(Json(SchedulerCallbackOutput {
                    error: Some(e.to_string()),
                }));
            }
        }
        _ => {
            warn!("Unknown scheduler callback payload: {}", input.payload);
        }
    }

    Ok(Json(SchedulerCallbackOutput::default()))
}

// ============================================================================
// WebSocket Callback Exports
// ============================================================================

/// Handles incoming WebSocket text messages.
#[plugin_fn]
pub fn nd_websocket_on_text_message(
    Json(input): Json<OnTextMessageInput>,
) -> FnResult<Json<OnTextMessageOutput>> {
    if let Err(e) = rpc::handle_websocket_message(&input.connection_id, &input.message) {
        return Ok(Json(OnTextMessageOutput {
            error: Some(e.to_string()),
        }));
    }
    Ok(Json(OnTextMessageOutput::default()))
}

/// Handles incoming WebSocket binary messages.
#[plugin_fn]
pub fn nd_websocket_on_binary_message(
    Json(_input): Json<OnBinaryMessageInput>,
) -> FnResult<Json<OnBinaryMessageOutput>> {
    // Binary messages are not expected from Discord
    Ok(Json(OnBinaryMessageOutput::default()))
}

/// Handles WebSocket errors.
#[plugin_fn]
pub fn nd_websocket_on_error(Json(input): Json<OnErrorInput>) -> FnResult<Json<OnErrorOutput>> {
    warn!(
        "WebSocket error for connection '{}': {}",
        input.connection_id, input.error
    );
    Ok(Json(OnErrorOutput::default()))
}

/// Handles WebSocket connection closure.
#[plugin_fn]
pub fn nd_websocket_on_close(Json(input): Json<OnCloseInput>) -> FnResult<Json<OnCloseOutput>> {
    info!(
        "WebSocket connection '{}' closed with code {}: {}",
        input.connection_id, input.code, input.reason
    );
    Ok(Json(OnCloseOutput::default()))
}
