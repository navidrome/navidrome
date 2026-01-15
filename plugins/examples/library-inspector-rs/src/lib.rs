//! Library Inspector Plugin for Navidrome
//!
//! This plugin demonstrates how to use the nd-pdk crate for accessing Navidrome
//! host services and implementing capabilities in Rust. It periodically logs details
//! about all music libraries and finds the largest file in the root of each library.
//!
//! ## Configuration
//!
//! Set the `cron` config key to customize the schedule (default: "@every 1m"):
//! ```toml
//! [PluginConfig.library-inspector]
//! cron = "@every 5m"
//! ```

use extism_pdk::*;
use nd_pdk::host::{library, scheduler};
use nd_pdk::lifecycle::{Error as LifecycleError, InitProvider};
use nd_pdk::scheduler::{CallbackProvider, Error as SchedulerError, SchedulerCallbackRequest};
use std::fs;

// Register capabilities using PDK macros
nd_pdk::register_lifecycle_init!(LibraryInspector);
nd_pdk::register_scheduler_callback!(LibraryInspector);

// ============================================================================
// Plugin Implementation
// ============================================================================

/// The library inspector plugin type.
#[derive(Default)]
struct LibraryInspector;

impl InitProvider for LibraryInspector {
    fn on_init(&self) -> Result<(), LifecycleError> {
        info!("Library Inspector plugin initializing...");

        // Get cron expression from config, default to every minute
        let cron = config::get("cron")
            .ok()
            .flatten()
            .unwrap_or_else(|| "@every 1m".to_string());

        info!("Scheduling library inspection with cron: {}", cron);

        // Schedule the recurring task using nd-pdk host scheduler
        match scheduler::schedule_recurring(&cron, "inspect", "library-inspect") {
            Ok(schedule_id) => {
                info!("Scheduled inspection task with ID: {}", schedule_id);
            }
            Err(e) => {
                let error_msg = format!("Failed to schedule inspection: {}", e);
                error!("{}", error_msg);
                return Err(LifecycleError::new(error_msg));
            }
        }

        // Run an initial inspection
        inspect_libraries();

        info!("Library Inspector plugin initialized successfully");
        Ok(())
    }
}

impl CallbackProvider for LibraryInspector {
    fn on_callback(&self, req: SchedulerCallbackRequest) -> Result<(), SchedulerError> {
        info!(
            "Scheduler callback fired: schedule_id={}, payload={}, recurring={}",
            req.schedule_id, req.payload, req.is_recurring
        );

        if req.payload == "inspect" {
            inspect_libraries();
        }

        Ok(())
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

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

    let libraries = match library::get_all_libraries() {
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
        if !lib.mount_point.is_empty() {
            info!("  Mount:    {}", lib.mount_point);

            match find_largest_file(&lib.mount_point) {
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
