package httpclient_test

import (
	"context"
	"fmt"
	"time"

	httpclient "github.com/futuretea/go-http-client"
)

// Example: Basic usage with JSON
func ExampleClient_basic() {
	// Create HTTP client
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	// Define request and response types
	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// Make a POST request with JSON body
	var user User
	err := client.POST("/api/v1/users").
		WithJSON(CreateUserRequest{
			Name:  "John Doe",
			Email: "john@example.com",
		}).
		Do(&user)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created user: %s\n", user.ID)
}

// Example: With retry and authentication
func ExampleClient_withRetry() {
	client := httpclient.NewClient(
		&httpclient.Config{
			BaseURL: "https://api.example.com",
			Timeout: 30 * time.Second,
		},
		// Enable retry with exponential backoff
		httpclient.WithRetry(3, 200*time.Millisecond, 10*time.Second),
		// Add authentication middleware
		httpclient.WithMiddleware(httpclient.AuthMiddleware("Bearer", "your-token-here")),
	)

	var result map[string]interface{}
	err := client.GET("/api/v1/resources").Do(&result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
}

// Example: With context and custom headers
func ExampleClient_withContext() {
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result map[string]interface{}
	err := client.GET("/api/v1/resources").
		WithContext(ctx).
		WithHeader("X-Custom-Header", "custom-value").
		WithQuery("page", "1").
		WithQuery("limit", "10").
		Do(&result)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
}

// Example: Error handling with type assertion
func ExampleClient_errorHandling() {
	client := httpclient.NewClient(&httpclient.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	var result map[string]interface{}
	err := client.GET("/api/v1/resources/not-found").Do(&result)

	if err != nil {
		// Type assert to APIError for detailed error information
		if apiErr, ok := err.(*httpclient.APIError); ok {
			if apiErr.IsNotFound() {
				fmt.Println("Resource not found")
				return
			}
			if apiErr.IsServerError() {
				fmt.Println("Server error, please retry later")
				return
			}
			fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
			return
		}
		fmt.Printf("Error: %v\n", err)
	}
}

// Example: Custom retry logic
func ExampleClient_customRetry() {
	client := httpclient.NewClient(
		&httpclient.Config{
			BaseURL: "https://api.example.com",
			Timeout: 30 * time.Second,
		},
		httpclient.WithRetry(5, 500*time.Millisecond, 30*time.Second),
	)

	var result map[string]interface{}
	err := client.GET("/api/v1/resources").Do(&result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
}
