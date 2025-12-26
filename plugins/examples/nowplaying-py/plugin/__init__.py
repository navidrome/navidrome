# Now Playing Logger Plugin for Navidrome
#
# This plugin demonstrates the Scheduler and SubsonicAPI host services by
# periodically logging what is currently playing in Navidrome.
#
# Build with:
#   extism-py plugin/__init__.py -o nowplaying-py.wasm
#
# Test manifest with:
#   extism call nowplaying-py.wasm nd_manifest --wasi
#
# Configuration:
#   [PluginConfig.nowplaying-py]
#   cron = "*/1 * * * *"  # Every minute (default)
#   user = "admin"     # User to query getNowPlaying (default)

import extism
import json

# Import generated host function wrappers
from nd_host_scheduler import scheduler_schedule_recurring, HostFunctionError
from nd_host_subsonicapi import subsonicapi_call as _subsonicapi_call_raw

# Schedule ID for our recurring task
SCHEDULE_ID = "nowplaying-check"


def subsonicapi_call(uri: str) -> dict:
    """Call a Subsonic API endpoint and parse the response.

    This is a convenience wrapper around the generated subsonicapi_call
    that parses the JSON response string into a dict.

    Args:
        uri: API path (e.g., "getNowPlaying")

    Returns:
        Parsed JSON response from the API
    """
    response_json = _subsonicapi_call_raw(uri)
    return json.loads(response_json) if response_json else {}


# =============================================================================
# Plugin Exports
# =============================================================================


@extism.plugin_fn
def nd_manifest():
    """Return the plugin manifest with metadata and permissions."""
    manifest = {
        "name": "Now Playing Logger (Python)",
        "author": "Navidrome",
        "version": "1.0.0",
        "description": "Periodically logs currently playing tracks - Python example demonstrating Scheduler and SubsonicAPI host services",
        "website": "https://github.com/navidrome/navidrome/tree/master/plugins/examples/nowplaying-py",
        "permissions": {
            "scheduler": {
                "reason": "Schedule periodic checks for now playing status"
            },
            "subsonicapi": {
                "reason": "Query the getNowPlaying API endpoint",
                "allowAdmins": True
            }
        }
    }
    extism.output_str(json.dumps(manifest))


@extism.plugin_fn
def nd_on_init():
    """Initialize the plugin by scheduling the recurring task."""
    # Read cron expression from config, default to every minute
    cron = extism.Config.get_str("cron")
    if not cron:
        cron = "*/1 * * * *"
    
    extism.log(extism.LogLevel.Info, f"Now Playing Logger initializing with cron: {cron}")
    
    try:
        schedule_id = scheduler_schedule_recurring(cron, "check", SCHEDULE_ID)
        extism.log(extism.LogLevel.Info, f"Scheduled recurring task with ID: {schedule_id}")
    except Exception as e:
        extism.log(extism.LogLevel.Error, f"Failed to schedule task: {e}")
        raise
    
    # Return empty success response
    extism.output_str(json.dumps({}))


@extism.plugin_fn
def nd_scheduler_callback():
    """Handle scheduler callback - check and log now playing tracks."""
    input_data = extism.input_json()
    schedule_id = input_data.get("schedule_id", "")
    
    # Only handle our schedule
    if schedule_id != SCHEDULE_ID:
        extism.output_str(json.dumps({}))
        return
    
    try:
        # Read user from config, default to admin
        user = extism.Config.get_str("user")
        if not user:
            user = "admin"
        
        # Call the getNowPlaying API
        response = subsonicapi_call(f"getNowPlaying?u={user}")
        
        # Extract the subsonic-response
        subsonic_response = response.get("subsonic-response", {})
        now_playing = subsonic_response.get("nowPlaying", {})
        entries = now_playing.get("entry", [])
        
        if not entries:
            extism.log(extism.LogLevel.Info, "🎵 No users currently playing music")
        else:
            # Handle both single entry and list of entries
            if isinstance(entries, dict):
                entries = [entries]
            
            for entry in entries:
                artist = entry.get("artist", "Unknown Artist")
                title = entry.get("title", "Unknown Title")
                album = entry.get("album", "Unknown Album")
                username = entry.get("username", "Unknown User")
                
                extism.log(
                    extism.LogLevel.Info,
                    f"🎵 {username} is playing: {artist} - {title} ({album})"
                )
        
        extism.output_str(json.dumps({}))
        
    except Exception as e:
        error_msg = str(e)
        extism.log(extism.LogLevel.Error, f"Failed to get now playing: {error_msg}")
        extism.output_str(json.dumps({"error": error_msg}))
