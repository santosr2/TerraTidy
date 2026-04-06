// Package main provides CLI helpers for TerraTidy commands.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santosr2/TerraTidy/internal/cache"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/output"
	"github.com/santosr2/TerraTidy/internal/vcs"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// getTargetFiles returns the list of files to process based on the provided paths
// and global flags. When --changed is set, it uses VCS to detect changed files.
func getTargetFiles(paths []string, changedOnly bool) ([]string, error) {
	if changedOnly {
		return getChangedFiles(paths)
	}
	return findHCLFilesFromPaths(paths)
}

// getChangedFiles uses VCS to get only changed Terraform/HCL files.
// If paths are provided, it filters the changed files to only those within the paths.
func getChangedFiles(filterPaths []string) ([]string, error) {
	git := vcs.NewGit(".")

	// Check if we're in a git repo
	if !git.IsGitRepo() {
		return nil, fmt.Errorf("not a git repository; --changed requires git")
	}

	// Get all changed Terraform files
	changedFiles, err := git.GetAllChangedTerraformFiles()
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}

	// If no filter paths provided, return all changed files
	if len(filterPaths) == 0 || (len(filterPaths) == 1 && filterPaths[0] == ".") {
		return vcs.FilterExisting(changedFiles), nil
	}

	// Filter changed files to only those within the specified paths
	var filteredFiles []string
	for _, file := range changedFiles {
		for _, filterPath := range filterPaths {
			absFilterPath, err := filepath.Abs(filterPath)
			if err != nil {
				continue
			}

			// Check if the file is within the filter path
			if isPathWithin(file, absFilterPath) {
				filteredFiles = append(filteredFiles, file)
				break
			}
		}
	}

	return vcs.FilterExisting(filteredFiles), nil
}

// isPathWithin checks if a file path is within a directory path.
func isPathWithin(filePath, dirPath string) bool {
	// Clean and normalize paths
	filePath = filepath.Clean(filePath)
	dirPath = filepath.Clean(dirPath)

	// Check if file starts with directory path
	if strings.HasPrefix(filePath, dirPath) {
		// Make sure it's actually within (not just a prefix match)
		remainder := strings.TrimPrefix(filePath, dirPath)
		return remainder == "" || strings.HasPrefix(remainder, string(filepath.Separator))
	}
	return false
}

// findHCLFilesFromPaths is a helper that handles default paths and delegates to findHCLFiles.
func findHCLFilesFromPaths(paths []string) ([]string, error) {
	targetPaths := paths
	if len(targetPaths) == 0 {
		targetPaths = []string{"."}
	}
	return findHCLFiles(targetPaths)
}

// findHCLFiles recursively finds all .tf and .hcl files in the given paths.
func findHCLFiles(paths []string) ([]string, error) {
	collector := newFileCollector()
	for _, path := range paths {
		if err := collector.collectPath(path); err != nil {
			return nil, err
		}
	}
	return collector.files, nil
}

// fileCollector collects unique HCL files.
type fileCollector struct {
	files []string
	seen  map[string]bool
}

func newFileCollector() *fileCollector {
	return &fileCollector{seen: make(map[string]bool)}
}

func (c *fileCollector) collectPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		return c.walkDirectory(path)
	}
	c.addFileIfHCL(path)
	return nil
}

