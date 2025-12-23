# hostgen

A code generator for Navidrome's plugin host functions. It reads Go interface definitions with special annotations and generates Extism host function wrappers.

## Usage

```bash
hostgen -input <dir> -output <dir> -package <name> [-v] [-dry-run]
```

### Flags

| Flag       | Description                                                    | Default  |
|------------|----------------------------------------------------------------|----------|
| `-input`   | Directory containing Go source files with annotated interfaces | Required |
| `-output`  | Directory where generated files will be written                | Required |
| `-package` | Package name for generated files                               | Required |
| `-v`       | Verbose output                                                 | `false`  |
| `-dry-run` | Parse and validate without writing files                       | `false`  |

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

Generated files are named `<servicename>_gen.go` (lowercase). Each file includes:

- `// Code generated by hostgen. DO NOT EDIT.` header
- Required imports (`context`, `encoding/json`, `extism`)
- Request/response struct types
- Registration function
- Host function wrappers
- Helper functions (`writeResponse`, `writeErrorResponse`)

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
