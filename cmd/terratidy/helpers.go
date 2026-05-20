// Package main provides CLI helpers for TerraTidy commands.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/santosr2/TerraTidy/internal/cache"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/output"
	"github.com/santosr2/TerraTidy/internal/vcs"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// skipDirs is a set of directory names to skip during traversal.
// Declared at package level to avoid allocation on every shouldSkipDir call.
var skipDirs = map[string]bool{
	"node_modules":      true,
	"vendor":            true,
	".terraform":        true,
	".terragrunt-cache": true,
	"__pycache__":       true,
}

// getTargetFiles returns the list of files to process based on the provided paths
// and global flags. When --changed is set, it uses VCS to detect changed files.
// Exclude patterns from both config and CLI flags are applied.
// When --no-recurse is set, only files in the specified directories (not subdirs) are scanned.
func getTargetFiles(paths []string, changedOnly bool) ([]string, error) {
	var files []string
	var err error

	if changedOnly {
		files, err = getChangedFiles(paths, !noRecurse)
	} else {
		files, err = findHCLFilesFromPaths(paths, !noRecurse)
	}
	if err != nil {
		return nil, err
	}

	// Apply exclude patterns (CLI patterns override/combine with config)
	return filterExcludedFiles(files, excludePatterns), nil
}

// getTargetFilesWithExcludes returns the list of files with explicit exclude patterns.
// Used by commands that need to pass config-based excludes.
// The cfg parameter is used to determine the recursive setting (CLI flag takes precedence).
func getTargetFilesWithExcludes(paths []string, changedOnly bool, excludes []string, cfg *config.Config) ([]string, error) {
	var files []string
	var err error

	recursive := getEffectiveRecursive(cfg)
	if changedOnly {
		files, err = getChangedFiles(paths, recursive)
	} else {
		files, err = findHCLFilesFromPaths(paths, recursive)
	}
	if err != nil {
		return nil, err
	}

	// Combine CLI and config excludes without mutating originals
	allExcludes := make([]string, 0, len(excludePatterns)+len(excludes))
	allExcludes = append(allExcludes, excludePatterns...)
	allExcludes = append(allExcludes, excludes...)
	return filterExcludedFiles(files, allExcludes), nil
}

// filterExcludedFiles removes files matching any of the exclude patterns.
func filterExcludedFiles(files []string, patterns []string) []string {
	if len(patterns) == 0 {
		return files
	}

	var result []string
	for _, file := range files {
		if !matchesAnyPattern(file, patterns) {
			result = append(result, file)
		}
	}
	return result
}

// matchesAnyPattern returns true if the file matches any of the patterns.
func matchesAnyPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchGlobPattern(file, pattern) {
			return true
		}
	}
	return false
}

// matchGlobPattern matches a file path against a glob pattern.
// Supports standard glob patterns including ** for recursive matching.
func matchGlobPattern(filePath, pattern string) bool {
	// Normalize paths to use forward slashes for consistent matching
	filePath = filepath.ToSlash(filePath)
	pattern = filepath.ToSlash(pattern)

	// Handle ** (recursive) patterns
	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(filePath, pattern)
	}

	// For simple patterns, use filepath.Match
	// Try matching against both the full path and just the filename
	if matched, _ := filepath.Match(pattern, filePath); matched {
		return true
	}
	if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
		return true
	}

	// Try matching against path segments (for patterns like "vendor")
	segments := strings.Split(filePath, "/")
	for _, seg := range segments {
		if matched, _ := filepath.Match(pattern, seg); matched {
			return true
		}
	}

	return false
}

// matchDoubleStarPattern handles glob patterns containing **.
// ** matches zero or more directory levels.
func matchDoubleStarPattern(filePath, pattern string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	if len(parts) == 1 {
		// No ** found, use simple match
		matched, _ := filepath.Match(pattern, filePath)
		return matched
	}

	// Check prefix (part before first **)
	prefix := parts[0]
	if prefix != "" && prefix != "/" {
		prefix = strings.TrimSuffix(prefix, "/")
		// For patterns like "vendor/**", check if the prefix appears anywhere in the path
		// This handles both relative and absolute paths
		if !strings.HasPrefix(filePath, prefix) && !strings.Contains(filePath, "/"+prefix) {
			return false
		}
	}

	// Check suffix (part after last **)
	suffix := parts[len(parts)-1]
	if suffix != "" && suffix != "/" {
		suffix = strings.TrimPrefix(suffix, "/")
		// For suffix patterns like "*.tf", match against the filename
		if strings.HasPrefix(suffix, "*") {
			matched, _ := filepath.Match(suffix, filepath.Base(filePath))
			if !matched {
				return false
			}
		} else if !strings.HasSuffix(filePath, suffix) {
			return false
		}
	}

	// For patterns like "**/vendor/**", check if vendor is in the path
	if len(parts) > 2 {
		for i := 1; i < len(parts)-1; i++ {
			middle := strings.Trim(parts[i], "/")
			if middle != "" && !strings.Contains(filePath, middle) {
				return false
			}
		}
	}

	return true
}

