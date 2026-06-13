package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPrintScript verifies --print-script exits 0 and emits the script to stdout
// without invoking k6 (per D-11).
func TestPrintScript(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"https://example.com"},
		Flags{VUs: 10, Duration: "30s", Method: "GET", PrintScript: true},
		&stdout, &stderr,
	)
	require.Equal(t, 0, code, "expected exit 0 for --print-script; stderr: %s", stderr.String())
	require.Contains(t, stdout.String(), "export const options")
	require.Contains(t, stdout.String(), `import http from "k6/http"`)
	require.Empty(t, stderr.String(), "expected no stderr output for --print-script")
}

// TestRun_NoK6 verifies that a missing k6 binary produces an actionable error
// with the install URL and exits 1 (per D-09).
//
// When k6 IS installed on the machine, this test is skipped — the "not found"
// code path is verified by source inspection (grep-verifiable pattern in run.go).
func TestRun_NoK6(t *testing.T) {
	k6Path, err := exec.LookPath("k6")
	if err != nil {
		// k6 not installed — test the error path directly.
		var stdout, stderr bytes.Buffer
		code := run(
			[]string{"https://example.com"},
			Flags{VUs: 10, Duration: "30s", Method: "GET"},
			&stdout, &stderr,
		)
		require.Equal(t, 1, code)
		require.Contains(t, stderr.String(), "https://grafana.com/docs/k6/latest/set-up/install-k6/")
		require.Contains(t, stderr.String(), "k6 not found in PATH")
		return
	}
	// k6 IS installed — skip the "not found" path.
	t.Logf("k6 found at %s — TestRun_NoK6 PATH-not-found branch skipped (k6 present)", k6Path)
	t.Skip("k6 is installed; install-URL error path only exercised when k6 absent")
}

// TestRun_InvalidURL verifies that an invalid URL (non-http/https scheme) exits 1
// with an actionable error message (per D-08).
func TestRun_InvalidURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"ftp://bad-scheme.com"},
		Flags{VUs: 10, Duration: "30s", Method: "GET"},
		&stdout, &stderr,
	)
	require.Equal(t, 1, code, "expected exit 1 for invalid URL scheme")
	require.Contains(t, stderr.String(), "http or https")
}

// TestRun_MissingURL verifies that missing URL argument exits 1.
func TestRun_MissingURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{},
		Flags{VUs: 10, Duration: "30s", Method: "GET"},
		&stdout, &stderr,
	)
	require.Equal(t, 1, code, "expected exit 1 for missing URL")
	require.NotEmpty(t, stderr.String())
}

// TestRun_InvalidHeader verifies that a malformed header (no colon) exits 1 (per D-08).
func TestRun_InvalidHeader(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"https://example.com"},
		Flags{VUs: 10, Duration: "30s", Method: "GET", Headers: []string{"BadHeaderNoColon"}},
		&stdout, &stderr,
	)
	require.Equal(t, 1, code, "expected exit 1 for invalid header format")
	require.Contains(t, stderr.String(), "colon")
}

// TestGeneratedScriptPassesK6Inspect validates that every generated golden script
// passes k6 inspect without errors (per D-13, QCK-07).
// Skipped when k6 is not in PATH (CI without k6 stays green).
func TestGeneratedScriptPassesK6Inspect(t *testing.T) {
	k6Path, err := exec.LookPath("k6")
	if err != nil {
		t.Skip("k6 not in PATH — skipping k6 inspect validation")
	}

	cases := []struct {
		name  string
		flags Flags
	}{
		{
			name:  "defaults",
			flags: Flags{URL: "https://api.example.com/health", VUs: 10, Duration: "30s", Method: "GET"},
		},
		{
			name:  "full_closed",
			flags: Flags{URL: "https://api.example.com/health", VUs: 50, Duration: "60s", Method: "POST"},
		},
		{
			name:  "iterations",
			flags: Flags{URL: "https://example.com", VUs: 5, Iterations: 100, Method: "GET"},
		},
		{
			name:  "rps",
			flags: Flags{URL: "https://example.com", VUs: 10, Duration: "30s", RPS: 50, Method: "GET"},
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

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			script := generateScript(tc.flags)
			//nolint:gosec // k6Path from LookPath; script is test-generated
			cmd := exec.Command(k6Path, "inspect", "-")
			cmd.Stdin = strings.NewReader(script)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "k6 inspect failed for case %q: %s\n\nScript:\n%s", tc.name, out, script)
		})
	}
}
