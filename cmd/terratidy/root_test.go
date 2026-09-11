package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "terratidy", rootCmd.Use)
		assert.Equal(t, "TerraTidy - Terraform/Terragrunt Quality Platform", rootCmd.Short)
		assert.NotEmpty(t, rootCmd.Long)
	})

	t.Run("has config flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("config")
		assert.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})

	t.Run("has profile flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("profile")
		assert.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})

	t.Run("has format flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})

	t.Run("has changed flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("changed")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has no-recurse flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("no-recurse")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has severity-threshold flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("severity-threshold")
		assert.NotNil(t, flag)
		assert.Empty(t, flag.DefValue)
	})

	t.Run("color flag defaults to what the destination implies", func(t *testing.T) {
		if os.Getenv("FORCE_COLOR") != "" {
			t.Skip("FORCE_COLOR overrides the auto-detected default")
		}
		flag := rootCmd.PersistentFlags().Lookup("color")
		assert.NotNil(t, flag)
		// Not pinned to a literal: the default is auto-detected so that callers
		// bypassing PersistentPreRunE still match the destination. Under `go
		// test`, stdout is a pipe, so color must default off.
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has exclude flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("exclude")
		assert.NotNil(t, flag)
		assert.Equal(t, "[]", flag.DefValue)
	})

	t.Run("has absolute-paths flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("absolute-paths")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("silence usage is true", func(t *testing.T) {
		assert.True(t, rootCmd.SilenceUsage)
	})

	t.Run("silence errors is true", func(t *testing.T) {
		assert.True(t, rootCmd.SilenceErrors)
	})
}

func TestExecute(t *testing.T) {
	t.Run("execute returns no error for help", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--help"})
		err := Execute()
		assert.NoError(t, err)
	})
}
