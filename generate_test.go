package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

type generateCase struct {
	name  string
	flags Flags
}

var generateCases = []generateCase{
	{
		name: "defaults",
		flags: Flags{
			URL:      "https://api.example.com/health",
			VUs:      10,
			Duration: "30s",
			Method:   "GET",
		},
	},
	{
		name: "full_closed",
		flags: Flags{
			URL:      "https://api.example.com/health",
			VUs:      50,
			Duration: "60s",
			Method:   "POST",
		},
	},
	{
		name: "iterations",
		flags: Flags{
			URL:        "https://example.com",
			VUs:        5,
			Iterations: 100,
			Method:     "GET",
		},
	},
	{
		name: "rps",
		flags: Flags{
			URL:      "https://example.com",
			VUs:      10,
			Duration: "30s",
			RPS:      50,
			Method:   "GET",
		},
	},
	{
		name: "headers_body",
		flags: Flags{
			URL:      "https://example.com/api",
			VUs:      10,
			Duration: "30s",
			Method:   "POST",
			Headers:  []string{"Content-Type: application/json", "Authorization: Bearer tok"},
			Body:     `{"name":"test"}`,
		},
	},
	{
		name: "thresholds",
		flags: Flags{
			URL:        "https://example.com",
			VUs:        10,
			Duration:   "30s",
			Method:     "GET",
			Thresholds: []string{"http_req_duration:p(95)<500", "http_req_failed:rate<0.01", "http_req_duration:p(99)<1000"},
		},
	},
	{
		name: "special_chars",
		flags: Flags{
			URL:      "https://example.com/q?a=1&b=2",
			VUs:      10,
			Duration: "30s",
			Method:   "GET",
			Headers:  []string{`X-Test: value"with"quotes`},
			Body:     "line1\nline2\tindented",
		},
	},
}

func TestGenerate(t *testing.T) {
	for _, tc := range generateCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := generateScript(tc.flags)
			goldenPath := filepath.Join("testdata", tc.name+".golden.js")
			if *update {
				require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
				require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0o644))
			}
			want, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "golden file not found; run with -update to create it")
			require.Equal(t, string(want), got)
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{name: "https valid", url: "https://example.com", wantErr: false},
		{name: "http valid", url: "http://example.com", wantErr: false},
		{name: "ftp scheme", url: "ftp://example.com", wantErr: true, errMsg: "http or https"},
		{name: "empty URL", url: "", wantErr: true},
		{name: "no scheme", url: "not-a-url", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	t.Run("single header", func(t *testing.T) {
		m, err := parseHeaders([]string{"Content-Type: application/json"})
		require.NoError(t, err)
		require.Equal(t, map[string]string{"Content-Type": "application/json"}, m)
	})
	t.Run("missing colon", func(t *testing.T) {
		_, err := parseHeaders([]string{"X-Bad-Header"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "colon")
	})
	t.Run("nil headers", func(t *testing.T) {
		m, err := parseHeaders(nil)
		require.NoError(t, err)
		require.Equal(t, map[string]string{}, m)
	})
}

func TestValidateMethod(t *testing.T) {
	t.Run("known method uppercased", func(t *testing.T) {
		result := validateMethod("get", io.Discard)
		require.Equal(t, "GET", result)
	})
	t.Run("unknown method warns", func(t *testing.T) {
		var buf strings.Builder
		result := validateMethod("CUSTOM_VERB", &buf)
		require.Equal(t, "CUSTOM_VERB", result)
		require.Contains(t, buf.String(), "warning")
	})
}
