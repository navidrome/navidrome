# hostgen

A code generator for Navidrome's plugin host functions. It reads Go interface definitions with special annotations and generates Extism host function wrappers.

## Usage

```bash
hostgen -input <dir> -output <dir> -package <name> [-v] [-dry-run] [-host-only] [-plugin-only]
```

### Flags

| Flag           | Description                                                    | Default  |
|----------------|----------------------------------------------------------------|----------|
| `-input`       | Directory containing Go source files with annotated interfaces | Required |
| `-output`      | Directory where generated files will be written                | Required |
| `-package`     | Package name for generated files                               | Required |
| `-v`           | Verbose output                                                 | `false`  |
| `-dry-run`     | Parse and validate without writing files                       | `false`  |
| `-host-only`   | Generate only host-side wrapper code                           | `false`  |
| `-plugin-only` | Generate only plugin/client-side wrapper code                  | `false`  |
| `-go`          | Generate Go client wrappers                                    | `true`*  |
| `-python`      | Generate Python client wrappers                                | `false`  |
| `-rust`        | Generate Rust client wrappers                                  | `false`  |

\* `-go` is enabled by default when neither `-python` nor `-rust` is specified. Use combinations like `-go -python -rust` to generate multiple languages.

By default, both host and Go plugin code are generated. Use `-host-only` or `-plugin-only` to generate only one type. Use `-python` to generate Python wrappers and `-rust` to generate Rust wrappers.

### Example

```bash
go run ./plugins/cmd/hostgen \
  -input ./plugins/host \
  -output ./plugins/host \
  -package host
```

Or via `go generate` (recommended):

```go
//go:generate go run ../cmd/hostgen -input . -output . -package host
package host
```

## Annotations

### `//nd:hostservice`

Marks an interface as a host service that will have wrappers generated.

```go
//nd:hostservice name=<ServiceName> permission=<permission>
type MyService interface { ... }
```

| Parameter    | Description                                                     | Required |
|--------------|-----------------------------------------------------------------|----------|
| `name`       | Service name used in generated type names and function prefixes | Yes      |
| `permission` | Permission required by plugins to use this service              | Yes      |

### `//nd:hostfunc`

Marks a method within a host service interface for export to plugins.

```go
//nd:hostfunc [name=<export_name>]
MethodName(ctx context.Context, ...) (result Type, err error)
```

| Parameter | Description                                                             | Required |
|-----------|-------------------------------------------------------------------------|----------|
| `name`    | Custom export name (default: `<servicename>_<methodname>` in lowercase) | No       |

## Input Format

Host service interfaces must follow these conventions:

1. **First parameter must be `context.Context`** - Required for all methods
2. **Last return value should be `error`** - For proper error handling
3. **Annotations must be on consecutive lines** - No blank comment lines between doc and annotation

### Example Interface

```go
package host

import "context"

// SubsonicAPIService provides access to Navidrome's Subsonic API.
// This documentation becomes part of the generated code.
//nd:hostservice name=SubsonicAPI permission=subsonicapi
type SubsonicAPIService interface {
    // Call executes a Subsonic API request and returns the response.
    //nd:hostfunc
    Call(ctx context.Context, uri string) (response string, err error)
}
```

## Generated Output

For each annotated interface, hostgen generates:

### Request/Response Types

```go
// SubsonicAPICallRequest is the request type for SubsonicAPI.Call.
type SubsonicAPICallRequest struct {
    Uri string `json:"uri"`
}

// SubsonicAPICallResponse is the response type for SubsonicAPI.Call.
type SubsonicAPICallResponse struct {
    Response string `json:"response,omitempty"`
    Error    string `json:"error,omitempty"`
}
```

### Registration Function

```go
// RegisterSubsonicAPIHostFunctions registers SubsonicAPI service host functions.
func RegisterSubsonicAPIHostFunctions(service SubsonicAPIService) []extism.HostFunction {
    return []extism.HostFunction{
        newSubsonicAPICallHostFunction(service),
    }
}
```

### Host Function Wrappers

Each method gets a wrapper that:
1. Reads JSON request from plugin memory
2. Unmarshals to the request type
3. Calls the service method
4. Marshals the response
5. Writes JSON response to plugin memory

## Supported Types

hostgen supports these Go types in method signatures:

