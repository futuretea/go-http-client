package httpclient

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDebugMiddleware_Basic(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Capture debug output
	var buf bytes.Buffer

	// Create client with debug middleware
	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: true,
	})))

	// Make request
	var result map[string]string
	err := client.GET("/api/test").Do(&result)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify debug output
	output := buf.String()

	// Should contain request line
	if !strings.Contains(output, "GET /api/test HTTP/1.1") {
		t.Errorf("Debug output missing request line, got: %s", output)
	}

	// Should contain ">" prefix for request
	if !strings.Contains(output, ">") {
		t.Errorf("Debug output missing '>' prefix, got: %s", output)
	}
}

func TestDebugMiddleware_WithJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: true,
	})))

	// POST with JSON body
	type User struct {
		Name string `json:"name"`
	}

	var result map[string]string
	err := client.POST("/api/users").
		WithJSON(User{Name: "John"}).
		Do(&result)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain POST method
	if !strings.Contains(output, "POST /api/users") {
		t.Errorf("Debug output missing POST request line, got: %s", output)
	}

	// Should contain Content-Type header
	if !strings.Contains(output, "Content-Type") {
		t.Errorf("Debug output missing Content-Type header, got: %s", output)
	}

	// Should contain JSON body
	if !strings.Contains(output, `"name":"John"`) {
		t.Errorf("Debug output missing request body, got: %s", output)
	}
}

func TestDebugMiddleware_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: false, // Disable body logging
	})))

	err := client.POST("/api/test").
		WithJSON(map[string]string{"key": "value"}).
		Do(nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain request line
	if !strings.Contains(output, "POST /api/test") {
		t.Errorf("Debug output missing request line, got: %s", output)
	}

	// Should NOT contain body (ShowBody = false)
	if strings.Contains(output, `"key":"value"`) {
		t.Errorf("Debug output should not contain body when ShowBody=false, got: %s", output)
	}
}

func TestDebugMiddleware_DefaultOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with nil options (should use defaults)
	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(nil)))

	err := client.GET("/api/test").Do(nil)
	if err != nil {
		t.Fatalf("Request with default debug options failed: %v", err)
	}

	// If we reach here, default options work correctly
	// (output goes to stdout, but we can't easily capture it in test)
}

func TestDebugMiddleware_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: true,
	})))

	err := client.GET("/api/resources").
		WithQuery("page", "1").
		WithQuery("limit", "10").
		Do(nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain query parameters in URL
	if !strings.Contains(output, "page=1") || !strings.Contains(output, "limit=10") {
		t.Errorf("Debug output missing query parameters, got: %s", output)
	}
}

func TestDebugMiddleware_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithMiddleware(DebugMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: true,
	})))

	err := client.GET("/api/test").
		WithHeader("X-Custom-Header", "custom-value").
		WithHeader("X-Request-ID", "12345").
		Do(nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain custom headers
	if !strings.Contains(output, "X-Custom-Header") || !strings.Contains(output, "custom-value") {
		t.Errorf("Debug output missing custom header, got: %s", output)
	}

	// Note: Go's http.Header canonicalizes header names, so "X-Request-ID" becomes "X-Request-Id"
	if !strings.Contains(output, "X-Request-Id") || !strings.Contains(output, "12345") {
		t.Errorf("Debug output missing X-Request-Id header, got: %s", output)
	}
}

func TestDebugResponseMiddleware_Basic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithResponseMiddleware(DebugResponseMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: true,
	})))

	var result map[string]string
	err := client.GET("/api/test").Do(&result)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain response status
	if !strings.Contains(output, "HTTP/1.1 200 OK") {
		t.Errorf("Debug output missing response status, got: %s", output)
	}

	// Should contain "<" prefix for response
	if !strings.Contains(output, "<") {
		t.Errorf("Debug output missing '<' prefix, got: %s", output)
	}

	// Should contain response body
	if !strings.Contains(output, `{"status":"ok"}`) {
		t.Errorf("Debug output missing response body, got: %s", output)
	}

	// Verify result was decoded correctly
	if result["status"] != "ok" {
		t.Errorf("Response not decoded correctly, got: %v", result)
	}
}

func TestDebugMiddleware_Full(t *testing.T) {
	// Test both request and response debug together
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Response", "response-value")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123","name":"Test"}`))
	}))
	defer server.Close()

	var buf bytes.Buffer

	// Create client with both request and response debug
	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	},
		WithMiddleware(DebugMiddleware(&DebugOptions{
			Color:    false,
			Writer:   &buf,
			ShowBody: true,
		})),
		WithResponseMiddleware(DebugResponseMiddleware(&DebugOptions{
			Color:    false,
			Writer:   &buf,
			ShowBody: true,
		})),
	)

	type CreateRequest struct {
		Name string `json:"name"`
	}

	var result map[string]string
	err := client.POST("/api/resources").WithJSON(CreateRequest{Name: "Test"}).Do(&result)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain request
	if !strings.Contains(output, "> POST /api/resources") {
		t.Errorf("Debug output missing request line, got: %s", output)
	}

	// Should contain request body
	if !strings.Contains(output, `"name":"Test"`) {
		t.Errorf("Debug output missing request body, got: %s", output)
	}

	// Should contain response
	if !strings.Contains(output, "< HTTP/1.1 201 Created") {
		t.Errorf("Debug output missing response status, got: %s", output)
	}

	// Should contain response headers
	if !strings.Contains(output, "X-Custom-Response") {
		t.Errorf("Debug output missing response header, got: %s", output)
	}

	// Should contain response body
	if !strings.Contains(output, `{"id":"123","name":"Test"}`) {
		t.Errorf("Debug output missing response body, got: %s", output)
	}

	// Verify result was decoded correctly
	if result["id"] != "123" || result["name"] != "Test" {
		t.Errorf("Response not decoded correctly, got: %v", result)
	}
}

func TestDebugResponseMiddleware_WithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"secret":"password123"}`))
	}))
	defer server.Close()

	var buf bytes.Buffer

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}, WithResponseMiddleware(DebugResponseMiddleware(&DebugOptions{
		Color:    false,
		Writer:   &buf,
		ShowBody: false, // Don't log response body
	})))

	err := client.GET("/api/secret").Do(nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	output := buf.String()

	// Should contain status
	if !strings.Contains(output, "200 OK") {
		t.Errorf("Debug output missing status, got: %s", output)
	}

	// Should NOT contain body
	if strings.Contains(output, "password123") {
		t.Errorf("Debug output should not contain body when ShowBody=false, got: %s", output)
	}
}
