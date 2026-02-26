package plugins

// CapabilityHTTPEndpoint indicates the plugin can handle incoming HTTP requests.
// Detected when the plugin exports the nd_http_handle_request function.
const CapabilityHTTPEndpoint Capability = "HTTPEndpoint"

const FuncHTTPHandleRequest = "nd_http_handle_request"

func init() {
	registerCapability(
		CapabilityHTTPEndpoint,
		FuncHTTPHandleRequest,
	)
}
