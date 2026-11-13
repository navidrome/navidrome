// Example test demonstrating how to use the PDK mock for unit testing.
// This file is only compiled for non-WASM builds.
//
//go:build !wasip1

package pdk_test

import (
	"testing"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/stretchr/testify/mock"
)

// ExamplePlugin demonstrates a simple plugin that uses PDK functions.
type ExamplePlugin struct{}

// ProcessMessage reads input, logs it, and outputs a response.
func (p *ExamplePlugin) ProcessMessage() error {
	// Get configuration
	prefix, ok := pdk.GetConfig("message_prefix")
	if !ok {
		prefix = "Hello"
	}

	// Read input
	message := pdk.InputString()

	// Log the message
	pdk.Log(pdk.LogInfo, "Processing: "+message)

	// Output the response
	pdk.OutputString(prefix + ", " + message + "!")

	return nil
}

func TestExamplePlugin_ProcessMessage(t *testing.T) {
	// Reset mock state before the test
	pdk.ResetMock()

	// Set up expectations
	pdk.PDKMock.On("GetConfig", "message_prefix").Return("Hi", true)
	pdk.PDKMock.On("InputString").Return("World")
	pdk.PDKMock.On("Log", pdk.LogInfo, "Processing: World").Return()
	pdk.PDKMock.On("OutputString", "Hi, World!").Return()

	// Call the plugin function
	plugin := &ExamplePlugin{}
	err := plugin.ProcessMessage()

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all expected calls were made
	pdk.PDKMock.AssertExpectations(t)
}

func TestExamplePlugin_ProcessMessage_DefaultPrefix(t *testing.T) {
	// Reset mock state before the test
	pdk.ResetMock()

	// Set up expectations - config key not found
	pdk.PDKMock.On("GetConfig", "message_prefix").Return("", false)
	pdk.PDKMock.On("InputString").Return("Test")
	pdk.PDKMock.On("Log", pdk.LogInfo, "Processing: Test").Return()
	pdk.PDKMock.On("OutputString", "Hello, Test!").Return()

	// Call the plugin function
	plugin := &ExamplePlugin{}
	err := plugin.ProcessMessage()

	// Verify no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all expected calls were made
	pdk.PDKMock.AssertExpectations(t)
}

// Example of testing JSON input/output
type Request struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Response struct {
	Message string `json:"message"`
	Total   int    `json:"total"`
}

func ProcessJSONRequest() error {
	var req Request
	if err := pdk.InputJSON(&req); err != nil {
		pdk.SetError(err)
		return err
	}

	resp := Response{
		Message: "Hello, " + req.Name,
		Total:   req.Count * 2,
	}

	return pdk.OutputJSON(resp)
}

func TestProcessJSONRequest(t *testing.T) {
	pdk.ResetMock()

	// Mock InputJSON to populate the request struct
	pdk.PDKMock.On("InputJSON", mock.AnythingOfType("*pdk_test.Request")).
		Return(nil).
		Run(func(args mock.Arguments) {
			req := args.Get(0).(*Request)
			req.Name = "Alice"
			req.Count = 5
		})

	// Expect OutputJSON with the correct response
	pdk.PDKMock.On("OutputJSON", Response{
		Message: "Hello, Alice",
		Total:   10,
	}).Return(nil)

	// Call the function
	err := ProcessJSONRequest()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pdk.PDKMock.AssertExpectations(t)
}

// =============================================================================
// HTTP requests go through the host HTTP service (host.HTTPSend), not
// pdk.NewHTTPRequest, which Navidrome does not enable.
// =============================================================================

// FetchData demonstrates a plugin function that makes an HTTP request.
func FetchData(url string) ([]byte, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method: "GET",
		URL:    url,
		Headers: map[string]string{
			"Accept":     "application/json",
			"User-Agent": "MyPlugin/1.0",
		},
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, nil
	}
	return resp.Body, nil
}

