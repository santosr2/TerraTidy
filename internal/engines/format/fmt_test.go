package format

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

// TestEngine_IsDiff verifies that the IsDiff signal on findings tracks the diff
// content carried in Message: true only when Diff mode is enabled AND the file
// actually needs formatting (so a diff was generated). Covers all four
// combinations of {check,fix} x {diff,no-diff}.
func TestEngine_IsDiff(t *testing.T) {
	unformatted := `resource "aws_instance" "example"   {
ami="ami-12345678"
}
`

	tests := []struct {
		name       string
		check      bool
		diff       bool
		wantRule   string
		wantIsDiff bool
	}{
		{"check mode without diff", true, false, "fmt.needs-formatting", false},
		{"check mode with diff", true, true, "fmt.needs-formatting", true},
		{"fix mode without diff", false, false, "fmt.formatted", false},
		{"fix mode with diff", false, true, "fmt.formatted", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(unformatted), 0o644))

			engine := New(&Config{Check: tt.check, Diff: tt.diff})
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)
			require.Len(t, findings, 1, "unformatted file should produce one finding")

			f := findings[0]
			assert.Equal(t, tt.wantRule, f.Rule)
			assert.Equal(t, tt.wantIsDiff, f.IsDiff,
				"IsDiff must match whether Message carries a unified diff")
			if tt.wantIsDiff {
				assert.Contains(t, f.Message, "@@",
					"diff message must contain unified diff hunk markers")
			} else {
				assert.NotContains(t, f.Message, "@@",
					"non-diff message must not contain unified diff hunk markers")
			}
		})
	}
}

// TestEngine_PreservesFileMode verifies that running fmt on a file with a
// non-default permission mode (0o755) does not change the mode after the
// formatted bytes are written back. Skipped on Windows where Unix-style
// permission bits don't apply.
func TestEngine_PreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style file permissions don't apply on Windows")
	}

	unformatted := `resource "aws_instance" "example" {
ami="ami-12345678"
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(unformatted), 0o755))
	require.NoError(t, os.Chmod(tmpFile, 0o755), "ensure mode is set even if umask altered WriteFile's perm")

	engine := New(&Config{})
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	require.Len(t, findings, 1, "unformatted file should produce a finding")
	assert.Equal(t, "fmt.formatted", findings[0].Rule)

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(),
		"fmt must preserve original file mode after writing formatted content")
}

// TestEngine_IsDiff_NoFinding_AlreadyFormatted verifies that an already-formatted
// file produces no finding at all (so IsDiff doesn't even apply). Guards against
// a regression where the engine might emit a stub finding with IsDiff=false.
func TestEngine_IsDiff_NoFinding_AlreadyFormatted(t *testing.T) {
	formatted := `resource "aws_instance" "example" {
  ami = "ami-12345678"
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(formatted), 0o644))

	engine := New(&Config{Check: true, Diff: true})
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	assert.Empty(t, findings, "already-formatted file must produce no finding even in diff mode")
}
