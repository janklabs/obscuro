package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var onConflict string

var importCmd = &cobra.Command{
	Use:          "import FILE",
	Short:        "Import secrets from a .env file",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch onConflict {
		case "skip", "overwrite", "fail":
		default:
			return fmt.Errorf("invalid --on-conflict %q: must be one of skip, overwrite, fail", onConflict)
		}

		key, err := authenticate()
		if err != nil {
			return err
		}

		parsed, err := parseImportFile(args[0])
		if err != nil {
			return err
		}

		existing, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		var newKeys, existingKeys []string
		for k := range parsed {
			if _, ok := existing[k]; ok {
				existingKeys = append(existingKeys, k)
			} else {
				newKeys = append(newKeys, k)
			}
		}
		sort.Strings(newKeys)
		sort.Strings(existingKeys)

		fmt.Fprintf(cmd.ErrOrStderr(), "Found %d new secrets and %d pre-existing secrets in %s.\n", len(newKeys), len(existingKeys), args[0])

		var action string
		if isatty.IsTerminal(os.Stdin.Fd()) {
			choice, err := runImportChoiceFn(len(newKeys), len(existingKeys))
			if err != nil {
				if errors.Is(err, ErrCancelled) {
					fmt.Fprint(cmd.ErrOrStderr(), "Import cancelled. No changes made.\n")
					return nil
				}
				return err
			}
			switch choice {
			case ImportChoiceCancel:
				fmt.Fprint(cmd.ErrOrStderr(), "Import cancelled. No changes made.\n")
				return nil
			case ImportChoiceNewOnly:
				action = "skip"
			case ImportChoiceOverwrite:
				action = "overwrite"
			default:
				return fmt.Errorf("unknown import choice %q", choice)
			}
		} else {
			if onConflict == "fail" && len(existingKeys) > 0 {
				return fmt.Errorf("existing keys would be overwritten: %s (rerun in a terminal or pass --on-conflict=skip|overwrite)", strings.Join(existingKeys, ", "))
			}
			if onConflict == "fail" {
				action = "skip"
			} else {
				action = onConflict
			}
		}

		merged := make(map[string]string, len(existing)+len(parsed))
		for k, v := range existing {
			merged[k] = v
		}
		var added, overwritten, skipped int
		for _, k := range append(newKeys, existingKeys...) {
			plaintext := parsed[k]
			enc, err := crypto.Encrypt(key, []byte(plaintext))
			if err != nil {
				return fmt.Errorf("encrypting %q: %w", k, err)
			}
			if _, exists := existing[k]; exists {
				if action == "skip" {
					skipped++
					continue
				}
				merged[k] = enc
				overwritten++
			} else {
				merged[k] = enc
				added++
			}
		}

		if err := store.SaveSecrets(merged); err != nil {
			return err
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "Import complete: %d added, %d overwritten, %d skipped.\n", added, overwritten, skipped)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&onConflict, "on-conflict", "fail", "how to handle pre-existing keys when non-interactive: skip, overwrite, or fail")
	rootCmd.AddCommand(importCmd)
}
