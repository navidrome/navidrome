package capabilities

// Lifecycle provides plugin lifecycle hooks.
// This capability allows plugins to perform initialization when loaded,
// such as establishing connections, starting background processes, or
// validating configuration.
//
// The OnInit function is called once when the plugin is loaded, and is NOT
// called when the plugin is hot-reloaded. Plugins should not assume this
// function will be called on every startup.
//
//nd:capability name=lifecycle
type Lifecycle interface {
	// OnInit is called after a plugin is fully loaded with all services registered.
	// Plugins can use this function to perform one-time initialization tasks.
	// The output can contain an error string if initialization failed, which will be
	// logged but will not prevent the plugin from being loaded.
	//nd:export name=nd_on_init
	OnInit(OnInitInput) (OnInitOutput, error)
}

// OnInitInput is the input provided to the init callback.
// Currently empty, reserved for future use.
type OnInitInput struct{}

// OnInitOutput is the output from the init callback.
type OnInitOutput struct {
	// Error is the error message if initialization failed.
	// Empty or null indicates success.
	// The error is logged but does not prevent the plugin from being loaded.
	Error *string `json:"error,omitempty"`
}
