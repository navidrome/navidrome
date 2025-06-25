# SubsonicAPI Demo Plugin

This example plugin demonstrates how to use the SubsonicAPI host service to access Navidrome's Subsonic API from within a plugin.

## What it does

The plugin performs the following operations during initialization:

1. **Ping the server**: Calls `/rest/ping` to check if the Subsonic API is responding
2. **Get license info**: Calls `/rest/getLicense` to retrieve server license information

## Key Features

- Shows how to request `subsonicapi` permission in the manifest
- Demonstrates making Subsonic API calls using the `subsonicapi.Call()` method
- Handles both successful responses and errors
- Uses proper lifecycle management with `OnInit`

## Usage

### Manifest Configuration

```json
{
  "permissions": {
    "subsonicapi": {
      "reason": "Demonstrate accessing Navidrome's Subsonic API from within plugins",
      "allowAdmins": true
    }
  }
}
```

### Plugin Implementation

```go
import "github.com/navidrome/navidrome/plugins/host/subsonicapi"

var subsonicService = subsonicapi.NewSubsonicAPIService()

// OnInit is called when the plugin is loaded
func (SubsonicAPIDemoPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
    // Make API calls
    response, err := subsonicService.Call(ctx, &subsonicapi.CallRequest{
        Url: "/rest/ping?u=admin",
    })
    // Handle response...
}
```

When running Navidrome with this plugin installed, it will automatically call the Subsonic API endpoints during the
server startup, and you can see the results in the logs:

```agsl
INFO[0000] 2022/01/01 00:00:00 SubsonicAPI Demo Plugin initializing...
DEBU[0000] API: New request /ping                        client=subsonicapi-demo username=admin version=1.16.1
DEBU[0000] API: Successful response                      endpoint=/ping status=OK
DEBU[0000] API: New request /getLicense                  client=subsonicapi-demo username=admin version=1.16.1
INFO[0000] 2022/01/01 00:00:00 SubsonicAPI ping response: {"subsonic-response":{"status":"ok","version":"1.16.1","type":"navidrome","serverVersion":"dev","openSubsonic":true}}
DEBU[0000] API: Successful response                      endpoint=/getLicense status=OK
DEBU[0000] Plugin initialized successfully               elapsed=41.9ms plugin=subsonicapi-demo
INFO[0000] 2022/01/01 00:00:00 SubsonicAPI license info: {"subsonic-response":{"status":"ok","version":"1.16.1","type":"navidrome","serverVersion":"dev","openSubsonic":true,"license":{"valid":true}}}
```

## Important Notes

1. **Authentication**: The plugin must provide valid authentication parameters in the URL:
   - **Required**: `u` (username) - The service validates this parameter is present
   - Example: `"/rest/ping?u=admin"`
2. **URL Format**: Only the path and query parameters from the URL are used - host, protocol, and method are ignored
3. **Automatic Parameters**: The service automatically adds:
   - `c`: Plugin name (client identifier)
   - `v`: Subsonic API version (1.16.1)
   - `f`: Response format (json)
4. **Internal Authentication**: The service sets up internal authentication using the `u` parameter
5. **Lifecycle**: This plugin uses `LifecycleManagement` with only the `OnInit` method

## Building

This plugin uses the `wasip1` build constraint and must be compiled for WebAssembly:

```bash
# Using the project's make target (recommended)
make plugin-examples

# Manual compilation (when using the proper toolchain)
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm plugin.go
```