func (c *fileCollector) walkDirectory(dir string) error {
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && shouldSkipDir(p, info.Name()) {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			c.addFileIfHCL(p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %s: %w", dir, err)
	}
	return nil
}

func (c *fileCollector) addFileIfHCL(path string) {
	if !isHCLFile(path) || c.seen[path] {
		return
	}
	absPath := toAbsPath(path)
	if c.seen[absPath] {
		return
	}
	c.files = append(c.files, absPath)
	c.seen[absPath] = true
}

func toAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// shouldSkipDir returns true if the directory should be skipped during traversal.
func shouldSkipDir(_ string, name string) bool {
	// Skip hidden directories
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	// Skip common non-terraform directories
	skipDirs := map[string]bool{
		"node_modules":      true,
		"vendor":            true,
		".terraform":        true,
		".terragrunt-cache": true,
		"__pycache__":       true,
	}
	return skipDirs[name]
}

// isHCLFile checks if a file has a Terraform/HCL extension.
func isHCLFile(path string) bool {
	return sdk.IsHCLFile(path)
}

// formatFileCount returns a human-readable file count string.
func formatFileCount(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

// countBySeverity counts findings by severity level.
func countBySeverity(findings []sdk.Finding) (errors, warnings, info int) {
	for _, finding := range findings {
		switch finding.Severity {
		case sdk.SeverityError:
			errors++
		case sdk.SeverityWarning:
			warnings++
		case sdk.SeverityInfo:
			info++
		}
	}
	return
}

// printNoFilesMessage prints the appropriate "no files found" message
// based on whether the --changed flag was used.
func printNoFilesMessage() {
	if changed {
		fmt.Println("No changed HCL files found")
	} else {
		fmt.Println("No HCL files found")
	}
}

// outputResults formats and prints findings, then returns an ExitError if
// there are errors. This is the shared implementation used by lint, style,
// and policy commands.
func outputResults(findings []sdk.Finding, label string) error {
	formatter, err := output.GetFormatterWithColor(format, true, version, color)
	if err != nil {
		return fmt.Errorf("getting formatter: %w", err)
	}

	if err := formatter.Format(findings, os.Stdout); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	// For text format, add summary
	if format == "" || format == "text" {
		errors, warnings, info := countBySeverity(findings)

		if len(findings) > 0 {
			fmt.Println()
			fmt.Println("---")
			fmt.Printf("%s: %d error(s), %d warning(s), %d info\n", label, errors, warnings, info)
		}

		if errors > 0 {
			return &sdk.ExitError{Code: 1}
		}
	} else {
		// Return exit error if there are errors (for structured output)
		for _, finding := range findings {
			if finding.Severity == sdk.SeverityError {
				return &sdk.ExitError{Code: 1}
			}
		}
	}

	return nil
}

// loadConfig loads the configuration from the config file and applies the profile if specified.
// It uses the global cfgFile and profile variables from root.go.
// Returns the default config if no config file is found.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Apply profile if specified
	if profile != "" {
		if err := cfg.ApplyProfile(profile); err != nil {
			return nil, fmt.Errorf("applying profile %q: %w", profile, err)
		}
	}

	// Configure cache from config (uses defaults for unset values)
	configureCacheFromConfig(cfg)

	return cfg, nil
}

// configureCacheFromConfig configures the global cache based on config settings.
// Uses cache defaults (5m TTL, 1000 entries) for any unset values.
func configureCacheFromConfig(cfg *config.Config) {
	if cfg == nil || !cfg.Cache.IsConfigured() {
		return // Use default cache settings
	}

	opts := cache.DefaultOptions()
	if cfg.Cache.MaxAge != 0 {
		opts.MaxAge = cfg.Cache.MaxAge.Duration()
	}
	if cfg.Cache.MaxSize != 0 {
		opts.MaxSize = cfg.Cache.MaxSize
	}
	if cfg.Cache.Disabled {
		opts.Disabled = true
	}
	cache.ConfigureDefault(opts)
}

// filterFindingsBySeverity filters findings based on the severity threshold.
// Only findings with severity >= threshold are returned.
// If threshold is empty, all findings are returned.
func filterFindingsBySeverity(findings []sdk.Finding, threshold string) []sdk.Finding {
	if threshold == "" {
		return findings
	}

	thresholdLevel := sdk.Severity(threshold).Level()

	var filtered []sdk.Finding
	for _, f := range findings {
		if f.Severity.Level() >= thresholdLevel {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// getEffectiveSeverityThreshold returns the severity threshold to use.
// CLI flag takes precedence over config file setting.
func getEffectiveSeverityThreshold(cfg *config.Config) string {
	if severityThreshold != "" {
		return severityThreshold
	}
	if cfg != nil {
		return cfg.SeverityThreshold
	}
	return ""
}

// getEffectiveParallel returns whether parallel execution should be used.
// CLI flag takes precedence over config file setting.
func getEffectiveParallel(cfg *config.Config, cliParallel bool) bool {
	if cliParallel {
		return true
	}
	if cfg != nil {
		return cfg.Parallel
	}
	return false
}

// shouldFailFast returns whether fail-fast mode is enabled from config.
func shouldFailFast(cfg *config.Config) bool {
	if cfg != nil {
		return cfg.FailFast
	}
	return false
}

// isEngineEnabled checks if an engine is enabled in the config.
// Returns true by default if config is nil or engine is not explicitly disabled.
func isEngineEnabled(cfg *config.Config, engine string) bool {
	if cfg == nil {
		return true
	}

	switch engine {
	case "fmt":
		return cfg.Engines.Fmt.IsEnabled()
	case "style":
		return cfg.Engines.Style.IsEnabled()
	case "lint":
		return cfg.Engines.Lint.IsEnabled()
	case "policy":
		return cfg.Engines.Policy.IsEnabled()
	default:
		return true
	}
}
