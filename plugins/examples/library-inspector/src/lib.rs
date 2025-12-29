//! Library Inspector Plugin for Navidrome
//!
//! This plugin demonstrates how to use the Library host service in Rust.
//! It periodically logs details about all music libraries and finds the largest
//! file in the root of each library directory.
//!
//! ## Configuration
//!
//! Set the `cron` config key to customize the schedule (default: "@every 1m"):
//! ```toml
//! [PluginConfig.library-inspector]
//! cron = "@every 5m"
//! ```

use extism_pdk::*;
use serde::{Deserialize, Serialize};
use std::fs;

// ============================================================================
// Library Types
// ============================================================================

#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct Library {
    id: i32,
    name: String,
    #[serde(default)]
    path: Option<String>,
    #[serde(default)]
    mount_point: Option<String>,
    last_scan_at: i64,
    total_songs: i32,
    total_albums: i32,
    total_artists: i32,
    total_size: i64,
    total_duration: f64,
}

#[derive(Deserialize)]
struct LibraryGetAllLibrariesResponse {
    result: Option<Vec<Library>>,
    #[serde(default)]
    error: Option<String>,
}

// ============================================================================
// Scheduler Types
// ============================================================================

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct SchedulerScheduleRecurringRequest {
    cron_expression: String,
    payload: String,
    schedule_id: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct SchedulerScheduleRecurringResponse {
    #[serde(default)]
    new_schedule_id: Option<String>,
    #[serde(default)]
    error: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct SchedulerCallbackInput {
    schedule_id: String,
    payload: String,
    is_recurring: bool,
}

// ============================================================================
// Lifecycle Types
// ============================================================================

#[derive(Serialize, Default)]
struct InitOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

// ============================================================================
// Host Function Imports
// ============================================================================

#[host_fn]
extern "ExtismHost" {
    fn library_getalllibraries(input: Json<serde_json::Value>) -> Json<LibraryGetAllLibrariesResponse>;
    fn scheduler_schedulerecurring(input: Json<SchedulerScheduleRecurringRequest>) -> Json<SchedulerScheduleRecurringResponse>;
}

// ============================================================================
// Helper Functions
// ============================================================================

/// Get all libraries from Navidrome
fn get_all_libraries() -> Result<Vec<Library>, String> {
    let response: Json<LibraryGetAllLibrariesResponse> = unsafe {
        library_getalllibraries(Json(serde_json::json!({})))
            .map_err(|e| format!("Failed to call library_getalllibraries: {:?}", e))?
    };

    if let Some(err) = response.0.error {
        return Err(err);
    }

    Ok(response.0.result.unwrap_or_default())
}

/// Schedule a recurring task
fn schedule_recurring(cron: &str, payload: &str, id: &str) -> Result<String, String> {
    let request = SchedulerScheduleRecurringRequest {
        cron_expression: cron.to_string(),
        payload: payload.to_string(),
        schedule_id: id.to_string(),
    };

    let response: Json<SchedulerScheduleRecurringResponse> = unsafe {
        scheduler_schedulerecurring(Json(request))
            .map_err(|e| format!("Failed to schedule task: {:?}", e))?
    };

    if let Some(err) = response.0.error {
        return Err(err);
    }

    Ok(response.0.new_schedule_id.unwrap_or_default())
}

/// Format bytes into human-readable size
fn format_size(bytes: i64) -> String {
    const KB: i64 = 1024;
    const MB: i64 = KB * 1024;
    const GB: i64 = MB * 1024;
    const TB: i64 = GB * 1024;

    if bytes >= TB {
        format!("{:.2} TB", bytes as f64 / TB as f64)
    } else if bytes >= GB {
        format!("{:.2} GB", bytes as f64 / GB as f64)
    } else if bytes >= MB {
        format!("{:.2} MB", bytes as f64 / MB as f64)
    } else if bytes >= KB {
        format!("{:.2} KB", bytes as f64 / KB as f64)
    } else {
        format!("{} bytes", bytes)
    }
}

/// Format duration in seconds to human-readable format
fn format_duration(seconds: f64) -> String {
    let total_seconds = seconds as i64;
    let hours = total_seconds / 3600;
    let minutes = (total_seconds % 3600) / 60;

    if hours > 0 {
        format!("{}h {}m", hours, minutes)
    } else {
        format!("{}m", minutes)
    }
}

/// Find the largest file in a directory (non-recursive)
fn find_largest_file(mount_point: &str) -> Option<(String, u64)> {
    let entries = match fs::read_dir(mount_point) {
        Ok(entries) => entries,
        Err(e) => {
            warn!("Failed to read directory {}: {}", mount_point, e);
            return None;
        }
    };

    let mut largest: Option<(String, u64)> = None;

    for entry in entries.flatten() {
        let path = entry.path();
        
        // Only consider files, not directories
        if !path.is_file() {
            continue;
        }

        let metadata = match entry.metadata() {
            Ok(m) => m,
            Err(_) => continue,
        };

        let size = metadata.len();
        let name = entry.file_name().to_string_lossy().to_string();

        match &largest {
            None => largest = Some((name, size)),
            Some((_, current_size)) if size > *current_size => {
                largest = Some((name, size));
            }
            _ => {}
        }
    }

    largest
}

/// Inspect and log all library details
fn inspect_libraries() {
    info!("=== Library Inspection Started ===");

    let libraries = match get_all_libraries() {
        Ok(libs) => libs,
        Err(e) => {
            error!("Failed to get libraries: {}", e);
            return;
        }
    };

    if libraries.is_empty() {
        info!("No libraries configured");
        return;
    }

    info!("Found {} libraries", libraries.len());

    for lib in &libraries {
        info!("----------------------------------------");
        info!("Library: {} (ID: {})", lib.name, lib.id);
        info!("  Songs:    {} tracks", lib.total_songs);
        info!("  Albums:   {}", lib.total_albums);
        info!("  Artists:  {}", lib.total_artists);
        info!("  Size:     {}", format_size(lib.total_size));
        info!("  Duration: {}", format_duration(lib.total_duration));

        // If we have filesystem access, find the largest file
        if let Some(mount_point) = &lib.mount_point {
            info!("  Mount:    {}", mount_point);

            match find_largest_file(mount_point) {
                Some((name, size)) => {
                    info!(
                        "  Largest file in root: {} ({})",
                        name,
                        format_size(size as i64)
                    );
                }
                None => {
                    info!("  Largest file in root: (no files found)");
                }
            }
        } else {
            info!("  (Filesystem access not enabled)");
        }
    }

    info!("=== Library Inspection Complete ===");
}

// ============================================================================
// Plugin Exports
// ============================================================================

/// Called when the plugin is initialized. Schedules the recurring inspection task.
#[plugin_fn]
pub fn nd_on_init() -> FnResult<Json<InitOutput>> {
    info!("Library Inspector plugin initializing...");

    // Get cron expression from config, default to every minute
    let cron = config::get("cron")
        .ok()
        .flatten()
        .unwrap_or_else(|| "@every 1m".to_string());

    info!("Scheduling library inspection with cron: {}", cron);

    // Schedule the recurring task
    match schedule_recurring(&cron, "inspect", "library-inspect") {
        Ok(schedule_id) => {
            info!("Scheduled inspection task with ID: {}", schedule_id);
        }
        Err(e) => {
            let error_msg = format!("Failed to schedule inspection: {}", e);
            error!("{}", error_msg);
            return Ok(Json(InitOutput {
                error: Some(error_msg),
            }));
        }
    }

    // Run an initial inspection
    inspect_libraries();

    info!("Library Inspector plugin initialized successfully");
    Ok(Json(InitOutput::default()))
}

/// Called when a scheduled task fires.
#[plugin_fn]
pub fn nd_scheduler_callback(Json(input): Json<SchedulerCallbackInput>) -> FnResult<()> {
    info!(
        "Scheduler callback fired: schedule_id={}, payload={}, recurring={}",
        input.schedule_id, input.payload, input.is_recurring
    );

    if input.payload == "inspect" {
        inspect_libraries();
    }

    Ok(())
}
