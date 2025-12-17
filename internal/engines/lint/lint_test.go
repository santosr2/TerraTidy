package lint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEngine_Run(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "basic terraform file",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Create engine
			engine := New(nil)

			// Run linter
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For now, we expect at least the "not-implemented" finding
			if len(findings) == 0 {
				t.Error("expected at least one finding (not-implemented)")
			}
		})
	}
}

func TestGroupFilesByDirectory(t *testing.T) {
	engine := New(nil)

	tests := []struct {
		name  string
		files []string
		want  int // number of directories
	}{
		{
			name:  "single directory",
			files: []string{"dir1/file1.tf", "dir1/file2.tf"},
			want:  1,
		},
		{
			name:  "multiple directories",
			files: []string{"dir1/file1.tf", "dir2/file2.tf", "dir3/file3.tf"},
			want:  3,
		},
		{
			name:  "nested directories",
			files: []string{"dir1/file1.tf", "dir1/subdir/file2.tf"},
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.groupFilesByDirectory(tt.files)
			if len(result) != tt.want {
				t.Errorf("groupFilesByDirectory() got %d directories, want %d", len(result), tt.want)
			}
		})
	}
}
