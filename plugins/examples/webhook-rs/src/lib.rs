//! Webhook Scrobbler Plugin for Navidrome
//!
//! This plugin demonstrates how to build a Navidrome plugin in Rust using the nd-pdk crate.
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

use extism_pdk::{config, error, http, info, warn, HttpRequest};
use nd_pdk::scrobbler::{
    Error, IsAuthorizedRequest, NowPlayingRequest, ScrobbleRequest,
    Scrobbler,
};

// Register the WASM exports for the Scrobbler capability
nd_pdk::register_scrobbler!(WebhookPlugin);

// ============================================================================
// Plugin Implementation
// ============================================================================

/// The webhook plugin type. Implements the Scrobbler trait.
#[derive(Default)]
struct WebhookPlugin;

impl Scrobbler for WebhookPlugin {
    /// Checks if a user is authorized. This plugin authorizes all users.
    fn is_authorized(&self, req: IsAuthorizedRequest) -> Result<bool, Error> {
        info!("Authorization check for user: {}", req.username);
        Ok(true)
    }

    /// Handles now playing notifications. This plugin ignores them (webhooks only on scrobble).
    fn now_playing(&self, req: NowPlayingRequest) -> Result<(), Error> {
        info!(
            "Now playing (ignored): {} - {} for user {}",
            req.track.artist, req.track.title, req.username
        );
        Ok(())
    }

    /// Handles scrobble events by sending HTTP GET requests to configured URLs.
    fn scrobble(&self, req: ScrobbleRequest) -> Result<(), Error> {
        // Get configured URLs
        let urls_config = match config::get("urls") {
            Ok(Some(urls)) if !urls.is_empty() => urls,
            _ => {
                warn!("No webhook URLs configured. Set 'urls' in plugin config.");
                return Ok(());
            }
        };

        info!(
            "Scrobble: {} - {} by user {}",
            req.track.artist, req.track.title, req.username
        );

        // Build query parameters
        let query = format!(
            "?title={}&artist={}&album={}&user={}&timestamp={}",
            urlencode(&req.track.title),
            urlencode(&req.track.artist),
            urlencode(&req.track.album),
            urlencode(&req.username),
            req.timestamp
        );

        // Send requests to each configured URL
        for url in urls_config.split(',') {
            let url = url.trim();
            if url.is_empty() {
                continue;
            }

            let full_url = format!("{}{}", url, query);
            info!("Sending webhook to: {}", full_url);

            let http_req = HttpRequest::new(&full_url);
            match http::request::<()>(&http_req, None) {
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

        Ok(())
    }
}

/// Simple URL encoding for query parameters.
fn urlencode(s: &str) -> String {
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
