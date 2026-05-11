package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var injectStrict bool

var placeholderRe = regexp.MustCompile(`__([A-Z][A-Z0-9_]*)__`)

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Replace __KEY__ placeholders in stdin with decrypted secrets",
	Long: `Reads stdin, replaces all __KEY__ placeholders with their decrypted
secret values, and writes the result to stdout.

Designed for use as a Helm post-renderer:
  helm install myrelease ./chart --post-renderer obscuro --post-renderer-args inject

By default, unresolved placeholders are left as-is and reported to stderr.
Use --strict (or set OBSCURO_INJECT_STRICT=1) to fail when placeholders cannot
be resolved; in strict mode no output is written to stdout on failure.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		strict := injectStrict || os.Getenv("OBSCURO_INJECT_STRICT") == "1"

		input, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}

		referencedSet := map[string]struct{}{}
		for _, m := range placeholderRe.FindAllSubmatch(input, -1) {
			referencedSet[string(m[1])] = struct{}{}
		}

		key, err := authenticate()
		if err != nil {
			return err
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		decrypted := make(map[string]string, len(referencedSet))
		var unresolved []string
		for name := range referencedSet {
			enc, ok := secrets[name]
			if !ok {
				unresolved = append(unresolved, name)
				continue
			}
			plain, err := crypto.Decrypt(key, enc)
			if err != nil {
				return fmt.Errorf("decrypting '%s': %w", name, err)
			}
			decrypted[name] = string(plain)
		}
		sort.Strings(unresolved)

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
		sort.Strings(injected)

		if len(unresolved) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: unresolved placeholders: %s\n", strings.Join(unresolved, ", "))
		}

		if strict && len(unresolved) > 0 {
			return fmt.Errorf("unresolved placeholders: %s", strings.Join(unresolved, ", "))
		}

		if len(injected) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Injected: %s\n", strings.Join(injected, ", "))
		} else if len(unresolved) == 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "No secrets injected.")
		}

		fmt.Fprint(Stdout, output)
		return nil
	},
}

func init() {
	injectCmd.Flags().BoolVar(&injectStrict, "strict", false, "fail on unresolved __KEY__ placeholders (default: warn-only)")
	rootCmd.AddCommand(injectCmd)
}
