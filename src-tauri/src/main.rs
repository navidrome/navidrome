// src-tauri/src/main.rs
#![cfg_attr(
    all(not(debug_assertions), target_os = "macos"),
    windows_subsystem = "macos"
)]

#[cfg(target_os = "macos")]
mod macos_media;

use tauri::{AppHandle, Manager, Runtime, SystemTray, CustomMenuItem, SystemTrayMenu, SystemTrayEvent, Menu};
use std::sync::Mutex;

#[cfg(target_os = "macos")]
use macos_media::MacOSMediaControl;

#[cfg(target_os = "macos")]
struct MediaState {
    controller: MacOSMediaControl,
}

#[cfg(not(target_os = "macos"))]
struct MediaState {}

// Commands callable from JavaScript
#[tauri::command]
fn update_media_info(
    #[cfg(target_os = "macos")]
    state: tauri::State<Mutex<MediaState>>,
    title: String,
    artist: String,
    album: String,
    duration: f64,
    elapsed: f64,
) {
    #[cfg(target_os = "macos")]
    {
        let media = state.lock().unwrap();
        media.controller.update_now_playing(&title, &artist, &album, duration, elapsed, None);
    }
    
    #[cfg(not(target_os = "macos"))]
    {
        println!("Media info update not supported on this platform");
    }
}

#[tauri::command]
fn update_playback_state(
    #[cfg(target_os = "macos")]
    state: tauri::State<Mutex<MediaState>>,
    is_playing: bool
) {
    #[cfg(target_os = "macos")]
    {
        let media = state.lock().unwrap();
        media.controller.set_playback_state(is_playing);
    }
    
    #[cfg(not(target_os = "macos"))]
    {
        println!("Playback state update not supported on this platform");
    }
}

#[tauri::command]
fn clear_media_info(
    #[cfg(target_os = "macos")]
    state: tauri::State<Mutex<MediaState>>
) {
    #[cfg(target_os = "macos")]
    {
        let media = state.lock().unwrap();
        media.controller.clear_now_playing();
    }
    
    #[cfg(not(target_os = "macos"))]
    {
        println!("Clear media info not supported on this platform");
    }
}

fn main() {
    let play_pause = CustomMenuItem::new("play_pause".to_string(), "Play / Pause");
    let next_track = CustomMenuItem::new("next_track".to_string(), "Next Track");
    let prev_track = CustomMenuItem::new("prev_track".to_string(), "Previous Track");
    let quit = CustomMenuItem::new("quit".to_string(), "Quit");
    
    let tray_menu = SystemTrayMenu::new()
        .add_item(play_pause)
        .add_item(prev_track)
        .add_item(next_track)
        .add_native_item(tauri::menu::SystemTrayMenuItem::Separator)
        .add_item(quit);

    let mut builder = tauri::Builder::default()
        .menu(Menu::new())
        .system_tray(SystemTray::new().with_menu(tray_menu))
        .on_system_tray_event(|app, event| {
            match event {
                SystemTrayEvent::MenuItemClick { id, .. } => {
                    match id.as_str() {
                        "play_pause" => { 
                            let _ = app.emit_all("media-key-event", "play-pause");
                        }
                        "next_track" => { 
                            let _ = app.emit_all("media-key-event", "next-track");
                        }
                        "prev_track" => { 
                            let _ = app.emit_all("media-key-event", "prev-track");
                        }
                        "quit" => { 
                            std::process::exit(0);
                        }
                        _ => {}
                    }
                }
                _ => {}
            }
        })
        .invoke_handler(tauri::generate_handler![
            update_media_info,
            update_playback_state,
            clear_media_info
        ]);

    #[cfg(target_os = "macos")]
    {
        let media_state = Mutex::new(MediaState {
            controller: MacOSMediaControl::new(),
        });
        builder = builder.manage(media_state);
    }

    #[cfg(not(target_os = "macos"))]
    {
        let media_state = Mutex::new(MediaState {});
        builder = builder.manage(media_state);
    }

    builder
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
