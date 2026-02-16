package httpclient_test

import (
	"fmt"
	"os"
	"time"

	httpclient "github.com/futuretea/go-http-client"
)

// Example_debugMiddleware demonstrates how to use debug middleware
func Example_debugMiddleware() {
	// 1. Simple debug mode (color output to stdout)
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	}, httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)))

	var result map[string]interface{}
	_ = client.GET("/api/users").Do(&result)

	// Output will be:
	// > GET /api/users HTTP/1.1
	// > Content-Type: application/json
	// >
}

// Example_debugMiddleware_noColor demonstrates debug without color
func Example_debugMiddleware_noColor() {
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	}, httpclient.WithMiddleware(httpclient.DebugMiddleware(&httpclient.DebugOptions{
		Color:    false, // Disable color
		Writer:   os.Stdout,
		ShowBody: true,
	})))

	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var result map[string]interface{}
	_ = client.POST("/api/users").
		WithJSON(CreateUserRequest{
			Name:  "John",
			Email: "john@example.com",
		}).
		Do(&result)

	// Output will be:
	// > POST /api/users HTTP/1.1
	// > Content-Type: application/json
	// >
	// {"name":"John","email":"john@example.com"}
}

// Example_debugMiddleware_toFile demonstrates writing debug output to a file
func Example_debugMiddleware_toFile() {
	logFile, err := os.Create("http-debug.log")
	if err != nil {
		fmt.Printf("Failed to create log file: %v\n", err)
		return
	}
	defer func() { _ = logFile.Close() }()

	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	}, httpclient.WithMiddleware(httpclient.DebugMiddleware(&httpclient.DebugOptions{
		Color:    false, // No color codes in file
		Writer:   logFile,
		ShowBody: true,
	})))

	var result map[string]interface{}
	_ = client.GET("/api/resources").
		WithQuery("page", "1").
		WithQuery("limit", "10").
		Do(&result)

	fmt.Println("Debug output written to http-debug.log")
	// Output: Debug output written to http-debug.log
}

// Example_debugMiddleware_withoutBody demonstrates debug without body logging
func Example_debugMiddleware_withoutBody() {
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	}, httpclient.WithMiddleware(httpclient.DebugMiddleware(&httpclient.DebugOptions{
		Color:    true,
		Writer:   os.Stdout,
		ShowBody: false, // Don't log request body
	})))

	var result map[string]interface{}
	_ = client.POST("/api/sensitive").
		WithJSON(map[string]string{
			"password": "secret123",
		}).
		Do(&result)

	// Output will NOT contain the password in body
	// > POST /api/sensitive HTTP/1.1
	// > Content-Type: application/json
	// >
}

// Example_debugMiddleware_combined demonstrates combining debug with other middleware
func Example_debugMiddleware_combined() {
	// Combine debug with authentication and retry
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	},
		// Enable retry
		httpclient.WithRetry(3, 200*time.Millisecond, 10*time.Second),
		// Add authentication
		httpclient.WithMiddleware(httpclient.AuthMiddleware("Bearer", "your-token")),
		// Add debug logging
		httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)),
	)

	var result map[string]interface{}
	_ = client.GET("/api/protected").Do(&result)

	// Output will show:
	// > GET /api/protected HTTP/1.1
	// > Authorization: Bearer your-token
	// >
}

// Example_debugMiddleware_fullDebug demonstrates full request/response debugging
func Example_debugMiddleware_fullDebug() {
	// Enable both request and response debugging
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	},
		// Debug requests
		httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)),
		// Debug responses
		httpclient.WithResponseMiddleware(httpclient.DebugResponseMiddleware(nil)),
	)

	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var result map[string]interface{}
	_ = client.POST("/api/users").
		WithJSON(CreateUserRequest{
			Name:  "John",
			Email: "john@example.com",
		}).
		Do(&result)

	// Output will show both request and response:
	// > POST /api/users HTTP/1.1
	// > Content-Type: application/json
	// >
	// {"name":"John","email":"john@example.com"}
	//
	// < HTTP/1.1 201 Created
	// < Content-Type: application/json
	// <
	// {"id":"123","name":"John","email":"john@example.com"}
}
