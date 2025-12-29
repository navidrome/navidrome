//! Webhook Scrobbler Plugin for Navidrome
//!
//! This plugin demonstrates how to build a Navidrome plugin in Rust using the Extism PDK.
//! It implements the Scrobbler capability and sends HTTP GET requests to configured URLs
//! whenever a track is scrobbled.
//!
//! ## Configuration
//!
//! Set the `urls` config key to a comma-separated list of webhook URLs:
//! ```toml
//! [PluginConfig.webhook-rs]
//! urls = "https://example.com/webhook1,https://example.com/webhook2"
//! ```

use extism_pdk::*;
use serde::{Deserialize, Serialize};

// ============================================================================
// Scrobbler Types
// ============================================================================

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct AuthInput {
    user_id: String,
    username: String,
}

#[derive(Serialize)]
struct AuthOutput {
    authorized: bool,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)] // Fields are deserialized from JSON but not all are used
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
#[allow(dead_code)] // Fields are deserialized from JSON but not all are used
struct NowPlayingInput {
    user_id: String,
    username: String,
    track: TrackInfo,
    position: i32,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)] // Fields are deserialized from JSON but not all are used
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

// ============================================================================
// Plugin Exports
// ============================================================================

/// Checks if a user is authorized. This plugin authorizes all users.
#[plugin_fn]
pub fn nd_scrobbler_is_authorized(Json(input): Json<AuthInput>) -> FnResult<Json<AuthOutput>> {
    info!(
        "Authorization check for user: {} ({})",
        input.username, input.user_id
    );
    Ok(Json(AuthOutput { authorized: true }))
}

/// Handles now playing notifications. This plugin ignores them (webhooks only on scrobble).
#[plugin_fn]
pub fn nd_scrobbler_now_playing(Json(input): Json<NowPlayingInput>) -> FnResult<Json<ScrobblerOutput>> {
    info!(
        "Now playing (ignored): {} - {} for user {}",
        input.track.artist, input.track.title, input.username
    );
    Ok(Json(ScrobblerOutput::default()))
}

/// Handles scrobble events by sending HTTP GET requests to configured URLs.
#[plugin_fn]
pub fn nd_scrobbler_scrobble(Json(input): Json<ScrobbleInput>) -> FnResult<Json<ScrobblerOutput>> {
    // Get configured URLs
    let urls_config = match config::get("urls") {
        Ok(Some(urls)) if !urls.is_empty() => urls,
        _ => {
            warn!("No webhook URLs configured. Set 'urls' in plugin config.");
            return Ok(Json(ScrobblerOutput::default()));
        }
    };

    info!(
        "Scrobble: {} - {} by user {}",
        input.track.artist, input.track.title, input.username
    );

    // Build query parameters
    let query = format!(
        "?title={}&artist={}&album={}&user={}&timestamp={}",
        urlencod(&input.track.title),
        urlencod(&input.track.artist),
        urlencod(&input.track.album),
        urlencod(&input.username),
        input.timestamp
    );

    // Send requests to each configured URL
    for url in urls_config.split(',') {
        let url = url.trim();
        if url.is_empty() {
            continue;
        }

        let full_url = format!("{}{}", url, query);
        info!("Sending webhook to: {}", full_url);

        let req = HttpRequest::new(&full_url);
        match http::request::<()>(&req, None) {
            Ok(res) => {
                let status = res.status_code();
                if status >= 200 && status < 300 {
                    info!("Webhook succeeded: {} (status {})", url, status);
                } else {
                    warn!("Webhook returned non-2xx status: {} (status {})", url, status);
                }
            }
            Err(e) => {
                error!("Webhook failed for {}: {:?}", url, e);
            }
        }
    }

    Ok(Json(ScrobblerOutput::default()))
}

/// Simple URL encoding for query parameters.
fn urlencod(s: &str) -> String {
    let mut result = String::with_capacity(s.len() * 3);
    for c in s.chars() {
        match c {
            'A'..='Z' | 'a'..='z' | '0'..='9' | '-' | '_' | '.' | '~' => result.push(c),
            ' ' => result.push_str("%20"),
            _ => {
                for b in c.to_string().as_bytes() {
                    result.push_str(&format!("%{:02X}", b));
                }
            }
        }
    }
    result
}
