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

	t.Run("has json flag", func(t *testing.T) {
		flag := versionCmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestVersionCmdExecution(t *testing.T) {
	t.Run("version command runs", func(t *testing.T) {
		// Reset flags
		versionShort = false
		versionJSON = false

		rootCmd.SetArgs([]string{"version"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("version with short flag runs", func(t *testing.T) {
		versionShort = false
		versionJSON = false

		rootCmd.SetArgs([]string{"version", "--short"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("version with json flag runs", func(t *testing.T) {
		versionShort = false
		versionJSON = false

		rootCmd.SetArgs([]string{"version", "--json"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})
}

func TestVersionJSONOutput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	versionShort = false
	versionJSON = true

	err := versionCmd.RunE(versionCmd, nil)
	require.NoError(t, err)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Reset for other tests
	versionJSON = false

	// The output should be valid JSON
	output := buf.String()
	var result map[string]string
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "JSON output should be valid: %s", output)

	// Check expected fields exist
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "commit")
	assert.Contains(t, result, "date")
	assert.Contains(t, result, "goVersion")
	assert.Contains(t, result, "platform")
}
