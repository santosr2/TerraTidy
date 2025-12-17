package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	fmtCheck bool
	fmtDiff  bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format Terraform and Terragrunt files",
	Long:  `Format .tf and .hcl files using the HCL formatter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🎨 Formatting files...")
		// TODO: Implement fmt logic
		return nil
	},
}

func init() {
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "check if files are formatted")
	fmtCmd.Flags().BoolVar(&fmtDiff, "diff", false, "show diff of formatting changes")
	rootCmd.AddCommand(fmtCmd)
}

