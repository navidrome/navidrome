# Now Playing Logger Plugin for Navidrome
#
# This plugin demonstrates the Scheduler and SubsonicAPI host services by
# periodically logging what is currently playing in Navidrome.
#
# Build with:
#   extism-py plugin/__init__.py -o nowplaying-py.wasm
#
# Configuration:
#   [PluginConfig.nowplaying-py]
#   cron = "*/1 * * * *"  # Every minute (default)
#   user = "admin"     # User to query getNowPlaying (default)

import extism
import json

# Schedule ID for our recurring task
SCHEDULE_ID = "nowplaying-check"


# =============================================================================
# Host Function Imports
# =============================================================================
# These are custom host functions provided by Navidrome.
# We import them using the extism:host/user namespace.


@extism.import_fn("extism:host/user", "scheduler_schedulerecurring")
def _scheduler_schedulerecurring(offset: int) -> int:
    """Raw host function - do not call directly."""
    ...


@extism.import_fn("extism:host/user", "subsonicapi_call")
def _subsonicapi_call(offset: int) -> int:
    """Raw host function - do not call directly."""
    ...


# =============================================================================
# Host Function Wrappers
# =============================================================================
# These wrappers handle JSON marshalling/unmarshalling and memory management.
# They were copied from plugins/host/python due to extism-py limitations.


def scheduler_schedule_recurring(cron_expression: str, payload: str, schedule_id: str) -> str:
    """Schedule a recurring task using a cron expression.
    
    Args:
        cron_expression: Cron format (e.g., "*/1 * * * *" for every minute)
        payload: Data to pass to the callback
        schedule_id: Unique identifier for the schedule
        
    Returns:
        The schedule ID (same as input or auto-generated)
    """
    request = {
        "cronExpression": cron_expression,
        "payload": payload,
        "scheduleId": schedule_id
    }
    request_bytes = json.dumps(request).encode('utf-8')
    request_mem = extism.memory.alloc(request_bytes)
    response_offset = _scheduler_schedulerecurring(request_mem.offset)
    response_mem = extism.memory.find(response_offset)
    response = json.loads(extism.memory.string(response_mem))
    
    if response.get("error"):
        raise Exception(response["error"])
    
    return response.get("newScheduleId", schedule_id)


def subsonicapi_call(uri: str) -> dict:
    """Call a Subsonic API endpoint.
    
    Args:
        uri: API path (e.g., "getNowPlaying")
        
    Returns:
        Parsed JSON response from the API
    """
    request = {"uri": uri}
    request_bytes = json.dumps(request).encode('utf-8')
    request_mem = extism.memory.alloc(request_bytes)
    response_offset = _subsonicapi_call(request_mem.offset)
    response_mem = extism.memory.find(response_offset)
    response = json.loads(extism.memory.string(response_mem))
    
    if response.get("error"):
        raise Exception(response["error"])
    
    # Parse the nested JSON response
    response_json = response.get("responseJson", "{}")
    return json.loads(response_json)


# =============================================================================
# Plugin Exports
# =============================================================================


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
    # No output - lifecycle callbacks don't return responses


@extism.plugin_fn
def nd_scheduler_callback():
    """Handle scheduler callback - check and log now playing tracks."""
    input_data = extism.input_json()
    schedule_id = input_data.get("scheduleId", "")
    
    # Only handle our schedule
    if schedule_id != SCHEDULE_ID:
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
        # No output - scheduler callbacks don't return responses
        
    except Exception as e:
        extism.log(extism.LogLevel.Error, f"Failed to get now playing: {e}")
        # Errors are logged but scheduler callbacks don't return responses
