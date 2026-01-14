//! Discord Rich Presence Plugin - RPC Communication
//!
//! This module handles all Discord gateway communication including WebSocket connections,
//! presence updates, and heartbeat management.

use extism_pdk::*;
use nd_pdk::host::{cache, scheduler, websocket};
use serde::{Deserialize, Serialize};

// ============================================================================
// Constants
// ============================================================================

const HEARTBEAT_OP_CODE: i32 = 1;
const GATE_OP_CODE: i32 = 2;
const PRESENCE_OP_CODE: i32 = 3;
const HEARTBEAT_INTERVAL: i32 = 41;
const DEFAULT_IMAGE: &str = "https://i.imgur.com/hb3XPzA.png";

const PAYLOAD_HEARTBEAT: &str = "heartbeat";

// ============================================================================
// Discord Types
// ============================================================================

#[derive(Serialize)]
pub struct Activity {
    pub name: String,
    #[serde(rename = "type")]
    pub activity_type: i32,
    pub details: String,
    pub state: String,
    #[serde(rename = "application_id")]
    pub application: String,
    pub timestamps: ActivityTimestamps,
    pub assets: ActivityAssets,
}

#[derive(Serialize)]
pub struct ActivityTimestamps {
    pub start: i64,
    pub end: i64,
}

#[derive(Serialize)]
pub struct ActivityAssets {
    pub large_image: String,
    pub large_text: String,
}

#[derive(Serialize)]
struct PresencePayload {
    activities: Vec<Activity>,
    since: i64,
    status: String,
    afk: bool,
}

#[derive(Serialize)]
struct IdentifyPayload {
    token: String,
    intents: i32,
    properties: IdentifyProperties,
}

#[derive(Serialize)]
struct IdentifyProperties {
    os: String,
    browser: String,
    device: String,
}

#[derive(Serialize)]
struct GatewayMessage<T> {
    op: i32,
    d: T,
}

#[derive(Deserialize)]
struct GatewayResponse {
    op: i32,
    #[serde(default)]
    #[allow(dead_code)]
    d: Option<serde_json::Value>,
    #[serde(default)]
    s: Option<i64>,
}

// ============================================================================
// Cache Keys
// ============================================================================

fn connection_key(username: &str) -> String {
    format!("discord.connection.{}", username)
}

fn token_key(username: &str) -> String {
    format!("discord.token.{}", username)
}

fn sequence_key(username: &str) -> String {
    format!("discord.sequence.{}", username)
}

// ============================================================================
// Connection Management
// ============================================================================

/// Tests if the connection is still valid by trying to send a heartbeat.
fn is_connected(username: &str) -> bool {
    match send_heartbeat(username) {
        Ok(_) => true,
        Err(e) => {
            trace!("Connection test failed for user {}: {:?}", username, e);
            false
        }
    }
}

/// Cleans up a connection for a user.
/// Called when heartbeat fails or connection is lost.
pub fn cleanup_connection(username: &str) {
    info!("Cleaning up failed connection for user {}", username);
    
    // Cancel the heartbeat schedule
    if let Err(e) = scheduler::cancel_schedule(username) {
        warn!("Failed to cancel heartbeat schedule for user {}: {:?}", username, e);
    }
    
    // Try to close the WebSocket connection
    let conn_key = connection_key(username);
    if let Ok(Some(conn_id)) = cache::get_string(&conn_key) {
        if !conn_id.is_empty() {
            if let Err(e) = websocket::close_connection(&conn_id, 1000, "Reconnecting") {
                trace!("Failed to close WebSocket for user {}: {:?}", username, e);
            }
            // Clean up reverse mapping
            let reverse_key = format!("discord.reverse.{}", conn_id);
            let _ = cache::remove(&reverse_key);
        }
    }
    
    // Clean up cache entries
    let _ = cache::remove(&conn_key);
    let _ = cache::remove(&sequence_key(username));
    
    info!("Cleaned up connection for user {}", username);
}

