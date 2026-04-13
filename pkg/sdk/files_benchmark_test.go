package sdk

import (
	"fmt"
	"path/filepath"
	"testing"
)

// generateFilePaths creates simulated file paths for benchmarking.
// Simulates a typical Terraform project structure with multiple modules.
func generateFilePaths(fileCount, dirsCount int) []string {
	files := make([]string, fileCount)
	for i := range fileCount {
		dir := fmt.Sprintf("modules/service-%d", i%dirsCount)
		var filename string
		switch i % 4 {
		case 0:
			filename = "main.tf"
		case 1:
			filename = "variables.tf"
		case 2:
			filename = "outputs.tf"
		case 3:
			filename = "terraform.tfvars"
		}
		files[i] = filepath.Join(dir, filename)
	}
	return files
}

// generateDeepPaths creates deeply nested file paths for benchmarking.
// Files are distributed across different directories at varying depths.
func generateDeepPaths(fileCount, depth int) []string {
	files := make([]string, fileCount)
	for i := range fileCount {
		parts := make([]string, depth+1)
		for d := range depth {
			// Vary directory at each level based on file index
			parts[d] = fmt.Sprintf("level%d-svc%d", d, (i+d)%10)
		}
		parts[depth] = fmt.Sprintf("file%d.tf", i)
		files[i] = filepath.Join(parts...)
	}
	return files
}

func BenchmarkGroupFilesByDirectory(b *testing.B) {
	tests := []struct {
		name      string
		fileCount int
		dirsCount int
	}{
		{"Small_10Files_5Dirs", 10, 5},
		{"Medium_100Files_20Dirs", 100, 20},
		{"Large_1000Files_50Dirs", 1000, 50},
		{"XLarge_5000Files_100Dirs", 5000, 100},
		{"ManyDirs_1000Files_500Dirs", 1000, 500},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			files := generateFilePaths(tc.fileCount, tc.dirsCount)
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := GroupFilesByDirectory(files)
				_ = result
			}
		})
	}
}

func BenchmarkGroupFilesByDirectory_DeepPaths(b *testing.B) {
	tests := []struct {
		name      string
		fileCount int
		depth     int
	}{
		{"100Files_Depth3", 100, 3},
		{"100Files_Depth10", 100, 10},
		{"1000Files_Depth5", 1000, 5},
		{"1000Files_Depth20", 1000, 20},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			files := generateDeepPaths(tc.fileCount, tc.depth)
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := GroupFilesByDirectory(files)
				_ = result
			}
		})
	}
}

func BenchmarkIsHCLFile(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"TerraformFile", "modules/service/main.tf"},
		{"HCLFile", "config/settings.hcl"},
		{"TFVarsFile", "environments/dev.tfvars"},
		{"NonHCLFile", "README.md"},
		{"GoFile", "internal/engine/format.go"},
		{"DeepPath", "a/b/c/d/e/f/g/main.tf"},
		{"NoExtension", "Makefile"},
		{"UppercaseExt", "main.TF"},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := IsHCLFile(tc.path)
				_ = result
			}
		})
	}
}

// BenchmarkIsHCLFile_Batch benchmarks checking many files in sequence.
func BenchmarkIsHCLFile_Batch(b *testing.B) {
	// Mix of HCL and non-HCL files
	files := []string{
		"main.tf",
		"variables.tf",
		"outputs.tf",
		"README.md",
		"terraform.tfvars",
		"config.hcl",
		"go.mod",
		"Makefile",
		"main.go",
		"dev.auto.tfvars",
	}

	tests := []struct {
		name  string
		count int
	}{
		{"10Files", 10},
		{"100Files", 100},
		{"1000Files", 1000},
	}

	for _, tc := range tests {
		// Expand file list to desired count
		expanded := make([]string, tc.count)
		for i := range tc.count {
			expanded[i] = files[i%len(files)]
		}

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				for _, f := range expanded {
					result := IsHCLFile(f)
					_ = result
				}
			}
		})
	}
}

// generateMixedFiles creates a mix of HCL and non-HCL files for workflow benchmarks.
func generateMixedFiles(fileCount, dirsCount int) []string {
	files := make([]string, fileCount)
	for i := range fileCount {
		dir := fmt.Sprintf("modules/service-%d", i%dirsCount)
		var filename string
		switch i % 5 {
		case 0:
			filename = "main.tf"
		case 1:
			filename = "variables.tf"
		case 2:
			filename = "README.md"
		case 3:
			filename = "outputs.hcl"
		case 4:
			filename = "main.go"
		}
		files[i] = filepath.Join(dir, filename)
	}
	return files
}

// BenchmarkFileDiscoveryWorkflow simulates a realistic file discovery workflow:
// 1. Filter HCL files from a list
// 2. Group them by directory
func BenchmarkFileDiscoveryWorkflow(b *testing.B) {
	tests := []struct {
		name      string
		fileCount int
		dirsCount int
	}{
		{"Small_100Files_10Dirs", 100, 10},
		{"Medium_1000Files_50Dirs", 1000, 50},
		{"Large_5000Files_100Dirs", 5000, 100},
		{"XLarge_10000Files_200Dirs", 10000, 200},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			allFiles := generateMixedFiles(tc.fileCount, tc.dirsCount)
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				// Step 1: Filter HCL files
				hclFiles := make([]string, 0, len(allFiles))
				for _, f := range allFiles {
					if IsHCLFile(f) {
						hclFiles = append(hclFiles, f)
					}
				}

				// Step 2: Group by directory
				grouped := GroupFilesByDirectory(hclFiles)
				_ = grouped
			}
		})
	}
}
