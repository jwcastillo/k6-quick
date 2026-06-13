package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

var knownMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
}

// validateURL parses the URL and checks for http/https scheme.
func validateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got %q", u.Scheme)
	}
	return nil
}

// parseHeaders converts []string "Key: Value" into map[string]string.
// Returns error if any entry has no colon.
func parseHeaders(headers []string) (map[string]string, error) {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("header %q must contain a colon in \"Key: Value\" format", h)
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out, nil
}

// validateMethod uppercases the method, warns on unknown HTTP verbs.
// Returns the uppercased method. Writes a warning to stderr if the method is unknown.
func validateMethod(method string, stderr io.Writer) string {
	upper := strings.ToUpper(method)
	if !knownMethods[upper] {
		fmt.Fprintf(stderr, "warning: unknown HTTP method %q — passing through to k6\n", upper)
	}
	return upper
}