/// Handles connection close by connection ID (called from WebSocket close callback).
/// This cleans up all state associated with the connection.
pub fn handle_connection_close(connection_id: &str) {
    // Find the username for this connection using the reverse mapping
    if let Ok(Some(username)) = find_username_for_connection(connection_id) {
        info!("Connection closed for user {}, cleaning up", username);
        
        // Cancel the heartbeat schedule
        if let Err(e) = scheduler::cancel_schedule(&username) {
            // Not an error if schedule doesn't exist
            trace!("Failed to cancel heartbeat schedule for user {}: {:?}", username, e);
        }
        
        // Cancel any pending clear-activity schedule
        let _ = scheduler::cancel_schedule(&format!("{}-clear", username));
        
        // Clean up cache entries
        let conn_key = connection_key(&username);
        let _ = cache::remove(&conn_key);
        let _ = cache::remove(&sequence_key(&username));
        
        // Clean up reverse mapping
        let reverse_key = format!("discord.reverse.{}", connection_id);
        let _ = cache::remove(&reverse_key);
        
        info!("Cleaned up connection state for user {}", username);
    } else {
        // Just clean up the reverse mapping if we can't find the username
        let reverse_key = format!("discord.reverse.{}", connection_id);
        let _ = cache::remove(&reverse_key);
    }
}

/// Connects to the Discord gateway for a user.
pub fn connect(username: &str, token: &str) -> Result<(), Error> {
    // Check if already connected and connection is valid
    if is_connected(username) {
        info!("Reusing existing connection for user {}", username);
        return Ok(());
    }
    
    // Clean up any stale connection state
    cleanup_connection(username);

    info!("Connecting to Discord gateway for user {}", username);

    // Store token for later use
    cache::set_string(&token_key(username), token, 86400)?;

    // Get Discord Gateway URL
    let gateway = get_discord_gateway()?;
    info!("Using gateway: {}", gateway);

    // Connect to Discord gateway
    let headers = std::collections::HashMap::new();
    let conn_id = websocket::connect(
        &gateway,
        headers,
        username, // Use username as connection ID for easy lookup
    )?;
    info!("WebSocket connection established: {}", conn_id);

    // Store connection ID
    let conn_key = connection_key(username);
    cache::set_string(&conn_key, &conn_id, 86400)?;

    // Send identify immediately (don't wait for Hello)
    identify(username)?;

    info!("Successfully connected and identified user {}", username);
    Ok(())
}

/// Handles a WebSocket message from Discord.
pub fn handle_websocket_message(connection_id: &str, message: &str) -> Result<(), Error> {
    let response: GatewayResponse = serde_json::from_str(message)
        .map_err(|e| Error::msg(format!("Failed to parse gateway message: {}", e)))?;

    // Update sequence number if present
    if let Some(seq) = response.s {
        // Find username for this connection
        if let Some(username) = find_username_for_connection(connection_id)? {
            cache::set_string(&sequence_key(&username), &seq.to_string(), 86400)?;
        }
    }

    match response.op {
        10 => {
            // Hello - we already identified in connect(), nothing to do
        }
        11 => {
            // Heartbeat ACK - no action needed
        }
        1 => {
            // Heartbeat request - send heartbeat
            if let Some(username) = find_username_for_connection(connection_id)? {
                send_heartbeat(&username)?;
            }
        }
        _ => {
            trace!("Received Discord gateway op: {}", response.op);
        }
    }

    Ok(())
}

/// Handles heartbeat callback from scheduler.
pub fn handle_heartbeat_callback(username: &str) -> Result<(), Error> {
    send_heartbeat(username)
}

/// Handles clear activity callback from scheduler.
pub fn handle_clear_activity_callback(username: &str) -> Result<(), Error> {
    info!("Clearing activity for user {}", username);

    let conn_key = connection_key(username);
    if let Some(conn_id) = cache::get_string(&conn_key)?.filter(|s| !s.is_empty()) {
        // Send empty presence to clear activity
        let msg = GatewayMessage {
            op: PRESENCE_OP_CODE,
            d: PresencePayload {
                activities: vec![],
                since: 0,
                status: "dnd".to_string(),
                afk: false,
            },
        };

        let json = serde_json::to_string(&msg)
            .map_err(|e| Error::msg(format!("Failed to serialize message: {}", e)))?;

        websocket::send_text(&conn_id, &json)?;
    }

    Ok(())
}

