package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEngine_Run(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantFinding bool
		rulePrefix  string
	}{
		{
			name: "proper spacing between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}

resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: false,
		},
		{
			name: "missing blank line between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: true,
			rulePrefix:  "style.blank-line",
		},
		{
			name: "too many blank lines between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}


resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: true,
			rulePrefix:  "style.blank-line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Create engine
			engine := New(&Config{
				Fix:   false,
				Rules: make(map[string]RuleConfig),
			})

			// Run style checks
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check findings
			if tt.wantFinding {
				if len(findings) == 0 {
					t.Error("expected findings but got none")
					return
				}
			} else {
				if len(findings) != 0 {
					t.Errorf("expected no findings but got %d: %+v", len(findings), findings)
				}
			}
		})
	}
}

func TestBlankLineBetweenBlocksRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "proper spacing",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}

resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: false,
		},
		{
			name: "no spacing",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: true,
		},
		{
			name: "single block",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			found := false
			for _, f := range findings {
				if f.Rule == "style.blank-line-between-blocks" {
					found = true
					break
				}
			}

			if found != tt.wantFinding {
				t.Errorf("wanted finding=%v, got finding=%v (findings: %+v)", tt.wantFinding, found, findings)
			}
		})
	}
}
