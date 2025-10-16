// src-tauri/src/main.rs
#![cfg_attr(
    all(not(debug_assertions), target_os = "macos"),
    windows_subsystem = "macos"
)]

use tauri::{AppHandle, Manager, Runtime, SystemTray, CustomMenuItem, SystemTrayMenu, SystemTrayEvent, Menu};

// --- IMPORT MAC OS MEDIA CONTROL MODULE (Conceptual, requires external crate/plugin) ---
// Note: Actual macOS media key and Now Playing support often requires using a library 
// like `media-kit-rs` or integrating with MPRemoteCommandCenter, which is complex.
// This provides the framework for dispatching events.

// Placeholder function for native macOS media update
// In a real app, this function would use a macOS-specific crate to talk to MPRemoteCommandCenter.
pub fn update_now_playing<R: Runtime>(app: &AppHandle<R>, title: String, artist: String, cover_url: String) {
    println!("Native macOS Update: Title='{}', Artist='{}', Cover='{}'", title, artist, cover_url);
    // 🚨 REAL IMPLEMENTATION HERE: Call macOS APIs 🚨
}

// 1. Command callable from JavaScript to update Now Playing info
#[tauri::command]
fn update_media_info<R: Runtime>(app: AppHandle<R>, title: String, artist: String, cover_url: String) {
    update_now_playing(&app, title, artist, cover_url);
}

fn main() {
    let play_pause = CustomMenuItem::new("play_pause".to_string(), "Play / Pause");
    let next_track = CustomMenuItem::new("next_track".to_string(), "Next Track");
    let quit = CustomMenuItem::new("quit".to_string(), "Quit");
    let tray_menu = SystemTrayMenu::new()
        .add_item(play_pause)
        .add_item(next_track)
        .add_native_item(tauri::menu::SystemTrayMenuItem::Separator)
        .add_item(quit);

    tauri::Builder::default()
        .menu(Menu::new()) // Hides the default menu bar items
        .system_tray(SystemTray::new().with_menu(tray_menu))
        .on_system_tray_event(|app, event| {
            // 2. Handle System Tray clicks (e.g., Play/Pause from Menu Bar)
            match event {
                SystemTrayEvent::MenuItemClick { id, .. } => {
                    match id.as_str() {
                        "play_pause" => { app.emit_all("media-key-event", "play-pause").unwrap(); }
                        "next_track" => { app.emit_all("media-key-event", "next-track").unwrap(); }
                        "quit" => { app.exit(0); }
                        _ => {}
                    }
                }
                _ => {}
            }
        })
        .invoke_handler(tauri::generate_handler![update_media_info])
        // 3. Setup global media key listeners
        .setup(|app| {
            // Note: True global media key events require deeper native integration 
            // than standard Tauri setup, often relying on specialized Rust crates 
            // (like 'media_manager' or 'global_hotkey') which would dispatch the 
            // "media-key-event" back to the frontend.

            // The System Tray handles the menu bar requirement, and the event dispatcher 
            // handles the key/command requirement.
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