/// Disconnects from Discord for a user.
pub fn disconnect(username: &str) -> Result<(), Error> {
    info!("Disconnecting from Discord for user {}", username);
    
    // Cancel the heartbeat schedule
    if let Err(e) = scheduler::cancel_schedule(username) {
        warn!("Failed to cancel heartbeat schedule: {:?}", e);
    }
    
    // Close the WebSocket connection
    let conn_key = connection_key(username);
    if let Some(conn_id) = cache::get_string(&conn_key)?.filter(|s| !s.is_empty()) {
        if let Err(e) = websocket::close_connection(&conn_id, 1000, "Navidrome disconnect") {
            warn!("Failed to close WebSocket connection: {:?}", e);
        }
        // Clean up reverse mapping
        let reverse_key = format!("discord.reverse.{}", conn_id);
        let _ = cache::remove(&reverse_key);
    }
    
    // Clean up cache entries
    let _ = cache::remove(&conn_key);
    let _ = cache::remove(&sequence_key(username));
    
    Ok(())
}

/// Sends an activity update to Discord.
pub fn send_activity(
    client_id: &str,
    username: &str,
    token: &str,
    mut activity: Activity,
) -> Result<(), Error> {
    let conn_key = connection_key(username);
    let conn_id = cache::get_string(&conn_key)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("Not connected to Discord"))?;

    // Process image URL
    activity.assets.large_image = process_image(&activity.assets.large_image, client_id, token)?;

    // Send presence update
    let msg = GatewayMessage {
        op: PRESENCE_OP_CODE,
        d: PresencePayload {
            activities: vec![activity],
            since: 0,
            status: "dnd".to_string(),
            afk: false,
        },
    };

    let json = serde_json::to_string(&msg)
        .map_err(|e| Error::msg(format!("Failed to serialize message: {}", e)))?;

    websocket::send_text(&conn_id, &json)?;

    Ok(())
}

// ============================================================================
// Internal Functions
// ============================================================================

fn find_username_for_connection(connection_id: &str) -> Result<Option<String>, Error> {
    // This is a simple approach - in production you might want to maintain a proper mapping
    // For now, we'll use a known pattern to find the username
    // The connection ID is stored as cache value, so we need to scan for it
    // Since we can't iterate cache, we'll use a workaround with a reverse mapping
    let reverse_key = format!("discord.reverse.{}", connection_id);
    Ok(cache::get_string(&reverse_key)?.filter(|s| !s.is_empty()))
}

fn get_discord_gateway() -> Result<String, Error> {
    let req = HttpRequest::new("https://discord.com/api/gateway")
        .with_method("GET");

    let resp = http::request::<String>(&req, None::<String>)?;
    if resp.status_code() >= 400 {
        return Err(Error::msg(format!(
            "Failed to get Discord gateway: HTTP {}",
            resp.status_code()
        )));
    }

    let body = resp.body();
    let data: std::collections::HashMap<String, String> = serde_json::from_slice(&body)
        .map_err(|e| Error::msg(format!("Failed to parse gateway response: {}", e)))?;

    data.get("url")
        .map(|url| url.to_string())
        .ok_or_else(|| Error::msg("No URL in gateway response"))
}

fn identify(username: &str) -> Result<(), Error> {
    info!("Identifying with Discord for user {}", username);

    let conn_key = connection_key(username);
    let conn_id = cache::get_string(&conn_key)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("No connection found"))?;

    let token_k = token_key(username);
    let token = cache::get_string(&token_k)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("No token found"))?;

    // Store reverse mapping for connection -> username
    let reverse_key = format!("discord.reverse.{}", conn_id);
    cache::set_string(&reverse_key, username, 86400)?;

    // Send identify
    let msg = GatewayMessage {
        op: GATE_OP_CODE,
        d: IdentifyPayload {
            token,
            intents: 0,
            properties: IdentifyProperties {
                os: "Windows 10".to_string(),
                browser: "Discord Client".to_string(),
                device: "Discord Client".to_string(),
            },
        },
    };

    let json = serde_json::to_string(&msg)
        .map_err(|e| Error::msg(format!("Failed to serialize message: {}", e)))?;

    websocket::send_text(&conn_id, &json)?;

    // Schedule heartbeat
    scheduler::schedule_recurring(
        &format!("@every {}s", HEARTBEAT_INTERVAL),
        PAYLOAD_HEARTBEAT,
        username,
    )?;

    Ok(())
}

