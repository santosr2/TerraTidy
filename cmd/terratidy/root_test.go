package main

import (
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

	t.Run("has color flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("color")
		assert.NotNil(t, flag)
		assert.Equal(t, "true", flag.DefValue)
	})

	t.Run("has exclude flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("exclude")
		assert.NotNil(t, flag)
		assert.Equal(t, "[]", flag.DefValue)
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