// getChangedFiles uses VCS to get only changed Terraform/HCL files.
// If paths are provided, it filters the changed files to only those within the paths.
// When recursive is false, only changed files directly in the specified directories are returned.
func getChangedFiles(filterPaths []string, recursive bool) ([]string, error) {
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

	// If no filter paths provided, handle based on recursive setting
	if len(filterPaths) == 0 || (len(filterPaths) == 1 && filterPaths[0] == ".") {
		if recursive {
			return vcs.FilterExisting(changedFiles), nil
		}
		// Non-recursive: only return files directly in current directory
		cwd, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		// Resolve symlinks and Windows 8.3 short names to canonical form
		if evalCwd, err := filepath.EvalSymlinks(cwd); err == nil {
			cwd = evalCwd
		}
		var topLevelFiles []string
		for _, file := range changedFiles {
			if isFileDirectlyIn(file, cwd) {
				topLevelFiles = append(topLevelFiles, file)
			}
		}
		return vcs.FilterExisting(topLevelFiles), nil
	}

	// Filter changed files to only those within the specified paths
	var filteredFiles []string
	for _, file := range changedFiles {
		for _, filterPath := range filterPaths {
			absFilterPath, err := filepath.Abs(filterPath)
			if err != nil {
				continue
			}
			// Resolve symlinks and Windows 8.3 short names to canonical form
			if evalPath, err := filepath.EvalSymlinks(absFilterPath); err == nil {
				absFilterPath = evalPath
			}

			// Check if the file is within the filter path (recursive or direct)
			if recursive {
				if isPathWithin(file, absFilterPath) {
					filteredFiles = append(filteredFiles, file)
					break
				}
			} else {
				if isFileDirectlyIn(file, absFilterPath) {
					filteredFiles = append(filteredFiles, file)
					break
				}
			}
		}
	}

	return vcs.FilterExisting(filteredFiles), nil
}

// pathsEqual compares two paths for equality.
// On Windows, comparison is case-insensitive due to drive letter casing differences.
func pathsEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// hasPathPrefix checks if path starts with prefix.
// On Windows, comparison is case-insensitive due to drive letter casing differences.
func hasPathPrefix(path, prefix string) bool {
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix))
	}
	return strings.HasPrefix(path, prefix)
}

// isPathWithin checks if a file path is within a directory path (recursively).
func isPathWithin(filePath, dirPath string) bool {
	// Clean and normalize paths
	filePath = filepath.Clean(filePath)
	dirPath = filepath.Clean(dirPath)

	// Check if file starts with directory path
	if hasPathPrefix(filePath, dirPath) {
		// Make sure it's actually within (not just a prefix match)
		remainder := filePath[len(dirPath):]
		return remainder == "" || strings.HasPrefix(remainder, string(filepath.Separator))
	}
	return false
}

// isFileDirectlyIn checks if a file is directly in a directory (not in a subdirectory).
func isFileDirectlyIn(filePath, dirPath string) bool {
	// Clean and normalize paths
	filePath = filepath.Clean(filePath)
	dirPath = filepath.Clean(dirPath)

	// Get the directory of the file
	fileDir := filepath.Dir(filePath)

	// File is directly in dirPath if its parent directory matches exactly
	// Use pathsEqual for case-insensitive comparison on Windows
	return pathsEqual(fileDir, dirPath)
}

// findHCLFilesFromPaths is a helper that handles default paths and delegates to findHCLFiles.
// When recursive is false, only files in the specified directories (not subdirs) are scanned.
func findHCLFilesFromPaths(paths []string, recursive bool) ([]string, error) {
	targetPaths := paths
	if len(targetPaths) == 0 {
		targetPaths = []string{"."}
	}
	return findHCLFiles(targetPaths, recursive)
}

