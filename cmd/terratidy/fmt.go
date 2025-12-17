package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	fmtengine "github.com/santosr2/terratidy/internal/engines/fmt"
	"github.com/spf13/cobra"
)

var (
	fmtCheck bool
	fmtDiff  bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [paths...]",
	Short: "Format Terraform and Terragrunt files",
	Long:  `Format .tf and .hcl files using the HCL formatter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine paths to format
		targetPaths := args
		if len(targetPaths) == 0 {
			targetPaths = []string{"."}
		}

		// Find all HCL files
		files, err := findHCLFiles(targetPaths)
		if err != nil {
			return fmt.Errorf("finding files: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("No HCL files found")
			return nil
		}

		// Create formatter engine
		engine := fmtengine.New(&fmtengine.Config{
			Check: fmtCheck,
			Diff:  fmtDiff,
		})

		// Run formatter
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("formatting files: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("✓ All files are properly formatted")
			return nil
		}

		for _, finding := range findings {
			if finding.Rule == "fmt.needs-formatting" {
				fmt.Printf("✗ %s: needs formatting\n", finding.File)
			} else if finding.Rule == "fmt.formatted" {
				fmt.Printf("✓ %s: formatted\n", finding.File)
			}
		}

		// In check mode, return error if any file needs formatting
		if fmtCheck {
			needsFormatting := 0
			for _, f := range findings {
				if f.Rule == "fmt.needs-formatting" {
					needsFormatting++
				}
			}
			if needsFormatting > 0 {
				return fmt.Errorf("%d file(s) need formatting", needsFormatting)
			}
		}

		return nil
	},
}

func init() {
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "check if files are formatted")
	fmtCmd.Flags().BoolVar(&fmtDiff, "diff", false, "show diff of formatting changes")
	rootCmd.AddCommand(fmtCmd)
}

// findHCLFiles recursively finds all .tf and .hcl files in the given paths
func findHCLFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isHCLFile(p) && !seen[p] {
					files = append(files, p)
					seen[p] = true
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", path, err)
			}
		} else if isHCLFile(path) && !seen[path] {
			files = append(files, path)
			seen[path] = true
		}
	}

	return files, nil
}

// isHCLFile checks if a file has .tf or .hcl extension
func isHCLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".tf" || ext == ".hcl"
}