func TestFetchData(t *testing.T) {
	host.HTTPMock.ExpectedCalls = nil

	expectedBody := []byte(`{"result": "success"}`)
	host.HTTPMock.On("Send", mock.MatchedBy(func(req host.HTTPRequest) bool {
		return req.Method == "GET" && req.URL == "https://api.example.com/data" &&
			req.Headers["Accept"] == "application/json"
	})).Return(&host.HTTPResponse{StatusCode: 200, Body: expectedBody}, nil)

	body, err := FetchData("https://api.example.com/data")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != string(expectedBody) {
		t.Errorf("expected body %q, got %q", expectedBody, body)
	}

	host.HTTPMock.AssertExpectations(t)
}

func TestFetchData_NonOKStatus(t *testing.T) {
	host.HTTPMock.ExpectedCalls = nil

	host.HTTPMock.On("Send", mock.Anything).
		Return(&host.HTTPResponse{StatusCode: 404, Body: []byte("Not Found")}, nil)

	body, err := FetchData("https://api.example.com/missing")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return nil for non-200 status
	if body != nil {
		t.Errorf("expected nil body for 404, got %q", body)
	}

	host.HTTPMock.AssertExpectations(t)
}

// ProcessMemoryData demonstrates working with Memory type.
func ProcessMemoryData(mem pdk.Memory) string {
	// Memory methods work directly on the stub - no mocking needed!
	data := mem.ReadBytes()
	return "Processed " + string(data) + " (length: " + formatUint64(mem.Length()) + ")"
}

func formatUint64(n uint64) string {
	return string(rune('0' + n%10)) // Simplified for demo
}

func TestProcessMemoryData(t *testing.T) {
	// Create stub memory with test data - no mocking needed!
	mem := pdk.NewStubMemory(0, 5, []byte("hello"))

	result := ProcessMemoryData(mem)

	expected := "Processed hello (length: 5)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// StoreAndRetrieve demonstrates Memory Store/Load methods.
func TestMemoryStoreAndLoad(t *testing.T) {
	// Create empty memory
	mem := pdk.NewStubMemory(100, 0, nil)

	// Store data - works directly, no mock needed
	mem.Store([]byte("test data"))

	// Verify the data was stored
	if mem.Length() != 9 {
		t.Errorf("expected length 9, got %d", mem.Length())
	}

	// Load into buffer
	buffer := make([]byte, 9)
	mem.Load(buffer)

	if string(buffer) != "test data" {
		t.Errorf("expected 'test data', got %q", buffer)
	}

	// Free the memory
	mem.Free()

	if mem.Length() != 0 {
		t.Errorf("expected length 0 after free, got %d", mem.Length())
	}
}

// HTTPMethodString demonstrates that HTTPMethod.String() works without mocking.
func TestHTTPMethodString(t *testing.T) {
	// These work directly - no mocking needed!
	tests := []struct {
		method   pdk.HTTPMethod
		expected string
	}{
		{pdk.MethodGet, "GET"},
		{pdk.MethodPost, "POST"},
		{pdk.MethodPut, "PUT"},
		{pdk.MethodDelete, "DELETE"},
	}

	for _, tc := range tests {
		result := tc.method.String()
		if result != tc.expected {
			t.Errorf("expected %q for method %d, got %q", tc.expected, tc.method, result)
		}
	}
}

// PostJSON demonstrates a more complex HTTP request with body.
func PostJSON(url string, data []byte) (int, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:  "POST",
		URL:     url,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    data,
	})
	if err != nil {
		return 0, err
	}
	return int(resp.StatusCode), nil
}

func TestPostJSON(t *testing.T) {
	host.HTTPMock.ExpectedCalls = nil

	host.HTTPMock.On("Send", host.HTTPRequest{
		Method:  "POST",
		URL:     "https://api.example.com/items",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"name":"test"}`),
	}).Return(&host.HTTPResponse{StatusCode: 201}, nil)

	status, err := PostJSON("https://api.example.com/items", []byte(`{"name":"test"}`))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != 201 {
		t.Errorf("expected status 201, got %d", status)
	}

	host.HTTPMock.AssertExpectations(t)
}
