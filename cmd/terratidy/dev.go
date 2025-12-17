package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	devWatch  string
	devTarget string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development mode with hot-reload",
	Long:  `Run in development mode with automatic reloading when rules change.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔄 Running in development mode...")
		// TODO: Implement dev mode logic
		return nil
	},
}

func init() {
	devCmd.Flags().StringVar(&devWatch, "watch", "rules/", "directory to watch for changes")
	devCmd.Flags().StringVar(&devTarget, "target", ".", "target directory to run checks against")
	rootCmd.AddCommand(devCmd)
}

