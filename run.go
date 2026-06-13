package main

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// run is the primary entrypoint for k6-quick logic. Returns exit code:
//
//	0 — success (k6 completed or --print-script)
//	1 — usage/validation/IO error or k6 not found
//	other — k6's exit code propagated verbatim (e.g. 99 for threshold failures)
//
// stdout/stderr injection allows full testing without subprocess spawning.
func run(args []string, flags Flags, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "error: URL argument is required")
		return 1
	}
	flags.URL = args[0]

	// Validate URL (per D-08).
	if err := validateURL(flags.URL); err != nil {
		fmt.Fprintln(stderr, "error: "+err.Error())
		return 1
	}

	// Validate and parse headers (per D-08) — pre-validate before generating script.
	_, err := parseHeaders(flags.Headers)
	if err != nil {
		fmt.Fprintln(stderr, "error: "+err.Error())
		return 1
	}

	// Validate method (per D-08) — warns on unknown, does not block.
	flags.Method = validateMethod(flags.Method, stderr)

	// Generate script (per D-02 — options baked in, not passed as k6 flags).
	script := generateScript(flags)

	// --print-script: emit to stdout and exit 0 without invoking k6 (per D-11).
	if flags.PrintScript {
		fmt.Fprint(stdout, script)
		return 0
	}

	// Locate k6 binary (per D-09).
	k6Path, err := exec.LookPath("k6")
	if err != nil {
		fmt.Fprintln(stderr, "k6 not found in PATH. Install k6: https://grafana.com/docs/k6/latest/set-up/install-k6/")
		fmt.Fprintln(stderr, "Or build a custom k6 with xk6: https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/")
		return 1
	}

	// Delegate to k6 via stdin pipe (per D-01 — zero temp files, per D-10 — passthrough).
	// k6 run - reads the entire stdin as the script source.
	cmd := exec.Command(k6Path, "run", "-") //nolint:gosec // k6Path comes from LookPath
	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = stdout // Direct passthrough — k6 writes progress bars + summary
	cmd.Stderr = stderr // Direct passthrough — k6 writes warnings + errors

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Propagate k6's exit code verbatim (99 = threshold failure, 108 = aborted, etc.)
			return exitErr.ExitCode()
		}
		// Non-exit error: k6 process failed to start after LookPath succeeded (rare).
		fmt.Fprintln(stderr, "error running k6: "+err.Error())
		return 1
	}
	return 0
}