fn send_heartbeat(username: &str) -> Result<(), Error> {
    let conn_key = connection_key(username);
    let conn_id = cache::get_string(&conn_key)?
        .filter(|s| !s.is_empty())
        .ok_or_else(|| Error::msg("No connection found"))?;

    // Get sequence number
    let seq_key = sequence_key(username);
    let seq: Option<i64> = cache::get_string(&seq_key)?
        .and_then(|s| s.parse().ok());

    // Send heartbeat
    let msg = GatewayMessage {
        op: HEARTBEAT_OP_CODE,
        d: seq,
    };

    let json = serde_json::to_string(&msg)
        .map_err(|e| Error::msg(format!("Failed to serialize message: {}", e)))?;

    websocket::send_text(&conn_id, &json)?;
    Ok(())
}

fn process_image(image_url: &str, client_id: &str, token: &str) -> Result<String, Error> {
    process_image_inner(image_url, client_id, token, false)
}

fn process_image_inner(
    image_url: &str,
    client_id: &str,
    token: &str,
    is_default: bool,
) -> Result<String, Error> {
    let url = if image_url.is_empty() {
        if is_default {
            return Err(Error::msg("default image URL is empty"));
        }
        return process_image_inner(DEFAULT_IMAGE, client_id, token, true);
    } else {
        image_url
    };

    // Already processed
    if url.starts_with("mp:") {
        return Ok(url.to_string());
    }

    // Check cache
    let cache_key = format!("discord.image.{:x}", md5_hash(url));
    if let Some(cached) = cache::get_string(&cache_key)?.filter(|s| !s.is_empty()) {
        return Ok(cached);
    }

    // Process via Discord API
    let body = format!(r#"{{"urls":["{}"]}}"#, url);
    let api_url = format!(
        "https://discord.com/api/v9/applications/{}/external-assets",
        client_id
    );

    let req = HttpRequest::new(&api_url)
        .with_method("POST")
        .with_header("Authorization", token)
        .with_header("Content-Type", "application/json");

    let resp = http::request::<String>(&req, Some(body))?;
    if resp.status_code() >= 400 {
        if is_default {
            return Err(Error::msg(format!(
                "failed to process default image: HTTP {}",
                resp.status_code()
            )));
        }
        return process_image_inner(DEFAULT_IMAGE, client_id, token, true);
    }

    let body = resp.body();
    let data: Vec<std::collections::HashMap<String, String>> = serde_json::from_slice(&body)
        .map_err(|e| Error::msg(format!("Failed to parse image response: {}", e)))?;

    if data.is_empty() {
        if is_default {
            return Err(Error::msg("no data returned for default image"));
        }
        return process_image_inner(DEFAULT_IMAGE, client_id, token, true);
    }

    let asset_path = data[0]
        .get("external_asset_path")
        .map(|s| s.as_str())
        .unwrap_or("");

    if asset_path.is_empty() {
        if is_default {
            return Err(Error::msg("empty external_asset_path for default image"));
        }
        return process_image_inner(DEFAULT_IMAGE, client_id, token, true);
    }

    let processed = format!("mp:{}", asset_path);

    // Cache the result
    let ttl = if is_default { 48 * 60 * 60 } else { 4 * 60 * 60 };
    let _ = cache::set_string(&cache_key, &processed, ttl);

    Ok(processed)
}

/// Simple hash function for cache keys.
fn md5_hash(input: &str) -> u64 {
    // A simple hash - not actual MD5, but sufficient for cache keys
    let mut hash: u64 = 0;
    for (i, byte) in input.bytes().enumerate() {
        hash = hash.wrapping_add((byte as u64).wrapping_mul((i as u64).wrapping_add(1)));
        hash = hash.wrapping_mul(31);
    }
    hash
}
