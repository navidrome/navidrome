// src-tauri/src/main.rs
#![cfg_attr(
    all(not(debug_assertions), target_os = "macos"),
    windows_subsystem = "macos"
)]

use tauri::{AppHandle, Manager, Runtime, SystemTray, CustomMenuItem, SystemTrayMenu, SystemTrayEvent, Menu};
use tauri::GlobalShortcutManager;

// --- MAC OS MEDIA CONTROL MODULE (Conceptual Placeholder) ---
// This function receives track info from JavaScript and calls the native macOS API.
pub fn update_now_playing<R: Runtime>(app: &AppHandle<R>, title: String, artist: String, cover_url: String) {
    // 🚨 REAL IMPLEMENTATION HERE: This is where FFI calls or a specialized crate 
    // would update the macOS MPRemoteCommandCenter (Control Center/Now Playing).
    println!("Native macOS Update Called: Title='{}', Artist='{}', Cover='{}'", title, artist, cover_url);
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
            // 2. Handle System Tray clicks (Menu Bar controls)
            match event {
                SystemTrayEvent::MenuItemClick { id, .. } => {
                    // Emit event back to the JavaScript frontend
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
        .setup(|app| {
            // 3. Setup global media key listeners
            let mut manager = app.global_shortcut_manager();
            let app_handle = app.handle();
            
            // Registering media keys. Note: Key names are system/library dependent.
            // On macOS, these typically map to the F-keys on external keyboards or system-wide media controls.
            // These lines enable the media key control functionality you requested.
            manager.register("PlayPause", move |_| {
                app_handle.emit_all("media-key-event", "play-pause").unwrap();
            }).ok();
            
            let app_handle = app.handle();
            manager.register("MediaNextTrack", move |_| {
                app_handle.emit_all("media-key-event", "next-track").unwrap();
            }).ok();
            
            let app_handle = app.handle();
            manager.register("MediaPreviousTrack", move |_| {
                app_handle.emit_all("media-key-event", "prev-track").unwrap();
            }).ok();

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