// findHCLFiles finds all .tf and .hcl files in the given paths.
// When recursive is true, it recursively scans subdirectories.
// When recursive is false, it only scans the specified directories (not subdirs).
func findHCLFiles(paths []string, recursive bool) ([]string, error) {
	collector := newFileCollector(recursive)
	for _, path := range paths {
		if err := collector.collectPath(path); err != nil {
			return nil, err
		}
	}
	return collector.files, nil
}

// fileCollector collects unique HCL files.
type fileCollector struct {
	files     []string
	seen      map[string]bool
	recursive bool
}

func newFileCollector(recursive bool) *fileCollector {
	return &fileCollector{seen: make(map[string]bool), recursive: recursive}
}

func (c *fileCollector) collectPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		if c.recursive {
			return c.walkDirectory(path)
		}
		return c.scanDirectory(path)
	}
	c.addFileIfHCL(path)
	return nil
}

// walkDirectory recursively walks a directory and collects HCL files.
func (c *fileCollector) walkDirectory(dir string) error {
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && shouldSkipDir(info.Name()) {
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

// scanDirectory scans only the top level of a directory (non-recursive).
func (c *fileCollector) scanDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories in non-recursive mode
		}
		path := filepath.Join(dir, entry.Name())
		c.addFileIfHCL(path)
	}
	return nil
}

func (c *fileCollector) addFileIfHCL(path string) {
	if !isHCLFile(path) {
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
func shouldSkipDir(name string) bool {
	// Skip hidden directories
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	// Skip common non-terraform directories (see package-level skipDirs)
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
		fmt.Println("Hint: Remove --changed to check all files, or stage changes with git add")
	} else {
		fmt.Println("No HCL files found")
		fmt.Println("Hint: Ensure you're in a directory with .tf, .tfvars, or .hcl files")
	}
}

// outputResults formats and prints findings, then returns an ExitError if
// there are errors. This is the shared implementation used by lint, style,
// and policy commands.
func outputResults(findings []sdk.Finding, label string, cfg *config.Config) error {
	formatter, err := output.GetFormatterWithColor(format, true, version, color, getEffectiveAbsolutePaths(cfg))
	if err != nil {
		return sdk.NewInternalError(fmt.Errorf("getting formatter: %w", err))
	}

	if err := formatter.Format(findings, os.Stdout); err != nil {
		return sdk.NewInternalError(fmt.Errorf("formatting output: %w", err))
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
			return sdk.NewFindingsError()
		}
	} else {
		// Return exit error if there are errors (for structured output)
		for _, finding := range findings {
			if finding.Severity == sdk.SeverityError {
				return sdk.NewFindingsError()
			}
		}
	}

	return nil
}

// loadConfig loads the configuration from the config file and applies the profile if specified.
// It uses the global cfgFile and profile variables from root.go.
// Returns the default config if no config file is found.
// Returns sdk.ExitError with ExitConfig code on configuration errors.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, sdk.NewConfigError(fmt.Errorf("loading config: %w", err))
	}

	// Apply profile if specified
	if profile != "" {
		if err := cfg.ApplyProfile(profile); err != nil {
			return nil, sdk.NewConfigError(fmt.Errorf("applying profile %q: %w", profile, err))
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
// Precedence: --no-parallel > --parallel > config > default (false).
func getEffectiveParallel(cfg *config.Config, cliParallel, cliNoParallel bool) bool {
	if cliNoParallel {
		return false
	}
	if cliParallel {
		return true
	}
	if cfg != nil {
		return cfg.IsParallel()
	}
	return false
}

// getEffectiveRecursive returns whether directory traversal should be recursive.
// CLI flag (--no-recurse) takes precedence over config file setting.
// Default is true (recursive) if not specified anywhere.
func getEffectiveRecursive(cfg *config.Config) bool {
	// CLI flag takes precedence
	if noRecurse {
		return false
	}
	// Then config
	if cfg != nil {
		return cfg.IsRecursive()
	}
	// Default to recursive
	return true
}

// getEffectiveAbsolutePaths returns whether output should use absolute file paths.
// CLI flag (--absolute-paths) takes precedence over config file setting.
// Default is false (relative paths) if not specified anywhere.
func getEffectiveAbsolutePaths(cfg *config.Config) bool {
	// CLI flag takes precedence
	if absolutePaths {
		return true
	}
	// Then config
	if cfg != nil {
		return cfg.IsAbsolutePaths()
	}
	// Default to relative paths
	return false
}

// shouldFailFast returns whether fail-fast mode is enabled from config.
func shouldFailFast(cfg *config.Config) bool {
	if cfg != nil {
		return cfg.IsFailFast()
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
