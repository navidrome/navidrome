# Now Playing Logger Plugin (Python)

A Python example plugin that demonstrates the **Scheduler** and **SubsonicAPI** host services by periodically logging what is currently playing in Navidrome.

## Features

- Uses `scheduler_schedulerecurring` host function to set up a recurring task
- Uses `subsonicapi_call` host function to query the `getNowPlaying` API
- Configurable cron expression and user via plugin config
- Demonstrates Python host function imports using `@extism.import_fn`

## Prerequisites

- [extism-py](https://github.com/extism/python-pdk) - Python PDK compiler
  ```bash
  curl -Ls https://raw.githubusercontent.com/extism/python-pdk/main/install.sh | bash
  ```

> **Note:** `extism-py` requires [Binaryen](https://github.com/WebAssembly/binaryen/) (`wasm-merge`, `wasm-opt`) to be installed.

## Building

From the `plugins/examples` directory:

```bash
make nowplaying-py.ndp
```

Or directly:

```bash
extism-py plugin/__init__.py -o plugin.wasm
zip -j nowplaying-py.ndp manifest.json plugin.wasm
```

## Installation

1. Copy `nowplaying-py.ndp` to your Navidrome plugins folder

2. Enable plugins in `navidrome.toml`:
   ```toml
   [Plugins]
   Enabled = true
   Folder = "/path/to/plugins"
   ```

3. Configure the plugin in the UI (Settings → Plugins → nowplaying-py)

## Configuration

| Key    | Description                         | Default       |
|--------|-------------------------------------|---------------|
| `cron` | Cron expression for check frequency | `*/1 * * * *` |
| `user` | Navidrome user for SubsonicAPI      | `admin`       |

## Testing

Test the manifest:

```bash
extism call nowplaying-py.wasm nd_manifest --wasi
```

## Output

When running, the plugin logs messages like:

```
🎵 john is playing: Pink Floyd - Comfortably Numb (The Wall)
🎵 jane is playing: Radiohead - Paranoid Android (OK Computer)
```

Or when no one is playing:

```
🎵 No users currently playing music
```

## How It Works

1. **Initialization (`nd_on_init`)**: Reads the cron expression from config and schedules a recurring task using the Scheduler host service.

2. **Callback (`nd_scheduler_callback`)**: When the scheduled task fires, calls the SubsonicAPI `getNowPlaying` endpoint and logs the results.

## Host Function Usage (Python)

This plugin demonstrates how to call Navidrome host functions from Python:

```python
import extism
import json

# Import the host function
@extism.import_fn("extism:host/user", "subsonicapi_call")
def _subsonicapi_call(offset: int) -> int:
    """Raw host function - returns memory offset."""
    ...

# Wrapper for JSON marshalling
def subsonicapi_call(uri: str) -> dict:
    request = {"uri": uri}
    request_bytes = json.dumps(request).encode('utf-8')
    request_mem = extism.memory.alloc(request_bytes)
    response_offset = _subsonicapi_call(request_mem.offset)
    response_mem = extism.memory.find(response_offset)
    response = json.loads(extism.memory.string(response_mem))
    
    if response.get("error"):
        raise Exception(response["error"])
    
    return json.loads(response.get("responseJSON", "{}"))
```