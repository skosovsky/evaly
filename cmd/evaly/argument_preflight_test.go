package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIRejectsCommandSpecificArgumentsBeforeStoreCreation(t *testing.T) {
	for _, test := range []struct {
		name, command string
		arguments     []string
	}{
		{"compare_behavior", "compare", []string{"--behavior", "good"}},
		{"compare_id", "compare", []string{"--id", "unused"}},
		{"compare_seed", "compare", []string{"--seed", "0"}},
		{"compare_case", "compare", []string{"--case", "unused"}},
		{"fixture_policy", "fixture", []string{"--policy", "missing.json"}},
		{"fixture_baseline", "fixture", []string{"--baseline", "unused"}},
		{"fixture_candidate", "fixture", []string{"--candidate", "unused"}},
		{"unknown_command", "typo", nil},
		{"unknown_flag", "fixture", []string{"--unknown", "value"}},
		{"positional_argument", "fixture", []string{"surprise"}},
		{"missing_fixture_id", "fixture", nil},
		{"invalid_fixture_id", "fixture", []string{"--id", "../invalid"}},
		{"invalid_fixture_behavior", "fixture", []string{"--id", "valid", "--behavior", "unknown"}},
		{"invalid_fixture_seed", "fixture", []string{"--id", "valid", "--seed", "9223372036854775808"}},
		{"missing_compare_policy", "compare", []string{"--baseline", "base", "--candidate", "candidate"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Arrange: a path that does not exist; argument validation must not create it.
			directory := filepath.Join(t.TempDir(), "not-created")
			arguments := append([]string{test.command, "--store", directory}, test.arguments...)
			var output, diagnostics bytes.Buffer
			// Act.
			status := run(context.Background(), arguments, &output, &diagnostics)
			// Assert: arguments fail before OpenFileStore or fixture work.
			if status != invalidExitCode || output.Len() != 0 || diagnostics.Len() == 0 {
				t.Fatal(status, output.String(), diagnostics.String())
			}
			if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("argument validation created store", err)
			}
		})
	}
}
