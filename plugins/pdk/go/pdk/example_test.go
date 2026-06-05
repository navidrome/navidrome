// Example test demonstrating how to use the PDK mock for unit testing.
// This file is only compiled for non-WASM builds.
//
//go:build !wasip1

package pdk_test

import (
	"testing"

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
// Examples using stub types (Memory, HTTPRequest, HTTPResponse)
// =============================================================================

// FetchData demonstrates a plugin function that makes an HTTP request.
func FetchData(url string) ([]byte, error) {
	// Create and configure the HTTP request
	// Note: SetHeader and SetBody work directly on the stub - no mocking needed!
	req := pdk.NewHTTPRequest(pdk.MethodGet, url)
	req.SetHeader("Accept", "application/json")
	req.SetHeader("User-Agent", "MyPlugin/1.0")

	// Send the request - this is mocked because it requires host interaction
	resp := req.Send()

	// Check status - works directly on the stub
	if resp.Status() != 200 {
		return nil, nil
	}

	// Return body - works directly on the stub
	return resp.Body(), nil
}

func TestFetchData(t *testing.T) {
	pdk.ResetMock()

	// Create a stub response with test data
	expectedBody := []byte(`{"result": "success"}`)
	stubResponse := pdk.NewStubHTTPResponse(200, map[string]string{
		"Content-Type": "application/json",
	}, expectedBody)

	// Mock NewHTTPRequest to return a real HTTPRequest struct
	// The struct methods (SetHeader, SetBody) work without mocking
	pdk.PDKMock.On("NewHTTPRequest", pdk.MethodGet, "https://api.example.com/data").
		Return(&pdk.HTTPRequest{})

	// Mock Send to return our stub response
	pdk.PDKMock.On("Send", mock.AnythingOfType("*pdk.HTTPRequest")).
		Return(stubResponse)

	// Call the function
	body, err := FetchData("https://api.example.com/data")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != string(expectedBody) {
		t.Errorf("expected body %q, got %q", expectedBody, body)
	}

	pdk.PDKMock.AssertExpectations(t)
}

func TestFetchData_NonOKStatus(t *testing.T) {
	pdk.ResetMock()

	// Create a stub response with 404 status
	stubResponse := pdk.NewStubHTTPResponse(404, nil, []byte("Not Found"))

	pdk.PDKMock.On("NewHTTPRequest", pdk.MethodGet, "https://api.example.com/missing").
		Return(&pdk.HTTPRequest{})
	pdk.PDKMock.On("Send", mock.AnythingOfType("*pdk.HTTPRequest")).
		Return(stubResponse)

	// Call the function
	body, err := FetchData("https://api.example.com/missing")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return nil for non-200 status
	if body != nil {
		t.Errorf("expected nil body for 404, got %q", body)
	}

	pdk.PDKMock.AssertExpectations(t)
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
	req := pdk.NewHTTPRequest(pdk.MethodPost, url)
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(data) // Works directly on stub

	resp := req.Send() // This is mocked
	return int(resp.Status()), nil
}

func TestPostJSON(t *testing.T) {
	pdk.ResetMock()

	stubResponse := pdk.NewStubHTTPResponse(201, nil, nil)

	pdk.PDKMock.On("NewHTTPRequest", pdk.MethodPost, "https://api.example.com/items").
		Return(&pdk.HTTPRequest{})
	pdk.PDKMock.On("Send", mock.AnythingOfType("*pdk.HTTPRequest")).
		Return(stubResponse)

	status, err := PostJSON("https://api.example.com/items", []byte(`{"name":"test"}`))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != 201 {
		t.Errorf("expected status 201, got %d", status)
	}

	pdk.PDKMock.AssertExpectations(t)
}