| Type                          | JSON Representation                      |
|-------------------------------|------------------------------------------|
| `string`, `int`, `bool`, etc. | Native JSON types                        |
| `[]T` (slices)                | JSON arrays                              |
| `map[K]V` (maps)              | JSON objects                             |
| `*T` (pointers)               | Nullable fields                          |
| `interface{}` / `any`         | Converts to `any`                        |
| Custom structs                | JSON objects (must be JSON-serializable) |

### Multiple Return Values

Methods can return multiple values (plus error):

```go
//nd:hostfunc
Search(ctx context.Context, query string) (results []string, total int, hasMore bool, err error)
```

Generates:

```go
type ServiceSearchResponse struct {
    Results []string `json:"results,omitempty"`
    Total   int      `json:"total,omitempty"`
    HasMore bool     `json:"hasMore,omitempty"`
    Error   string   `json:"error,omitempty"`
}
```

## Output Files

### Host Code (Navidrome-side)

Generated files are named `<servicename>_gen.go` (lowercase) and placed in the output directory. Each file includes:

- `// Code generated by hostgen. DO NOT EDIT.` header
- Required imports (`context`, `encoding/json`, `extism`)
- Request/response struct types
- Registration function
- Host function wrappers
- Helper functions (`writeResponse`, `writeErrorResponse`)

### Plugin/Client Code (TinyGo WASM)

Generated files are named `nd_host_<servicename>.go` (lowercase) and placed in the `go/` subdirectory of the output directory. These files are intended for use in Navidrome plugins built with TinyGo. Each file includes:

- `// Code generated by hostgen. DO NOT EDIT.` header
- Required imports (`encoding/json`, `errors`, `github.com/extism/go-pdk`)
- `//go:wasmimport` declarations for each host function
- Response struct types
- Wrapper functions that handle memory allocation and JSON parsing

### Example Output Structure

```
output/
├── subsonicapi_gen.go      # Host-side code (for Navidrome)
├── go/
│   └── nd_host_subsonicapi.go  # Plugin-side code (for TinyGo plugins)
├── python/
│   └── nd_host_subsonicapi.py  # Plugin-side code (for Python plugins)
└── rust/
    └── nd_host_subsonicapi.rs  # Plugin-side code (for Rust plugins)
```

### Python Client Code (extism-py WASM)

Generated files are named `nd_host_<servicename>.py` (lowercase) and placed in the `python/` subdirectory of the output directory. These files are intended for use in Navidrome plugins built with extism-py. Each file includes:

- `# Code generated by hostgen. DO NOT EDIT.` header
- Required imports (`dataclasses`, `typing`, `extism`, `json`)
- `HostFunctionError` exception class for error handling
- `@extism.import_fn` declarations for raw host functions
- `@dataclass` types for methods with multiple return values
- Wrapper functions with type hints, docstrings, and snake_case names

#### Python Type Mapping

| Go Type                 | Python Type |
|-------------------------|-------------|
| `string`                | `str`       |
| `int`, `int32`, `int64` | `int`       |
| `float32`, `float64`    | `float`     |
| `bool`                  | `bool`      |
| `[]byte`                | `bytes`     |
| Unknown                 | `Any`       |

#### Python Function Naming

Functions follow PEP 8 snake_case convention:

| Go Method                     | Python Function                  |
|-------------------------------|----------------------------------|
| `SubsonicAPI.Call`            | `subsonicapi_call()`             |
| `Scheduler.ScheduleRecurring` | `scheduler_schedule_recurring()` |
| `Cache.GetString`             | `cache_get_string()`             |

#### Multi-Value Returns

Methods with multiple return values use dataclasses:

```python
@dataclass
class CacheGetStringResult:
    value: str
    exists: bool

def cache_get_string(key: str) -> CacheGetStringResult:
    ...
```

#### Python Plugin Usage

> **Important:** Due to a limitation in extism-py, you cannot directly import the generated Python wrappers.
> The `@extism.import_fn` decorators are only detected when defined in the plugin's main `__init__.py` file.
> Generated Python files serve as **reference/template code** - copy the needed functions into your plugin.

Example of copying the generated wrapper into your plugin's `__init__.py`:

