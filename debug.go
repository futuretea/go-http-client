package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// DebugOptions configures debug output behavior
type DebugOptions struct {
	Color    bool      // Enable color output (ANSI color codes)
	Writer   io.Writer // Writer to output debug information (default: os.Stdout)
	ShowBody bool      // Controls whether to print request/response body
}

// applyDefaults applies default values to DebugOptions
func (o *DebugOptions) applyDefaults() *DebugOptions {
	if o == nil {
		return &DebugOptions{
			Color:    true,
			Writer:   os.Stdout,
			ShowBody: true,
		}
	}
	if o.Writer == nil {
		o.Writer = os.Stdout
	}
	return o
}

// DebugMiddleware returns a middleware that logs HTTP requests for debugging
// This provides curl-style HTTP request logging for debugging purposes
//
// Example usage:
//
//	// Simple debug (color output to stdout)
//	client := httpclient.NewClient(config,
//	    httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)))
//
//	// Custom options
//	client := httpclient.NewClient(config,
//	    httpclient.WithMiddleware(httpclient.DebugMiddleware(&httpclient.DebugOptions{
//	        Color: false,
//	        Writer: logFile,
//	        ShowBody: true,
//	    })))
func DebugMiddleware(opts *DebugOptions) Middleware {
	opts = opts.applyDefaults()

	return func(req *http.Request) error {
		printRequestLine(opts.Writer, req)
		printHeaders(opts.Writer, opts.Color, ">", req.Header)

		if opts.ShowBody && req.Body != nil {
			return printBody(opts.Writer, opts.Color, req.Body, &req.Body)
		}
		return nil
	}
}

// ANSI color codes
const (
	colorReset      = "\033[0m"
	colorPurpleCode = "\033[35m"
	colorBlueCode   = "\033[34m"
)

func colorPurple(s string) string {
	return colorPurpleCode + s + colorReset
}

func colorBlue(s string) string {
	return colorBlueCode + s + colorReset
}

// DebugResponseMiddleware returns a middleware that logs HTTP responses for debugging
// This complements DebugMiddleware to provide full request/response logging
//
// Example usage:
//
//	client := httpclient.NewClient(config,
//	    httpclient.WithMiddleware(httpclient.DebugMiddleware(nil)),
//	    httpclient.WithResponseMiddleware(httpclient.DebugResponseMiddleware(nil)))
//
// Note: This middleware reads the entire response body. For large responses,
// consider disabling ShowBody or using a custom filter.
func DebugResponseMiddleware(opts *DebugOptions) ResponseMiddleware {
	opts = opts.applyDefaults()

	return func(resp *http.Response) error {
		_, _ = fmt.Fprintf(opts.Writer, "< %s %s\n", resp.Proto, resp.Status)
		printHeaders(opts.Writer, opts.Color, "<", resp.Header)

		if opts.ShowBody && resp.Body != nil {
			return printBody(opts.Writer, opts.Color, resp.Body, &resp.Body)
		}
		return nil
	}
}

// printRequestLine prints HTTP request line
func printRequestLine(w io.Writer, req *http.Request) {
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	_, _ = fmt.Fprintf(w, "> %s %s %s\n", req.Method, path, req.Proto)
}

// printHeaders prints HTTP headers with optional color
func printHeaders(w io.Writer, useColor bool, prefix string, headers http.Header) {
	for key, values := range headers {
		headerName := key
		headerValue := strings.Join(values, ", ")

		if useColor {
			headerName = colorPurple(key)
			headerValue = colorBlue(headerValue)
		}

		_, _ = fmt.Fprintf(w, "%s %s: %s\n", prefix, headerName, headerValue)
	}
	_, _ = fmt.Fprintf(w, "%s\n", prefix)
}

// printBody reads, prints and restores HTTP body
// The bodyPtr parameter is updated to point to the restored body
func printBody(w io.Writer, _ bool, body io.ReadCloser, bodyPtr *io.ReadCloser) error {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read body for debug: %w", err)
	}
	_ = body.Close()

	// Restore body immediately
	*bodyPtr = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) > 0 {
		_, _ = fmt.Fprintln(w, string(bodyBytes))
		_, _ = fmt.Fprintln(w)
	}
	return nil
}
