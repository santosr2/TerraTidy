package fmt

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEngine_Run(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		want      string
		checkMode bool
		wantErr   bool
		wantFix   bool
	}{
		{
			name: "already formatted",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			want: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			checkMode: false,
			wantErr:   false,
			wantFix:   false,
		},
		{
			name: "needs formatting",
			content: `resource "aws_instance" "example"   {
ami="ami-12345678"
instance_type =   "t2.micro"
}
`,
			want: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			checkMode: false,
			wantErr:   false,
			wantFix:   true,
		},
		{
			name: "check mode - needs formatting",
			content: `resource "aws_instance" "example"   {
ami="ami-12345678"
}
`,
			checkMode: true,
			wantErr:   false,
			wantFix:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Create engine
			engine := New(&Config{
				Check: tt.checkMode,
			})

			// Run formatter
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check findings
			if tt.wantFix {
				if len(findings) == 0 {
					t.Error("expected findings but got none")
					return
				}
				if findings[0].Rule != "fmt.needs-formatting" && findings[0].Rule != "fmt.formatted" {
					t.Errorf("unexpected finding rule: %s", findings[0].Rule)
				}
			} else {
				if len(findings) != 0 {
					t.Errorf("expected no findings but got %d", len(findings))
				}
			}

			// In non-check mode, verify file was actually formatted
			if !tt.checkMode && tt.wantFix {
				content, err := os.ReadFile(tmpFile)
				if err != nil {
					t.Fatalf("failed to read formatted file: %v", err)
				}
				if string(content) != tt.want {
					t.Errorf("formatted content mismatch:\ngot:\n%s\nwant:\n%s", string(content), tt.want)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "basic formatting",
			input: `resource "aws_instance" "example"   {
ami="ami-12345678"
instance_type =   "t2.micro"
}
`,
			want: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
		},
		{
			name: "already formatted",
			input: `resource "aws_instance" "example" {
  ami = "ami-12345678"
}
`,
			want: `resource "aws_instance" "example" {
  ami = "ami-12345678"
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format([]byte(tt.input))
			if string(got) != tt.want {
				t.Errorf("Format() mismatch:\ngot:\n%s\nwant:\n%s", string(got), tt.want)
			}
		})
	}
}

func TestIsHCLFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"terraform file", "main.tf", true},
		{"terragrunt file", "terragrunt.hcl", true},
		{"uppercase tf", "main.TF", true},
		{"go file", "main.go", false},
		{"json file", "config.json", false},
		{"no extension", "README", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHCLFile(tt.path); got != tt.want {
				t.Errorf("isHCLFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
