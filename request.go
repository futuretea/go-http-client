package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RequestBuilder provides a fluent API for building HTTP requests
type RequestBuilder struct {
	client  *HTTPClient
	method  string
	path    string
	body    []byte
	headers map[string]string
	query   url.Values
	ctx     context.Context
	err     error
}

// Middleware is a function that can inspect/modify HTTP requests before they are sent
type Middleware func(*http.Request) error

// ResponseMiddleware is a function that can inspect/modify HTTP responses after they are received
// The response body will be restored after middleware execution
type ResponseMiddleware func(*http.Response) error

// GET sets the HTTP method to GET
func (b *RequestBuilder) GET(path string) *RequestBuilder {
	b.method = http.MethodGet
	b.path = path
	return b
}

// POST sets the HTTP method to POST
func (b *RequestBuilder) POST(path string) *RequestBuilder {
	b.method = http.MethodPost
	b.path = path
	return b
}

// PUT sets the HTTP method to PUT
func (b *RequestBuilder) PUT(path string) *RequestBuilder {
	b.method = http.MethodPut
	b.path = path
	return b
}

// DELETE sets the HTTP method to DELETE
func (b *RequestBuilder) DELETE(path string) *RequestBuilder {
	b.method = http.MethodDelete
	b.path = path
	return b
}

// PATCH sets the HTTP method to PATCH
func (b *RequestBuilder) PATCH(path string) *RequestBuilder {
	b.method = http.MethodPatch
	b.path = path
	return b
}

// WithContext sets the request context.
// The context must not be nil.
func (b *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	b.ctx = ctx
	return b
}

// WithJSON serializes the given object as JSON and sets it as the request body
// Automatically sets Content-Type: application/json
func (b *RequestBuilder) WithJSON(v interface{}) *RequestBuilder {
	if b.err != nil {
		return b
	}

	data, err := json.Marshal(v)
	if err != nil {
		b.err = fmt.Errorf("failed to marshal JSON: %w", err)
		return b
	}

	b.body = data
	b.headers["Content-Type"] = "application/json"
	return b
}

// WithBody sets the request body directly
func (b *RequestBuilder) WithBody(body []byte) *RequestBuilder {
	b.body = body
	return b
}

// WithHeader sets a single header
func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

// WithHeaders sets multiple headers
func (b *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		b.headers[k] = v
	}
	return b
}

// WithQuery adds query parameters
func (b *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	if b.query == nil {
		b.query = url.Values{}
	}
	b.query.Add(key, value)
	return b
}

// WithQueryParams adds multiple query parameters
func (b *RequestBuilder) WithQueryParams(params map[string]string) *RequestBuilder {
	if b.query == nil {
		b.query = url.Values{}
	}
	for k, v := range params {
		b.query.Add(k, v)
	}
	return b
}

// Do executes the HTTP request and parses the response
func (b *RequestBuilder) Do(result interface{}) error {
	if b.err != nil {
		return b.err
	}

	resp, err := b.execute()
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle error responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return handleErrorResponse(resp)
	}

	// Parse response if result is provided
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// DoWithResponse executes the HTTP request and returns the raw response
// This is useful when you need access to response headers or status code
func (b *RequestBuilder) DoWithResponse() (*http.Response, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.execute()
}

// execute builds and executes the actual HTTP request
func (b *RequestBuilder) execute() (*http.Response, error) {
	// Build full URL by properly joining base URL and path
	fullURL := joinURL(b.client.baseURL, b.path)
	if len(b.query) > 0 {
		fullURL += "?" + b.query.Encode()
	}

	// Create body reader
	var bodyReader io.Reader
	if b.body != nil {
		bodyReader = bytes.NewReader(b.body)
	}

	// Create request
	req, err := http.NewRequestWithContext(b.ctx, b.method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range b.headers {
		req.Header.Set(k, v)
	}

	// Apply middleware
	for _, mw := range b.client.middleware {
		if err := mw(req); err != nil {
			return nil, fmt.Errorf("middleware error: %w", err)
		}
	}

	// Execute with retry if configured
	var resp *http.Response
	if b.client.retryConfig != nil {
		resp, err = executeWithRetry(b.ctx, b.client.httpClient, req, b.client.retryConfig)
	} else {
		resp, err = b.client.httpClient.Do(req)
	}

	if err != nil {
		return nil, err
	}

	// Apply response middleware if configured
	if len(b.client.responseMiddleware) > 0 {
		if err := b.applyResponseMiddleware(resp); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
	}

	return resp, nil
}

// applyResponseMiddleware applies all response middleware to the response.
// It reads the body once, applies all middleware, and restores the body for downstream use.
// If any middleware fails, the body is still restored and the error is returned.
func (b *RequestBuilder) applyResponseMiddleware(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body for middleware: %w", err)
	}
	_ = resp.Body.Close()

	// Restore body so middleware can read it
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Apply all middleware
	for _, mw := range b.client.responseMiddleware {
		if err := mw(resp); err != nil {
			// Ensure body is restored even on error
			resp.Body = io.NopCloser(bytes.NewReader(body))
			return fmt.Errorf("response middleware error: %w", err)
		}
		// Restore body for next middleware
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return nil
}

// joinURL properly joins base URL and path, handling slashes correctly.
// It ensures there is exactly one slash between base and path.
func joinURL(base, p string) string {
	if p == "" {
		return base
	}
	// Remove trailing slash from base
	base = strings.TrimSuffix(base, "/")
	// Remove leading slash from path
	p = strings.TrimPrefix(p, "/")
	return base + "/" + p
}
