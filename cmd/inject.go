package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Replace __KEY__ placeholders in stdin with decrypted secrets",
	Long: `Reads stdin, replaces all __KEY__ placeholders with their decrypted
secret values, and writes the result to stdout.

Designed for use as a Helm post-renderer:
  helm install myrelease ./chart --post-renderer obscuro --post-renderer-args inject`,
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := authenticate()
		if err != nil {
			return err
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		// Decrypt all secrets upfront
		decrypted := make(map[string]string, len(secrets))
		for name, enc := range secrets {
			plain, err := crypto.Decrypt(key, enc)
			if err != nil {
				return fmt.Errorf("decrypting '%s': %w", name, err)
			}
			decrypted[name] = string(plain)
		}

		// Read all stdin
		input, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}

		// Replace placeholders and track which were injected
		output := string(input)
		var injected []string
		for name, value := range decrypted {
			placeholder := "__" + name + "__"
			replaced := strings.ReplaceAll(output, placeholder, value)
			if replaced != output {
				injected = append(injected, name)
				output = replaced
			}
		}

		// Print summary to terminal
		if tty, err := openTTY(); err == nil {
			if len(injected) > 0 {
				sort.Strings(injected)
				fmt.Fprintf(tty, "Injected: %s\n", strings.Join(injected, ", "))
			} else {
				fmt.Fprintln(tty, "No secrets injected.")
			}
			tty.Close()
		}

		cmd.Print(output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(injectCmd)
}
