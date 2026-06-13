// Package main is the entry point for the k6-quick CLI tool.
// k6-quick generates and runs an ephemeral k6 load test against a target URL.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// buildRootCmd creates the cobra command with all flags registered (per D-03).
// stdout and stderr are injectable for testability.
func buildRootCmd(stdout, stderr io.Writer) (*cobra.Command, *Flags) {
	var flags Flags
	cmd := &cobra.Command{
		Use:   "k6-quick <url>",
		Short: "Zero-config URL load test: generates and runs an ephemeral k6 script",
		Long: `k6-quick generates a k6 script for the given URL and delegates to the k6 binary.

The generated script is piped to k6 via stdin — no temporary files are created.
Use --print-script to preview the generated script before running it.

Exit codes: k6's exit code is propagated verbatim (0=ok, 99=threshold failure, etc.)`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		// RunE is a no-op; all real logic is in run() called from main().
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	// Suppress unused parameter warning — stdout/stderr used by run().
	_ = stdout
	_ = stderr

	// -c/--vus: number of virtual users (default 10, per D-03)
	cmd.Flags().IntVarP(&flags.VUs, "vus", "c", 10, "number of virtual users")
	// -d/--duration: test duration (default "30s", per D-03)
	cmd.Flags().StringVarP(&flags.Duration, "duration", "d", "30s", "test duration (e.g. 30s, 1m)")
	// -n/--iterations: total iterations (overrides duration, per D-04)
	cmd.Flags().IntVarP(&flags.Iterations, "iterations", "n", 0, "total iterations (omits duration if set)")
	// --rps: arrival rate in requests per second — triggers arrival-rate template (per D-04)
	cmd.Flags().IntVar(&flags.RPS, "rps", 0, "target requests per second (generates arrival-rate scenario)")
	// -m/--method: HTTP method (default GET, per D-03)
	cmd.Flags().StringVarP(&flags.Method, "method", "m", "GET", "HTTP method")
	// -H/--header: repeatable header flag — StringArrayVar preserves each invocation intact
	// (NOT StringSliceVar — comma splitting corrupts header values per RESEARCH.md Pitfall 1)
	cmd.Flags().StringArrayVarP(&flags.Headers, "header", "H", nil, `request header in "Key: Value" format (repeatable)`)
	// -b/--body: request body
	cmd.Flags().StringVarP(&flags.Body, "body", "b", "", "request body")
	// --threshold: repeatable k6 threshold expressions "metric:expr" (per D-05)
	// StringArrayVar preserves each value intact (same pitfall as --header)
	cmd.Flags().StringArrayVar(&flags.Thresholds, "threshold", nil, `k6 threshold expression "metric:expr" (repeatable)`)
	// --print-script: emit generated script to stdout without running k6 (per D-11)
	cmd.Flags().BoolVar(&flags.PrintScript, "print-script", false, "print generated script to stdout without running it")

	return cmd, &flags
}

func main() {
	cmd, flags := buildRootCmd(os.Stdout, os.Stderr)

	// Use ExecuteC() not Execute() to avoid internal os.Exit calls (per D-02, RESEARCH.md §Anti-Patterns).
	_, err := cmd.ExecuteC()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := cmd.Flags().Args()
	os.Exit(run(args, *flags, os.Stdout, os.Stderr))
}