```python
import extism
import json

# Copy host function declarations from generated files into your __init__.py
@extism.import_fn("extism:host/user", "subsonicapi_call")
def _host_subsonicapi_call(input_ptr: extism.JsonI64) -> extism.JsonI64:
    pass

def subsonicapi_call(endpoint: str) -> str:
    """Call the SubsonicAPI with the given endpoint."""
    result = _host_subsonicapi_call(endpoint)
    return result

# Now use it in your plugin
@extism.plugin_fn
def my_plugin_function():
    try:
        response = subsonicapi_call("getAlbumList2?type=random&size=10")
        data = json.loads(response)
    except Exception as e:
        extism.log(extism.LogLevel.Error, f"API error: {e}")
```

### Rust Client Code (extism-pdk WASM)

Generated files are named `nd_host_<servicename>.rs` (lowercase) and placed in the `rust/` subdirectory of the output directory. These files are intended for use in Navidrome plugins built with `extism-pdk`. Each file includes:

- `// Code generated by hostgen. DO NOT EDIT.` header
- Required imports (`extism_pdk::*`, `serde`)
- Request/response struct types with `#[derive(Serialize, Deserialize)]`
- `#[host_fn]` extern blocks for raw host function imports
- Public wrapper functions with `Result<T, Error>` returns and snake_case names

#### Rust Type Mapping

| Go Type                       | Rust Type                     | Notes                        |
|-------------------------------|-------------------------------|------------------------------|
| `string`                      | `String` / `&str`             | `&str` for params            |
| `int`, `int32`                | `i32`                         |                              |
| `int64`                       | `i64`                         |                              |
| `float32`                     | `f32`                         |                              |
| `float64`                     | `f64`                         |                              |
| `bool`                        | `bool`                        |                              |
| `[]byte`                      | `Vec<u8>`                     |                              |
| `[]T`                         | `Vec<T>`                      |                              |
| `map[K]V`                     | `HashMap<K, V>`               | From `std::collections`      |
| `*T`                          | `Option<T>`                   |                              |
| `interface{}` / `any`         | `serde_json::Value`           |                              |

#### Rust Function Naming

Functions follow Rust snake_case convention:

| Go Method                     | Rust Function                    |
|-------------------------------|----------------------------------|
| `SubsonicAPI.Call`            | `subsonicapi_call()`             |
| `Scheduler.ScheduleRecurring` | `scheduler_schedule_recurring()` |
| `Cache.GetString`             | `cache_get_string()`             |

#### Rust Plugin Usage

The generated Rust wrappers form a library crate (`nd-host`) that plugins can depend on. Add the dependency to your plugin's `Cargo.toml`:

```toml
[dependencies]
extism-pdk = "1.2"
nd-host = { path = "../../host/rust" }
```

Then import and use the services:

```rust
use nd_host::{cache, scheduler, artwork};

#[plugin_fn]
pub fn my_callback(input: String) -> FnResult<String> {
    // Use cache service
    cache::cache_set_string("key", "value", 3600)?;
    let value = cache::cache_get_string("key")?;

    // Schedule a task
    scheduler::scheduler_schedule_one_time(60, "payload", "task-id")?;

    // Get artwork URL
    let url = artwork::artwork_get_track_url("track-id", 300)?;

    Ok("done")
}
```

See [discord-rich-presence-rs](../examples/discord-rich-presence-rs/) for a complete example using all Rust host wrappers.

## Troubleshooting

### Annotations Not Detected

Ensure annotations are on consecutive lines with no blank `//` lines:

```go
// ✅ Correct
// Documentation for the service.
//nd:hostservice name=Test permission=test

// ❌ Wrong - blank comment line breaks detection
// Documentation for the service.
//
//nd:hostservice name=Test permission=test
```

### Methods Not Exported

Methods without `//nd:hostfunc` annotation are skipped. Ensure the annotation is directly above the method:

```go
// ✅ Correct
// Method documentation.
//nd:hostfunc
MyMethod(ctx context.Context) error

// ❌ Wrong - annotation not directly above method
//nd:hostfunc

MyMethod(ctx context.Context) error
```

### Generated Files Skipped

Files ending in `_gen.go` are automatically skipped during parsing to avoid processing previously generated code.

## Development

Run tests:

```bash
go test -v ./plugins/cmd/hostgen/...
```

The test suite includes:
- CLI integration tests
- Complex type handling (structs, slices, maps, pointers)
- Multiple return value scenarios
- Error cases and edge conditions
