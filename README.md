# HTTP Client Library

A lightweight, production-ready HTTP client library for Go with a fluent chain API.

## Features

- ✅ **Chain API** - Fluent, readable request building
- ✅ **Interface Abstraction** - Easy to mock for unit testing
- ✅ **Retry with Exponential Backoff** - Automatic retry with jitter (AWS best practices)
- ✅ **Middleware Support** - Pluggable authentication, logging, etc.
- ✅ **Debug Mode** - Built-in request/response logging for debugging
- ✅ **Structured Error Handling** - Rich error types with helper methods
- ✅ **Context Propagation** - Full support for request cancellation and timeouts
- ✅ **Connection Pooling** - Reuses HTTP connections for performance
- ✅ **No Global State** - Thread-safe, testable design

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "time"
    
    httpclient "github.com/futuretea/go-http-client"
)

func main() {
    // Create client
    client := httpclient.NewClient(&httpclient.Config{
        BaseURL: "https://api.example.com",
        Timeout: 30 * time.Second,
    })
    
    // Make a GET request
    var result map[string]interface{}
    err := client.GET("/api/v1/users").Do(&result)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Result: %v\n", result)
}
```

### POST with JSON

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

var user User
err := client.POST("/api/v1/users").
    WithJSON(CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }).
    Do(&user)
```

### With Retry and Authentication

```go
client := httpclient.NewClient(
    &httpclient.Config{
        BaseURL: "https://api.example.com",
        Timeout: 30 * time.Second,
    },
    // Enable retry with exponential backoff
    httpclient.WithRetry(3, 200*time.Millisecond, 10*time.Second),
    // Add authentication middleware
    httpclient.WithMiddleware(httpclient.AuthMiddleware("Bearer", "your-token")),
)
```

### Custom Headers and Query Parameters

```go
var result map[string]interface{}
err := client.GET("/api/v1/resources").
    WithHeader("X-Custom-Header", "value").
    WithQuery("page", "1").
    WithQuery("limit", "10").
    Do(&result)
```

### Error Handling

```go
var result map[string]interface{}
err := client.GET("/api/v1/resources/123").Do(&result)

if err != nil {
    // Type assert to APIError for detailed information
    if apiErr, ok := err.(*httpclient.APIError); ok {
        if apiErr.IsNotFound() {
            fmt.Println("Resource not found")
            return
        }
        if apiErr.IsServerError() {
            fmt.Println("Server error, retry later")
            return
        }
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        return
    }
    fmt.Printf("Error: %v\n", err)
}
```

## Configuration

### Client Config

```go
config := &httpclient.Config{
    BaseURL: "https://api.example.com",
    Timeout: 30 * time.Second,
    
    // Optional: Connection pool settings
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}

client := httpclient.NewClient(config)
```

### Options

#### Retry Configuration

```go
httpclient.WithRetry(
    maxAttempts,    // e.g., 3
    waitTime,       // e.g., 200 * time.Millisecond
    maxWaitTime,    // e.g., 10 * time.Second
)
```

Retry will automatically retry on:
- Network errors
- 5xx server errors
- 429 Too Many Requests

#### Authentication Middleware

```go
// Bearer Token
httpclient.WithMiddleware(httpclient.AuthMiddleware("Bearer", "token"))

// API Key
httpclient.WithMiddleware(httpclient.AuthMiddleware("APIKey", "key"))

// Basic Auth
httpclient.WithMiddleware(httpclient.AuthMiddleware("Basic", base64EncodedCreds))
```

#### Custom Middleware

```go
customMiddleware := func(req *http.Request) error {
    req.Header.Set("X-Request-ID", generateRequestID())
    return nil
}

client := httpclient.NewClient(config, httpclient.WithMiddleware(customMiddleware))
```

#### Debug Middleware

Log HTTP requests and responses for debugging:

**Request-only debugging:**
```go
// Simple debug mode (color output to stdout)
client := httpclient.NewClient(config,
    httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)))
```

**Full request/response debugging:**
```go
// Debug both requests and responses
client := httpclient.NewClient(config,
    // Debug requests
    httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)),
    // Debug responses
    httpclient.WithResponseMiddleware(httpclient.DebugResponseMiddleware(nil)))
```

**Custom debug options:**
```go
client := httpclient.NewClient(config,
    httpclient.WithMiddleware(httpclient.DebugMiddleware(&httpclient.DebugOptions{
        Color:    false,      // Disable color highlighting
        Writer:   logFile,    // Write to file instead of stdout
        ShowBody: true,       // Show request body
    })),
    httpclient.WithResponseMiddleware(httpclient.DebugResponseMiddleware(&httpclient.DebugOptions{
        Color:    false,
        Writer:   logFile,
        ShowBody: true,       // Show response body
    })))
```

Debug output format:
```
> POST /api/v1/users HTTP/1.1
> Content-Type: application/json
> Authorization: Bearer token
>
{"name":"John","email":"john@example.com"}

< HTTP/1.1 201 Created
< Content-Type: application/json
<
{"id":"123","name":"John","email":"john@example.com"}
```

## Testing

### Mocking the Client

The `Client` interface makes it easy to mock for testing:

```go
type MockClient struct {
    GetFunc func(path string) *RequestBuilder
}

func (m *MockClient) GET(path string) *RequestBuilder {
    if m.GetFunc != nil {
        return m.GetFunc(path)
    }
    return nil
}

// Use in tests
func TestMyService(t *testing.T) {
    mockClient := &MockClient{
        GetFunc: func(path string) *RequestBuilder {
            // Return mock response
        },
    }
    
    service := NewService(mockClient)
    // Test service...
}
```

### Using httptest

```go
func TestClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    }))
    defer server.Close()
    
    client := httpclient.NewClient(&httpclient.Config{
        BaseURL: server.URL,
        Timeout: 5 * time.Second,
    })
    
    var result map[string]string
    err := client.GET("/api/test").Do(&result)
    
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
}
```

## Design Principles

### 1. Interface Abstraction
The library defines clear interfaces (`Client`, `Doer`) that make it easy to mock and test.

### 2. No Global State
All configuration is passed explicitly, making the library thread-safe and testable.

### 3. Built on Standards
- Interface abstractions for testability
- No global configuration
- Standard library compatibility

### 4. Error Handling
Structured `APIError` type with helper methods:
- `IsNotFound()` - Check for 404
- `IsUnauthorized()` - Check for 401
- `IsClientError()` - Check for 4xx
- `IsServerError()` - Check for 5xx

## Comparison

| Feature | This Library | net/http |
|---------|-------------|----------|
| Chain API | ✅ | ❌ |
| Interface Abstraction | ✅ | ✅ |
| No Global State | ✅ | ✅ |
| Retry Built-in | ✅ | ❌ |
| Auto Content-Type | ✅ | ❌ |
| Easy to Mock | ✅ | ✅ |

## License

Same as the parent project.
