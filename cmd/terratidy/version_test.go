package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
