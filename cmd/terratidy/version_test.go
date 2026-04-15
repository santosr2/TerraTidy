package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "version", versionCmd.Use)
		assert.Equal(t, "Print version information", versionCmd.Short)
		assert.NotEmpty(t, versionCmd.Long)
	})

	t.Run("has short flag", func(t *testing.T) {
		flag := versionCmd.Flags().Lookup("short")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("uses global format flag", func(t *testing.T) {
		// Version command should use the global --format flag from root
		flag := rootCmd.PersistentFlags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})
}

func TestVersionCmdExecution(t *testing.T) {
	t.Run("version command runs", func(t *testing.T) {
		versionShort = false
		format = "text"
		t.Cleanup(func() { format = "text"; versionShort = false })

		rootCmd.SetArgs([]string{"version"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("version with short flag runs", func(t *testing.T) {
		versionShort = false
		format = "text"
		t.Cleanup(func() { format = "text"; versionShort = false })

		rootCmd.SetArgs([]string{"version", "--short"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("version with format json runs", func(t *testing.T) {
		versionShort = false
		format = "text"
		t.Cleanup(func() { format = "text"; versionShort = false })

		rootCmd.SetArgs([]string{"version", "--format", "json"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("version with format json-compact runs", func(t *testing.T) {
		versionShort = false
		format = "text"
		t.Cleanup(func() { format = "text"; versionShort = false })

		rootCmd.SetArgs([]string{"version", "--format", "json-compact"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		versionShort = false
		format = "sarif"
		t.Cleanup(func() { format = "text"; versionShort = false })

		err := versionCmd.RunE(versionCmd, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
		assert.Contains(t, err.Error(), "sarif")
	})

	t.Run("short and format json conflict returns error", func(t *testing.T) {
		versionShort = true
		format = "json"
		t.Cleanup(func() { format = "text"; versionShort = false })

		err := versionCmd.RunE(versionCmd, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--short cannot be combined")
	})
}

func TestVersionJSONOutput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	versionShort = false
	format = "json"
	t.Cleanup(func() {
		w.Close()
		os.Stdout = oldStdout
		format = "text"
		versionShort = false
	})

	err := versionCmd.RunE(versionCmd, nil)
	w.Close()
	os.Stdout = oldStdout
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// The output should be valid JSON
	output := buf.String()
	var result map[string]string
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "JSON output should be valid: %s", output)

	// Check expected fields exist (snake_case for consistency)
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "commit")
	assert.Contains(t, result, "date")
	assert.Contains(t, result, "go_version")
	assert.Contains(t, result, "platform")
}

func TestVersionJSONCompactOutput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	versionShort = false
	format = "json-compact"
	t.Cleanup(func() {
		w.Close()
		os.Stdout = oldStdout
		format = "text"
		versionShort = false
	})

	err := versionCmd.RunE(versionCmd, nil)
	w.Close()
	os.Stdout = oldStdout
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// The output should be valid JSON on a single line
	output := buf.String()
	var result map[string]string
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "JSON output should be valid: %s", output)

	// Check expected fields exist (snake_case for consistency)
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "commit")
	assert.Contains(t, result, "date")
	assert.Contains(t, result, "go_version")
	assert.Contains(t, result, "platform")

	// json-compact should not have newlines in the JSON itself
	assert.NotContains(t, output[:len(output)-1], "\n", "json-compact should be single line")
}
