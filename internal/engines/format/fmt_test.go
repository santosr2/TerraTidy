package format

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			wantFix: true,
		},
		{
			name: "check mode - needs formatting",
			content: `resource "aws_instance" "example"   {
ami="ami-12345678"
}
`,
			checkMode: true,
			wantFix:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(&Config{Check: tt.checkMode})
			findings, err := engine.Run(context.Background(), []string{tmpFile})

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantFix {
				require.NotEmpty(t, findings, "expected findings")
				assert.Contains(t, []string{"fmt.needs-formatting", "fmt.formatted"}, findings[0].Rule)
			} else {
				assert.Empty(t, findings)
			}

			if !tt.checkMode && tt.wantFix {
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)
				assert.Equal(t, tt.want, string(content))
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
			name:  "empty file",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \n\t\n  ",
			want:  "\n\n  ", // hclwrite normalizes but preserves some structure
		},
		{
			name:  "only hash comment",
			input: "# This is a comment\n",
			want:  "# This is a comment\n",
		},
		{
			name:  "only slash comment",
			input: "// This is a comment\n",
			want:  "// This is a comment\n",
		},
		{
			name:  "only block comment",
			input: "/* Block comment */\n",
			want:  "/* Block comment */\n",
		},
		{
			name:  "multiple comments only",
			input: "# Comment 1\n# Comment 2\n",
			want:  "# Comment 1\n# Comment 2\n",
		},
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
			assert.Equal(t, tt.want, string(got))
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
			assert.Equal(t, tt.want, isHCLFile(tt.path))
		})
	}
}

func TestConfigFromEngine(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		engineCfg := config.FmtEngineConfig{}
		cfg := ConfigFromEngine(engineCfg)

		require.NotNil(t, cfg)
		assert.False(t, cfg.Check)
		assert.False(t, cfg.Diff)
	})

	t.Run("check mode enabled", func(t *testing.T) {
		engineCfg := config.FmtEngineConfig{
			Check: true,
		}
		cfg := ConfigFromEngine(engineCfg)

		assert.True(t, cfg.Check)
		assert.False(t, cfg.Diff)
	})

	t.Run("diff mode enabled", func(t *testing.T) {
		engineCfg := config.FmtEngineConfig{
			Diff: true,
		}
		cfg := ConfigFromEngine(engineCfg)

		assert.False(t, cfg.Check)
		assert.True(t, cfg.Diff)
	})

	t.Run("both modes enabled", func(t *testing.T) {
		engineCfg := config.FmtEngineConfig{
			Check: true,
			Diff:  true,
		}
		cfg := ConfigFromEngine(engineCfg)

		assert.True(t, cfg.Check)
		assert.True(t, cfg.Diff)
	})
}
