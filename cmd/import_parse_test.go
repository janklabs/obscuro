package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseImportFile_table(t *testing.T) {
	cases := []struct {
		name       string
		content    string            // .env file contents; ignored when useMissing is true
		useMissing bool              // pass a path that does not exist
		wantErr    []string          // if non-empty, error must Contains each substring
		wantMap    map[string]string // when wantErr is empty, result must equal this
	}{
		{
			name:    "happy-path",
			content: "KEY=value\nANOTHER=secret\n",
			wantMap: map[string]string{"KEY": "value", "ANOTHER": "secret"},
		},
		{
			name:       "file-not-found",
			useMissing: true,
			wantErr:    []string{"parsing .env file"},
		},
		{
			name:    "invalid-key-lower",
			content: "foo=bar\n",
			wantErr: []string{"invalid key names", "foo"},
		},
		{
			name:    "invalid-key-digit",
			content: "1FOO=bar\n",
			wantErr: []string{"invalid key names", "1FOO"},
		},
		{
			name:    "empty-value",
			content: "EMPTY=\n",
			wantErr: []string{"empty values not allowed", "EMPTY"},
		},
		{
			name:    "both-classes",
			content: "foo=bar\nEMPTY=\nGOOD=x\n",
			wantErr: []string{"invalid key names", "empty values not allowed", "; "},
		},
		{
			name:    "comments-blanks",
			content: "# comment\n\nA=1\n",
			wantMap: map[string]string{"A": "1"},
		},
		{
			name:    "quoted-value",
			content: "A=\"hello world\"\n",
			wantMap: map[string]string{"A": "hello world"},
		},
		{
			name:    "var-interpolation",
			content: "B=x\nA=${B}\n",
			wantMap: map[string]string{"A": "x", "B": "x"},
		},
		{
			name:    "sorted-errors",
			content: "ZZZ=\nAAA=\n",
			wantErr: []string{"AAA, ZZZ"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			if tc.useMissing {
				path = filepath.Join(t.TempDir(), "does-not-exist.env")
			} else {
				path = filepath.Join(t.TempDir(), "input.env")
				if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			}

			got, err := parseImportFile(path)

			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("expected error containing %v, got nil (map=%v)", tc.wantErr, got)
				}
				msg := err.Error()
				for _, sub := range tc.wantErr {
					if !strings.Contains(msg, sub) {
						t.Fatalf("error %q missing substring %q", msg, sub)
					}
				}
				if got != nil {
					t.Fatalf("expected nil map on error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.wantMap) {
				t.Fatalf("map mismatch:\n want: %v\n  got: %v", tc.wantMap, got)
			}
		})
	}
}
