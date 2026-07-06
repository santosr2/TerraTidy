// Package main provides the dev command for TerraTidy.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/santosr2/TerraTidy/internal/engines/policy"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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
	printDevHeader()

	if !watchDirExists() {
		printWatchDirMissingHelp()
		return nil
	}

	targetFiles, err := getTargetFiles([]string{devTarget}, false)
	if err != nil {
		return sdk.NewInternalError(fmt.Errorf("finding target files: %w", err))
	}

	if len(targetFiles) == 0 {
		// User-correctable: wrong directory or missing files.
		return sdk.NewConfigError(fmt.Errorf("no HCL files found in target directory: %s", devTarget))
	}

	fmt.Printf("Found %d target file(s)\n\n", len(targetFiles))

	// Intentionally swallow errors: dev mode is interactive, transient failures
	// should not kill the session. User sees the error in output.
	if err := runDevCheck(targetFiles); err != nil {
		fmt.Printf("Initial check error: %v\n", err)
	}

	return runFileWatcher()
}

func printDevHeader() {
	fmt.Println("Starting development mode...")
	fmt.Printf("  Watching: %s\n", devWatch)
	fmt.Printf("  Target:   %s\n", devTarget)
	fmt.Println()
}

func watchDirExists() bool {
	_, err := os.Stat(devWatch)
	return !os.IsNotExist(err)
}

func printWatchDirMissingHelp() {
	fmt.Printf("Watch directory does not exist: %s\n\n", devWatch)
	fmt.Println("Create it with:")
	fmt.Printf("  mkdir -p %q\n\n", devWatch)
	fmt.Println("Or use terratidy init-rule to create a rule:")
	fmt.Println("  terratidy init-rule --name my-rule --type rego")
}

func runFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return sdk.NewInternalError(fmt.Errorf("creating watcher: %w", err))
	}
	defer func() { _ = watcher.Close() }()

	if err := setupWatchDirs(watcher); err != nil {
		return err // Already wrapped as ExitInternal by setupWatchDirs.
	}

	fmt.Println("Watching for changes... (Ctrl+C to stop)")
	fmt.Println()

	runWatchLoop(watcher)
	return nil
}

func setupWatchDirs(watcher *fsnotify.Watcher) error {
	err := filepath.Walk(devWatch, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return sdk.NewInternalError(fmt.Errorf("setting up watch: %w", err))
	}

	// Also watch target directory (non-fatal if fails)
	_ = filepath.Walk(devTarget, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	return nil
}

func runWatchLoop(watcher *fsnotify.Watcher) {
	var debounceTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			debounceTimer = handleWatchEvent(event, debounceTimer, debounceDelay)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

func handleWatchEvent(event fsnotify.Event, timer *time.Timer, delay time.Duration) *time.Timer {
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return timer
	}

	if !isRelevantFile(event.Name) {
		return timer
	}

	if timer != nil {
		timer.Stop()
	}

	return time.AfterFunc(delay, func() {
		fmt.Printf("\n[%s] File changed: %s\n\n", time.Now().Format("15:04:05"), event.Name)

		refreshedFiles, err := getTargetFiles([]string{devTarget}, false)
		if err != nil {
			fmt.Printf("Error refreshing files: %v\n", err)
			return
		}

		if err := runDevCheck(refreshedFiles); err != nil {
			fmt.Printf("Check error: %v\n", err)
		}
	})
}

func isRelevantFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".rego" || ext == ".tf" || ext == ".hcl" || ext == ".tfvars"
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
	errors, warnings, info := countBySeverity(findings)

	// Display findings
	for i := range findings {
		icon := devSeverityIcon(findings[i].Severity)
		fmt.Printf("  [%s] %s\n", icon, findings[i].Rule)
		fmt.Printf("      %s\n", findings[i].Message)
		if findings[i].File != "" {
			fmt.Printf("      File: %s\n", findings[i].File)
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
			if !strings.HasSuffix(path, "_test.rego") {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, err
}

// devSeverityIcon returns a single-character icon for the given severity.
// Lowercase 'i' visually de-emphasizes info-level findings.
func devSeverityIcon(severity sdk.Severity) string {
	switch severity {
	case sdk.SeverityError:
		return "E"
	case sdk.SeverityWarning:
		return "W"
	case sdk.SeverityInfo:
		return "i"
	default:
		return "?"
	}
}
