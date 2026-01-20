// Package host provides host services that can be called by plugins via Extism host functions.
//
// Host services allow plugins to access Navidrome functionality like the Subsonic API,
// scheduler, and other internal services. Services are defined as Go interfaces with
// special annotations that enable automatic code generation of Extism host function wrappers.
//
// # Annotation Format
//
// Host services use Go doc comment annotations to mark interfaces and methods for code generation:
//
//	// MyService provides some functionality.
//	//nd:hostservice name=MyService permission=myservice
//	type MyService interface {
//	    // DoSomething performs an action.
//	    //nd:hostfunc
//	    DoSomething(ctx context.Context, input string) (output string, err error)
//	}
//
// Service-level annotations:
//   - //nd:hostservice - Marks an interface as a host service
//   - name=<ServiceName> - Service identifier used in generated code
//   - permission=<key> - Manifest permission key (e.g., "subsonicapi", "scheduler")
//
// Method-level annotations:
//   - //nd:hostfunc - Marks a method for host function wrapper generation
//   - name=<CustomName> - Optional: override the export name
//
// # Generated Code
//
// The ndpgen tool reads annotated interfaces and generates Extism host function wrappers
// that handle:
//   - JSON serialization/deserialization of request/response types
//   - Memory operations (ReadBytes, WriteBytes, Alloc)
//   - Error handling and propagation
//   - Service registration functions
//
// Generated files follow the pattern <servicename>_gen.go and include a header comment
// indicating they should not be edited manually.
package host
