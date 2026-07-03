package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

// authDoctorCmd is a diagnostic-only subcommand that prints one row per
// detected password backend along with verbose per-row diagnostics. It never
// mutates cfg.PasswordBackend, .obscuro/, or the XDG pwfile directory, and
// always exits 0 so scripts can capture the report without special-casing
// failure modes. Individual probes may still perform minimal transient side
// effects (e.g. a keychain trial Set that is immediately cleaned up).
var authDoctorCmd = &cobra.Command{
	Use:          "doctor",
	Short:        "Diagnose keychain and backend availability (diagnostic-only)",
	SilenceUsage: true,
	RunE:         doctorRunE,
}

// doctorRunE performs a best-effort backend scan and prints results to
// cmd.Stdout. Config load failure is non-fatal: we degrade to a zero-value
// Config so the keychain probe still runs and the user sees actionable
// output even before `obscuro init`.
func doctorRunE(cmd *cobra.Command, args []string) error {
	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault not initialized: %v\n", err)
		cfg = &store.Config{}
	}

	statuses := detectBackends(*cfg)

	fmt.Fprint(Stdout, "obscuro auth doctor — backend availability\n")
	for _, status := range statuses {
		mark := "✗"
		if status.Available {
			mark = "✓"
		}
		fmt.Fprintf(Stdout, "[%s] %s: %s — %s\n", mark, status.Kind, status.Name, status.Reason)
		for _, line := range status.Verbose {
			fmt.Fprintf(Stdout, "    %s\n", line)
		}
	}
	fmt.Fprintf(Stdout, "\nsee %s\n", docsURL)

	return nil
}

func init() {
	authCmd.AddCommand(authDoctorCmd)
}
