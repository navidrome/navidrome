// src-tauri/src/macos_media.rs
#[cfg(target_os = "macos")]
use cocoa::base::{id, nil};
use objc::runtime::{Class, Object};
use objc::{msg_send, sel, sel_impl};
use objc_foundation::{NSString};
use objc_id::Id;

#[cfg(target_os = "macos")]
pub struct MacOSMediaControl {
    now_playing_info_center: Id<Object>,
}

#[cfg(target_os = "macos")]
impl MacOSMediaControl {
    pub fn new() -> Self {
        unsafe {
            let class = Class::get("MPNowPlayingInfoCenter").expect("MPNowPlayingInfoCenter class not found");
            let now_playing_info_center: *mut Object = msg_send![class, defaultCenter];
            
            MacOSMediaControl {
                now_playing_info_center: Id::from_retained_ptr(now_playing_info_center),
            }
        }
    }
    
    pub fn update_now_playing(&self, title: &str, artist: &str, album: &str, duration: f64, elapsed: f64, _artwork_url: Option<&str>) {
        unsafe {
            let dict = NSMutableDictionary();
            
            // Title
            let title_key = NSString::from_str("MPMediaItemPropertyTitle");
            let title_val = NSString::from_str(title);
            dict.setObject_forKey_(title_val, title_key);
            
            // Artist
            let artist_key = NSString::from_str("MPMediaItemPropertyArtist");
            let artist_val = NSString::from_str(artist);
            dict.setObject_forKey_(artist_val, artist_key);
            
            // Album
            if !album.is_empty() {
                let album_key = NSString::from_str("MPMediaItemPropertyAlbumTitle");
                let album_val = NSString::from_str(album);
                dict.setObject_forKey_(album_val, album_key);
            }
            
            // Duration
            let duration_key = NSString::from_str("MPMediaItemPropertyPlaybackDuration");
            let duration_val: id = msg_send![class!(NSNumber), numberWithDouble: duration];
            dict.setObject_forKey_(duration_val, duration_key);
            
            // Elapsed time
            let elapsed_key = NSString::from_str("MPNowPlayingInfoPropertyElapsedPlaybackTime");
            let elapsed_val: id = msg_send![class!(NSNumber), numberWithDouble: elapsed];
            dict.setObject_forKey_(elapsed_val, elapsed_key);
            
            // Playback rate (1.0 for playing, 0.0 for paused)
            let rate_key = NSString::from_str("MPNowPlayingInfoPropertyPlaybackRate");
            let rate_val: id = msg_send![class!(NSNumber), numberWithDouble: 1.0];
            dict.setObject_forKey_(rate_val, rate_key);
            
            let _: () = msg_send![*self.now_playing_info_center, setNowPlayingInfo: dict];
        }
    }
    
    pub fn set_playback_state(&self, is_playing: bool) {
        unsafe {
            let current_info: id = msg_send![*self.now_playing_info_center, nowPlayingInfo];
            if current_info != nil {
                let mutable_info: id = msg_send![current_info, mutableCopy];
                let rate_key = NSString::from_str("MPNowPlayingInfoPropertyPlaybackRate");
                let rate: f64 = if is_playing { 1.0 } else { 0.0 };
                let rate_val: id = msg_send![class!(NSNumber), numberWithDouble: rate];
                let _: () = msg_send![mutable_info, setObject:rate_val forKey:rate_key];
                let _: () = msg_send![*self.now_playing_info_center, setNowPlayingInfo: mutable_info];
            }
        }
    }
    
    pub fn clear_now_playing(&self) {
        unsafe {
            let _: () = msg_send![*self.now_playing_info_center, setNowPlayingInfo: nil];
        }
    }
}

#[cfg(target_os = "macos")]
unsafe fn NSMutableDictionary() -> id {
    msg_send![class!(NSMutableDictionary), new]
}

#[cfg(target_os = "macos")]
trait NSDictionaryExt {
    unsafe fn setObject_forKey_(self, object: id, key: id);
}

#[cfg(target_os = "macos")]
impl NSDictionaryExt for id {
    unsafe fn setObject_forKey_(self, object: id, key: id) {
        msg_send![self, setObject:object forKey:key]
    }
}

#[cfg(target_os = "macos")]
trait NSStringExt {
    fn from_str(s: &str) -> id;
}

#[cfg(target_os = "macos")]
impl NSStringExt for NSString {
    fn from_str(s: &str) -> id {
        unsafe {
            let cls = class!(NSString);
            let bytes = s.as_ptr() as *const std::os::raw::c_void;
            let obj: id = msg_send![cls, alloc];
            let obj: id = msg_send![obj, initWithBytes:bytes
                                            length:s.len()
                                          encoding:4]; // UTF8 encoding
            obj
        }
    }
}
