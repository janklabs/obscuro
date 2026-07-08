package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

var importKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// parseImportFile reads a .env file at path, validates key names and values,
// and returns the parsed map. It uses godotenv.Read (never Load) so no
// values are written to os.Environ.
func parseImportFile(path string) (map[string]string, error) {
	parsed, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("parsing .env file: %w", err)
	}

	var badKeys, emptyKeys []string
	for k, v := range parsed {
		if !importKeyRe.MatchString(k) {
			badKeys = append(badKeys, k)
		} else if len(v) == 0 {
			emptyKeys = append(emptyKeys, k)
		}
	}
	sort.Strings(badKeys)
	sort.Strings(emptyKeys)

	switch {
	case len(badKeys) > 0 && len(emptyKeys) > 0:
		return nil, fmt.Errorf(
			"invalid key names (must match [A-Z][A-Z0-9_]*): %s; empty values not allowed for keys: %s",
			strings.Join(badKeys, ", "),
			strings.Join(emptyKeys, ", "),
		)
	case len(badKeys) > 0:
		return nil, fmt.Errorf(
			"invalid key names (must match [A-Z][A-Z0-9_]*): %s",
			strings.Join(badKeys, ", "),
		)
	case len(emptyKeys) > 0:
		return nil, fmt.Errorf(
			"empty values not allowed for keys: %s",
			strings.Join(emptyKeys, ", "),
		)
	}

	return parsed, nil
}
