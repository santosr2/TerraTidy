// Package main provides the dev command for TerraTidy.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/santosr2/terratidy/internal/engines/policy"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	devWatch  string
	devTarget string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development mode with file watching",
	Long: `Run in development mode with automatic re-evaluation when files change.

This mode is useful when developing custom rules. It watches for changes
in rule files and automatically re-runs checks against target files.`,
	Example: `  # Watch rules directory and check current directory
  terratidy dev

  # Watch specific directory
  terratidy dev --watch ./policies

  # Check specific target directory
  terratidy dev --target ./modules`,
	RunE: runDev,
}

func init() {
	devCmd.Flags().StringVar(&devWatch, "watch", "policies/", "directory to watch for changes")
	devCmd.Flags().StringVar(&devTarget, "target", ".", "target directory to run checks against")
	rootCmd.AddCommand(devCmd)
}

func runDev(_ *cobra.Command, _ []string) error {
	fmt.Println("Starting development mode...")
	fmt.Printf("  Watching: %s\n", devWatch)
	fmt.Printf("  Target:   %s\n", devTarget)
	fmt.Println()

	// Check if watch directory exists
	if _, err := os.Stat(devWatch); os.IsNotExist(err) {
		fmt.Printf("Watch directory does not exist: %s\n", devWatch)
		fmt.Println()
		fmt.Println("Create it with:")
		fmt.Printf("  mkdir -p %s\n", devWatch)
		fmt.Println()
		fmt.Println("Or use terratidy init-rule to create a rule:")
		fmt.Println("  terratidy init-rule --name my-rule --type rego")
		return nil
	}

	// Find target files
	targetFiles, err := getTargetFiles([]string{devTarget}, false)
	if err != nil {
		return fmt.Errorf("finding target files: %w", err)
	}

	if len(targetFiles) == 0 {
		return fmt.Errorf("no HCL files found in target directory: %s", devTarget)
	}

	fmt.Printf("Found %d target file(s)\n", len(targetFiles))
	fmt.Println()

	// Run initial check
	if err := runDevCheck(targetFiles); err != nil {
		fmt.Printf("Initial check error: %v\n", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	// Watch the directory and subdirectories
	err = filepath.Walk(devWatch, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("setting up watch: %w", err)
	}

	// Also watch target directory for changes
	err = filepath.Walk(devTarget, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		// Non-fatal if target watch fails
		fmt.Printf("Warning: could not watch target directory: %v\n", err)
	}

	fmt.Println("Watching for changes... (Ctrl+C to stop)")
	fmt.Println()

	// Debounce timer
	var debounceTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only react to write and create events
			if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
				continue
			}

			// Check if it's a relevant file
			ext := filepath.Ext(event.Name)
			if ext != ".rego" && ext != ".tf" && ext != ".hcl" && ext != ".tfvars" {
				continue
			}

			// Debounce multiple rapid events
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				fmt.Printf("\n[%s] File changed: %s\n\n", time.Now().Format("15:04:05"), event.Name)

				// Refresh target files in case new files were added
				refreshedFiles, err := getTargetFiles([]string{devTarget}, false)
				if err != nil {
					fmt.Printf("Error refreshing files: %v\n", err)
					return
				}

				if err := runDevCheck(refreshedFiles); err != nil {
					fmt.Printf("Check error: %v\n", err)
				}
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

func runDevCheck(targetFiles []string) error {
	// Find policy files
	policyFiles, err := findPolicyFiles(devWatch)
	if err != nil {
		return fmt.Errorf("finding policy files: %w", err)
	}

	if len(policyFiles) == 0 {
		fmt.Println("No policy files found in watch directory")
		fmt.Println()
		fmt.Println("Create a policy with:")
		fmt.Println("  terratidy init-rule --name my-rule --type rego")
		return nil
	}

	fmt.Printf("Running %d policy(ies) against %d file(s)...\n\n", len(policyFiles), len(targetFiles))

	// Create policy engine
	engine := policy.New(&policy.Config{
		PolicyFiles: policyFiles,
	})

	// Run checks
	ctx := context.Background()
	findings, err := engine.Run(ctx, targetFiles)
	if err != nil {
		return fmt.Errorf("running checks: %w", err)
	}

	// Display results
	if len(findings) == 0 {
		fmt.Println("  No issues found")
		fmt.Println()
		return nil
	}

	// Count by severity
	var errors, warnings, info int
	for _, f := range findings {
		switch f.Severity {
		case sdk.SeverityError:
			errors++
		case sdk.SeverityWarning:
			warnings++
		case sdk.SeverityInfo:
			info++
		}
	}

	// Display findings
	for _, finding := range findings {
		icon := "i"
		switch finding.Severity {
		case sdk.SeverityError:
			icon = "!"
		case sdk.SeverityWarning:
			icon = "!"
		}

		fmt.Printf("  [%s] %s\n", icon, finding.Rule)
		fmt.Printf("      %s\n", finding.Message)
		if finding.File != "" {
			fmt.Printf("      File: %s\n", finding.File)
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("---\n")
	fmt.Printf("Summary: %d error(s), %d warning(s), %d info\n", errors, warnings, info)
	fmt.Println()

	return nil
}

func findPolicyFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".rego" {
			// Skip test files
			if filepath.Ext(filepath.Base(path[:len(path)-5])) != "_test" {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, err
}
