package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckColorDetection exercises the end-to-end --color wiring through
// rootCmd.Execute(), verifying the CLI resolves color the same way stdout
// consumers (pipes, redirected files, CI logs) actually behave: no escape
// codes unless a terminal, NO_COLOR, FORCE_COLOR, or an explicit flag says
// otherwise.
func TestCheckColorDetection(t *testing.T) {
	dir := t.TempDir()
	// A leading blank line inside a block reliably produces a style finding,
	// which is enough to exercise the colorized finding-line code path.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(
		"resource \"aws_instance\" \"b\" {\n\n  ami = \"x\"\n}\n",
	), 0o644))

	tests := []struct {
		name         string
		args         []string
		noColorEnv   string
		forceColor   string
		wantEscapes  bool
		descOfReason string
	}{
		{
			name:         "not a terminal, no flag, no env: no escapes",
			args:         []string{"check", dir},
			wantEscapes:  false,
			descOfReason: "stdout is a pipe in tests, so auto-detection disables color",
		},
		{
			name:         "NO_COLOR set: no escapes",
			args:         []string{"check", dir},
			noColorEnv:   "1",
			wantEscapes:  false,
			descOfReason: "NO_COLOR must win over any auto-detection default",
		},
		{
			name:         "FORCE_COLOR set through a pipe: escapes present",
			args:         []string{"check", dir},
			forceColor:   "1",
			wantEscapes:  true,
			descOfReason: "FORCE_COLOR must enable color even when stdout is not a terminal",
		},
		{
			name:         "explicit --color=false wins over FORCE_COLOR: no escapes",
			args:         []string{"check", "--color=false", dir},
			forceColor:   "1",
			wantEscapes:  false,
			descOfReason: "an explicit flag must take precedence over environment variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldColor := color
			t.Cleanup(func() {
				color = oldColor
				if f := rootCmd.PersistentFlags().Lookup("color"); f != nil {
					f.Changed = false
				}
				rootCmd.SetArgs(nil)
			})
			color = true
			if f := rootCmd.PersistentFlags().Lookup("color"); f != nil {
				f.Changed = false
			}

			if tt.noColorEnv != "" {
				t.Setenv("NO_COLOR", tt.noColorEnv)
			} else {
				t.Setenv("NO_COLOR", "")
			}
			if tt.forceColor != "" {
				t.Setenv("FORCE_COLOR", tt.forceColor)
			} else {
				t.Setenv("FORCE_COLOR", "")
			}

			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stdout = w
			rootCmd.SetArgs(tt.args)
			_ = rootCmd.Execute() // findings produce a non-nil error; only output matters here
			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			_ = r.Close()

			hasEscapes := strings.Contains(buf.String(), "\x1b[")
			require.Equal(t, tt.wantEscapes, hasEscapes, tt.descOfReason)
		})
	}
}
