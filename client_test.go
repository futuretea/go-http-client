package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := &Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	httpClient, ok := client.(*HTTPClient)
	if !ok {
		t.Fatal("Client is not *HTTPClient")
	}

	if httpClient.baseURL != config.BaseURL {
		t.Errorf("Expected baseURL %s, got %s", config.BaseURL, httpClient.baseURL)
	}
}

func TestClient_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/test" {
			t.Errorf("Expected path /api/v1/test, got %s", r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "success",
		})
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	var result map[string]string
	err := client.GET("/api/v1/test").Do(&result)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

func TestClient_POST_WithJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		_ = json.Unmarshal(body, &req)

		if req["name"] != "test" {
			t.Errorf("Expected name 'test', got '%s'", req["name"])
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   "123",
			"name": req["name"],
		})
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	var result map[string]string
	err := client.POST("/api/v1/users").
		WithJSON(map[string]string{"name": "test"}).
		Do(&result)

	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("Expected id '123', got '%s'", result["id"])
	}
}

func TestClient_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Expected X-Custom-Header not found")
		}
		if r.Header.Get("X-Another-Header") != "another-value" {
			t.Error("Expected X-Another-Header not found")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := client.GET("/api/v1/test").
		WithHeader("X-Custom-Header", "custom-value").
		WithHeaders(map[string]string{
			"X-Another-Header": "another-value",
		}).
		Do(nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestClient_WithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got page=%s", query.Get("page"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got limit=%s", query.Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	err := client.GET("/api/v1/test").
		WithQuery("page", "1").
		WithQuery("limit", "10").
		Do(nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestClient_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.GET("/api/v1/test").
		WithContext(ctx).
		Do(nil)

	if err == nil {
		t.Fatal("Expected context deadline exceeded error")
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "resource not found",
		})
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	var result map[string]string
	err := client.GET("/api/v1/notfound").Do(&result)

	if err == nil {
		t.Fatal("Expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatal("Expected APIError type")
	}

	if !apiErr.IsNotFound() {
		t.Error("Expected IsNotFound() to return true")
	}
}

func TestClient_WithMiddleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		&Config{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		},
		WithMiddleware(AuthMiddleware("Bearer", "test-token")),
	)

	err := client.GET("/api/v1/test").Do(nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}
