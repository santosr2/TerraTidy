package lsp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/internal/plugins"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeBuffer is a thread-safe bytes.Buffer for use in concurrent tests.
type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// pathToFileURI converts a file path to a file:// URI.
// Handles Windows paths correctly (C:\path -> file:///C:/path).
func pathToFileURI(path string) string {
	// Convert backslashes to forward slashes for URI
	path = filepath.ToSlash(path)
	// On Windows, paths like C:/... need file:///C:/...
	if runtime.GOOS == "windows" || (len(path) >= 2 && path[1] == ':') {
		return "file:///" + path
	}
	// Unix paths already start with /, so file:// + /path = file:///path
	return "file://" + path
}

func TestNewServer(t *testing.T) {
	in := strings.NewReader("")
	out := &bytes.Buffer{}

	server := NewServer(in, out)

	assert.NotNil(t, server)
	assert.NotNil(t, server.reader)
	assert.NotNil(t, server.writer)
	assert.NotNil(t, server.documents)
	assert.False(t, server.initialized)
	assert.False(t, server.shutdown)
}

func TestServer_WriteMessage(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	msg := ResponseMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Result:  "test",
	}

	err := server.writeMessage(msg)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Content-Length:")
	assert.Contains(t, output, `"jsonrpc":"2.0"`)
	assert.Contains(t, output, `"result":"test"`)
}

func TestServer_ReadMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		content := `{"jsonrpc":"2.0","method":"test"}`
		// Build proper Content-Length header
		contentLen := len(content)
		input := "Content-Length: " + strconv.Itoa(contentLen) + "\r\n\r\n" + content

		server := NewServer(strings.NewReader(input), &bytes.Buffer{})

		msg, err := server.readMessage()
		require.NoError(t, err)
		assert.NotNil(t, msg)

		var req RequestMessage
		err = json.Unmarshal(msg, &req)
		require.NoError(t, err)
		assert.Equal(t, "test", req.Method)
	})

	t.Run("no content length", func(t *testing.T) {
		input := "\r\n"
		server := NewServer(strings.NewReader(input), &bytes.Buffer{})

		_, err := server.readMessage()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or missing content length")
	})

	t.Run("oversized content length rejected", func(t *testing.T) {
		// Content-Length exceeds 10 MB limit
		oversizedLength := 11 * 1024 * 1024 // 11 MB
		input := "Content-Length: " + strconv.Itoa(oversizedLength) + "\r\n\r\n"

		server := NewServer(strings.NewReader(input), &bytes.Buffer{})

		_, err := server.readMessage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("max content length allowed", func(t *testing.T) {
		// Exactly at the limit should be allowed (but we can't actually send 10MB in a test)
		// Instead, test with a reasonable size that's under the limit
		content := `{"jsonrpc":"2.0","method":"test"}`
		contentLen := len(content)
		input := "Content-Length: " + strconv.Itoa(contentLen) + "\r\n\r\n" + content

		server := NewServer(strings.NewReader(input), &bytes.Buffer{})

		msg, err := server.readMessage()
		require.NoError(t, err)
		assert.NotNil(t, msg)
	})
}

func TestServer_SendResult(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	err := server.sendResult(json.RawMessage(`1`), map[string]string{"key": "value"})
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, `"result"`)
	assert.Contains(t, output, `"key":"value"`)
}

func TestServer_SendError(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	err := server.sendError(json.RawMessage(`1`), -32600, "Invalid Request")
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, `"error"`)
	assert.Contains(t, output, `"code":-32600`)
	assert.Contains(t, output, `"message":"Invalid Request"`)
}

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		// Basic Unix paths
		{"file:///tmp/test.tf", "/tmp/test.tf"},
		{"file:///home/user/main.tf", "/home/user/main.tf"},
		{"/direct/path.tf", "/direct/path.tf"},

		// Windows drive letters
		{"file:///C:/Users/dev/main.tf", "C:/Users/dev/main.tf"},
		{"file:///D:/projects/test.tf", "D:/projects/test.tf"},

		// URL encoding - spaces
		{"file:///C:/path%20with%20spaces/main.tf", "C:/path with spaces/main.tf"},

		// URL encoding - special characters including encoded slashes
		{"file:///tmp/a%2Fb.tf", "/tmp/a/b.tf"},                   // %2F -> /
		{"file:///tmp/foo%2Fbar%2Fbaz.tf", "/tmp/foo/bar/baz.tf"}, // multiple encoded slashes
		{"file:///tmp/%2e%2e/etc/passwd", "/tmp/../etc/passwd"},   // %2e%2e -> .. (decoded, not traversed)
		{"file:///tmp/file%23name.tf", "/tmp/file#name.tf"},       // %23 -> #
		{"file:///tmp/%E2%9C%93.tf", "/tmp/\u2713.tf"},            // Unicode checkmark

		// UNC paths (Windows network shares)
		{"file://server/share/path.tf", "//server/share/path.tf"},
		{"file://myserver/data/main.tf", "//myserver/data/main.tf"},

		// Invalid URIs return empty string
		{"file://[invalid-host/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			result := uriToPath(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateWorkspacePath(t *testing.T) {
	tests := []struct {
		name          string
		workspaceRoot string
		path          string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "path within workspace",
			workspaceRoot: "/workspace",
			path:          "/workspace/src/main.tf",
			wantErr:       false,
		},
		{
			name:          "path at workspace root",
			workspaceRoot: "/workspace",
			path:          "/workspace",
			wantErr:       false,
		},
		{
			name:          "path escapes via dot-dot",
			workspaceRoot: "/workspace",
			path:          "/workspace/../etc/passwd",
			wantErr:       true,
			errContains:   "escapes workspace",
		},
		{
			name:          "path completely outside workspace",
			workspaceRoot: "/workspace",
			path:          "/etc/passwd",
			wantErr:       true,
			errContains:   "escapes workspace",
		},
		{
			name:          "empty workspace root skips validation",
			workspaceRoot: "",
			path:          "/any/path/main.tf",
			wantErr:       false,
		},
		{
			name:          "nested path within workspace",
			workspaceRoot: "/workspace/project",
			path:          "/workspace/project/modules/vpc/main.tf",
			wantErr:       false,
		},
		{
			name:          "sibling directory escape",
			workspaceRoot: "/workspace/project-a",
			path:          "/workspace/project-b/main.tf",
			wantErr:       true,
			errContains:   "escapes workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(strings.NewReader(""), &bytes.Buffer{})
			server.workspaceRoot = tt.workspaceRoot

			result, err := server.validateWorkspacePath(tt.path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestURIToPath_Security tests security-relevant URI decoding behavior.
// These tests verify that uriToPath correctly decodes encoded traversal sequences
// so that validateWorkspacePath can properly detect and block them.
func TestURIToPath_Security(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		// Basic traversal sequences (decoded correctly for validation)
		{
			name:     "encoded dot-dot slash",
			uri:      "file:///workspace/..%2Fetc/passwd",
			expected: "/workspace/../etc/passwd",
		},
		{
			name:     "double encoded dot-dot",
			uri:      "file:///workspace/%2e%2e/%2e%2e/etc/passwd",
			expected: "/workspace/../../etc/passwd",
		},
		{
			name:     "mixed encoding",
			uri:      "file:///workspace/..%2F%2e%2e/etc/passwd",
			expected: "/workspace/../../etc/passwd",
		},
		{
			name:     "uppercase hex encoding",
			uri:      "file:///workspace/%2E%2E%2Fetc/passwd",
			expected: "/workspace/../etc/passwd",
		},
		{
			name:     "mixed case hex encoding",
			uri:      "file:///workspace/%2e%2E%2Fetc/passwd",
			expected: "/workspace/../etc/passwd",
		},

		// Windows backslash encoding
		{
			name:     "encoded backslash traversal",
			uri:      "file:///C:/workspace/..%5c..%5cWindows/System32",
			expected: "C:/workspace/..\\..\\Windows/System32",
		},

		// Deep traversal
		{
			name:     "deep traversal",
			uri:      "file:///workspace/a/b/c/..%2F..%2F..%2F..%2Fetc/passwd",
			expected: "/workspace/a/b/c/../../../../etc/passwd",
		},

		// Null bytes (Go's url.PathUnescape handles these safely)
		{
			name:     "null byte in path",
			uri:      "file:///workspace/file%00.tf",
			expected: "/workspace/file\x00.tf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uriToPath(tt.uri)
			assert.Equal(t, tt.expected, result, "URI should be decoded correctly for security validation")
		})
	}
}

// TestPathTraversalBlocking tests end-to-end path traversal blocking.
// This combines uriToPath decoding with validateWorkspacePath validation.
func TestPathTraversalBlocking(t *testing.T) {
	tests := []struct {
		name          string
		workspaceRoot string
		uri           string
		shouldBlock   bool
	}{
		// Encoded traversal attempts - should be blocked
		{
			name:          "encoded dot-dot slash escape",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/..%2Fetc/passwd",
			shouldBlock:   true,
		},
		{
			name:          "double encoded dot-dot escape",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/%2e%2e/%2e%2e/etc/passwd",
			shouldBlock:   true,
		},
		{
			name:          "deep traversal escape",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/a/b/..%2F..%2F..%2F..%2Fetc/passwd",
			shouldBlock:   true,
		},

		// Legitimate paths - should be allowed
		{
			name:          "normal path within workspace",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/src/main.tf",
			shouldBlock:   false,
		},
		{
			name:          "path with encoded spaces",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/path%20with%20spaces/main.tf",
			shouldBlock:   false,
		},
		{
			name:          "nested dot-dot that stays within workspace",
			workspaceRoot: "/workspace",
			uri:           "file:///workspace/a/b/../c/main.tf",
			shouldBlock:   false,
		},

		// Windows paths
		{
			name:          "windows traversal escape",
			workspaceRoot: "C:/workspace",
			uri:           "file:///C:/workspace/..%2F..%2FWindows/System32",
			shouldBlock:   true,
		},
		{
			name:          "windows normal path",
			workspaceRoot: "C:/workspace",
			uri:           "file:///C:/workspace/modules/main.tf",
			shouldBlock:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(strings.NewReader(""), &bytes.Buffer{})
			server.workspaceRoot = tt.workspaceRoot

			// Convert URI to path (decodes encoded sequences)
			path := uriToPath(tt.uri)
			require.NotEmpty(t, path, "uriToPath should return a path")

			// Validate path stays within workspace
			_, err := server.validateWorkspacePath(path)

			if tt.shouldBlock {
				assert.Error(t, err, "path traversal should be blocked: %s", path)
			} else {
				assert.NoError(t, err, "legitimate path should be allowed: %s", path)
			}
		})
	}
}

// TestUNCPathHandling tests Windows UNC path handling for security.
func TestUNCPathHandling(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "basic UNC path",
			uri:      "file://server/share/file.tf",
			expected: "//server/share/file.tf",
		},
		{
			name:     "UNC with IP address",
			uri:      "file://192.168.1.1/share/file.tf",
			expected: "//192.168.1.1/share/file.tf",
		},
		{
			name:     "UNC admin share",
			uri:      "file://server/c$/Windows/System32",
			expected: "//server/c$/Windows/System32",
		},
		{
			name:     "UNC with encoded characters",
			uri:      "file://server/share%20name/file.tf",
			expected: "//server/share name/file.tf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uriToPath(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInvalidURIHandling tests that invalid URIs fail securely.
func TestInvalidURIHandling(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"invalid host bracket", "file://[invalid/path"},
		{"malformed percent encoding", "file:///path/%GG/file.tf"},
		{"incomplete percent encoding", "file:///path/%2/file.tf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uriToPath(tt.uri)
			// Invalid URIs should return empty string (fail secure)
			// or the path portion if URL parsing succeeds but decoding fails
			// The key is that we don't panic or return unsafe paths
			assert.NotPanics(t, func() {
				_ = uriToPath(tt.uri)
			})
			// If result is non-empty, it should still be safe to validate
			if result != "" {
				server := NewServer(strings.NewReader(""), &bytes.Buffer{})
				server.workspaceRoot = "/workspace"
				_, _ = server.validateWorkspacePath(result)
			}
		})
	}
}

// TestGetDiagnostics_PathTraversalBlocked tests that getDiagnostics returns
// empty diagnostics when a path traversal attack is attempted.
func TestGetDiagnostics_PathTraversalBlocked(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = "/workspace"

	// Add a document with a traversal attack URI
	attackURI := "file:///workspace/..%2F..%2Fetc/passwd"
	server.documents[attackURI] = &Document{
		URI:     attackURI,
		Content: "resource \"test\" {}",
		Version: 1,
	}

	// getDiagnostics should return empty when path escapes workspace
	diagnostics := server.getDiagnostics(attackURI)
	assert.Empty(t, diagnostics, "path traversal should be blocked, returning no diagnostics")
}

func TestSeverityToLSP(t *testing.T) {
	tests := []struct {
		severity sdk.Severity
		expected int
	}{
		{sdk.SeverityError, 1},
		{sdk.SeverityWarning, 2},
		{sdk.SeverityInfo, 3},
		{sdk.Severity("unknown"), 4}, // Default to hint
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := severityToLSP(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocument(t *testing.T) {
	doc := &Document{
		URI:     "file:///test.tf",
		Content: "resource {}",
		Version: 1,
	}

	assert.Equal(t, "file:///test.tf", doc.URI)
	assert.Equal(t, "resource {}", doc.Content)
	assert.Equal(t, 1, doc.Version)
}

func TestServer_HandleMessage(t *testing.T) {
	t.Run("unknown method with ID", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "unknownMethod",
		}
		content, _ := json.Marshal(msg)

		err := server.handleMessage(content)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, `"error"`)
		assert.Contains(t, output, "Method not found")
	})

	t.Run("unknown method without ID (notification)", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		msg := RequestMessage{
			JSONRPC: "2.0",
			Method:  "unknownNotification",
		}
		content, _ := json.Marshal(msg)

		err := server.handleMessage(content)
		require.NoError(t, err)

		// Should not write any response for notifications
		assert.Empty(t, out.String())
	})
}

func TestServer_HandleInitialize(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: "file:///tmp/test-project",
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, `"capabilities"`)
	assert.Contains(t, output, `"serverInfo"`)
	assert.Contains(t, output, `"terratidy-lsp"`)
	assert.Equal(t, "/tmp/test-project", server.workspaceRoot)
}

func TestServer_HandleInitialize_WithOptions(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: "file:///tmp/test-project",
		InitializationOptions: &InitializationOptions{
			Profile:           "production",
			SeverityThreshold: "error",
			Engines: EngineToggles{
				Fmt:    true,
				Style:  true,
				Lint:   false,
				Policy: true,
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify init options were stored
	require.NotNil(t, server.initOptions)
	assert.Equal(t, "production", server.initOptions.Profile)
	assert.Equal(t, "error", server.initOptions.SeverityThreshold)
	assert.True(t, server.initOptions.Engines.Fmt)
	assert.True(t, server.initOptions.Engines.Policy)
	assert.False(t, server.initOptions.Engines.Lint)

	// Severity threshold should be applied to config
	assert.Equal(t, "error", server.config.SeverityThreshold)
}

func TestServer_HandleInitialize_WithConfigPath(t *testing.T) {
	// Create a temp directory with a custom config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")

	// Write a config file with custom settings
	configContent := `
version: 1
severity_threshold: error
engines:
  fmt:
    enabled: true
  style:
    enabled: true
profiles:
  strict:
    engines:
      policy:
        enabled: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			ConfigPath: configPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify config was loaded from the custom path
	require.NotNil(t, server.config)
	assert.Equal(t, "error", server.config.SeverityThreshold)
	assert.Equal(t, configPath, server.initOptions.ConfigPath)
}

func TestServer_HandleInitialize_WithProfile(t *testing.T) {
	// Create a temp directory with a config file containing profiles
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Write a config file with profiles
	// Note: Profile can override engine settings but not severity_threshold
	// (Profile struct has Engines field only)
	configContent := `
version: 1
severity_threshold: info
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false
profiles:
  production:
    engines:
      policy:
        enabled: true
      lint:
        enabled: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			Profile: "production",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify profile was applied to config
	require.NotNil(t, server.config)
	// Base severity_threshold remains unchanged (profiles can't override this)
	assert.Equal(t, "info", server.config.SeverityThreshold)
	// Profile "production" enables policy engine (was false in base config)
	assert.True(t, server.config.Engines.Policy.IsEnabled())
	// Profile "production" disables lint engine (was true in base config)
	assert.False(t, server.config.Engines.Lint.IsEnabled())
}

func TestServer_HandleInitialize_ProfileNotFound(t *testing.T) {
	// Create a temp directory with a config file without the requested profile
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	configContent := `
version: 1
severity_threshold: info
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			Profile: "nonexistent",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	// Profile not found should not fail initialization
	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Config should still be loaded with defaults
	require.NotNil(t, server.config)
	assert.Equal(t, "info", server.config.SeverityThreshold)
}

func TestServer_HandleInitialize_SeverityThresholdOverridesConfig(t *testing.T) {
	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Config file says "info" but client will override to "error"
	configContent := `
version: 1
severity_threshold: info
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			SeverityThreshold: "warning",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Client-provided severity threshold should override config file
	require.NotNil(t, server.config)
	assert.Equal(t, "warning", server.config.SeverityThreshold)
}

func TestServer_HandleInitialize_EngineToggles_Stored(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: "file:///tmp/test-project",
		InitializationOptions: &InitializationOptions{
			Engines: EngineToggles{
				Fmt:    false,
				Style:  true,
				Lint:   false,
				Policy: true,
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify toggles are stored
	require.NotNil(t, server.initOptions)
	assert.False(t, server.initOptions.Engines.Fmt)
	assert.True(t, server.initOptions.Engines.Style)
	assert.False(t, server.initOptions.Engines.Lint)
	assert.True(t, server.initOptions.Engines.Policy)

	// Verify isEngineEnabled respects toggles
	assert.False(t, server.isEngineEnabled("fmt"))
	assert.True(t, server.isEngineEnabled("style"))
	assert.False(t, server.isEngineEnabled("lint"))
	assert.True(t, server.isEngineEnabled("policy"))
}

func TestServer_IsEngineEnabled_Defaults(t *testing.T) {
	// When no InitializationOptions are set, default behavior applies
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})

	// With nil initOptions, all engines except policy default to enabled
	assert.True(t, server.isEngineEnabled("fmt"))
	assert.True(t, server.isEngineEnabled("style"))
	assert.True(t, server.isEngineEnabled("lint"))
	assert.False(t, server.isEngineEnabled("policy")) // policy is opt-in
	assert.True(t, server.isEngineEnabled("unknown")) // unknown engines enabled by default
}

func TestServer_IsEngineEnabled_WithOptions(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.initOptions = &InitializationOptions{
		Engines: EngineToggles{
			Fmt:    true,
			Style:  false,
			Lint:   true,
			Policy: false,
		},
	}

	assert.True(t, server.isEngineEnabled("fmt"))
	assert.False(t, server.isEngineEnabled("style"))
	assert.True(t, server.isEngineEnabled("lint"))
	assert.False(t, server.isEngineEnabled("policy"))
}

func TestServer_HandleInitialize_StyleRulesFromConfig(t *testing.T) {
	// Create a temp directory with a config file that has engines.style.rules
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Config file with engines.style.rules to configure style rules
	configContent := `
version: 1
engines:
  style:
    enabled: true
    rules:
      style.resource-name-convention:
        enabled: false
        severity: info
      style.blank-lines-between-blocks:
        enabled: true
        severity: error
        config:
          min_lines: 2
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify style rules were loaded into config
	require.NotNil(t, server.config)
	assert.Len(t, server.config.Engines.Style.Rules, 2)

	// Verify resource-name-convention is disabled
	resourceRule := server.config.Engines.Style.Rules["style.resource-name-convention"]
	assert.False(t, *resourceRule.Enabled)
	assert.Equal(t, "info", resourceRule.Severity)

	// Verify blank-lines-between-blocks is enabled with error severity
	blankLinesRule := server.config.Engines.Style.Rules["style.blank-lines-between-blocks"]
	assert.True(t, *blankLinesRule.Enabled)
	assert.Equal(t, "error", blankLinesRule.Severity)
	assert.Equal(t, 2, blankLinesRule.Config["min_lines"])

	// Verify style engine was configured with rules
	require.NotNil(t, server.styleEngine)
}

func TestServer_HandleInitialize_StyleRulesAffectStyleEngine(t *testing.T) {
	// This test verifies that engines.style.rules from config are passed to the style engine
	// through buildStyleConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	configContent := `
version: 1
engines:
  style:
    enabled: true
    rules:
      style.resource-name-matches-type:
        enabled: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)

	// Verify the style config was built with engine rules
	styleCfg := server.buildStyleConfig()
	require.NotNil(t, styleCfg)

	// The disabled rule should be in the style config
	ruleConfig, exists := styleCfg.Rules["style.resource-name-matches-type"]
	require.True(t, exists, "rule should be in style config")
	assert.False(t, *ruleConfig.Enabled, "rule should be disabled")
}

// TestVSCodeSettings_MapsToLSPOptions verifies that the JSON field names in
// InitializationOptions match what the VSCode extension sends in
// vscode/src/extension.ts:getInitializationOptions().
//
// VSCode settings (package.json) -> InitializationOptions:
//
//	terratidy.profile           -> profile
//	terratidy.configPath        -> configPath
//	terratidy.engines.fmt       -> engines.fmt
//	terratidy.engines.style     -> engines.style
//	terratidy.engines.lint      -> engines.lint
//	terratidy.engines.policy    -> engines.policy
//	terratidy.severityThreshold -> severityThreshold
//
// Settings NOT sent to LSP:
//
//	terratidy.executablePath    -> used by extension to locate binary
//	terratidy.trace.server      -> standard LSP tracing, not custom option
func TestVSCodeSettings_MapsToLSPOptions(t *testing.T) {
	// Simulate what VSCode extension sends (from getInitializationOptions)
	vsCodePayload := `{
		"profile": "production",
		"configPath": "/path/to/config.yaml",
		"engines": {
			"fmt": true,
			"style": true,
			"lint": false,
			"policy": true
		},
		"severityThreshold": "warning"
	}`

	var opts InitializationOptions
	err := json.Unmarshal([]byte(vsCodePayload), &opts)
	require.NoError(t, err, "VSCode payload should unmarshal to InitializationOptions")

	// Verify all fields were correctly parsed
	assert.Equal(t, "production", opts.Profile)
	assert.Equal(t, "/path/to/config.yaml", opts.ConfigPath)
	assert.True(t, opts.Engines.Fmt)
	assert.True(t, opts.Engines.Style)
	assert.False(t, opts.Engines.Lint)
	assert.True(t, opts.Engines.Policy)
	assert.Equal(t, "warning", opts.SeverityThreshold)
}

// TestServer_ConfigChangesOnRestart verifies that when the LSP server is
// restarted, it picks up config file changes. This tests the "restart server
// to apply changes" workflow that VSCode uses.
func TestServer_ConfigChangesOnRestart(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Initial config
	initialConfig := `
version: 1
severity_threshold: info
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  policy:
    enabled: false
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err)

	// First server instance
	server1 := NewServer(strings.NewReader(""), &bytes.Buffer{})
	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server1.handleInitialize(msg)
	require.NoError(t, err)

	// Verify initial config
	assert.Equal(t, "info", server1.config.SeverityThreshold)
	assert.False(t, server1.config.Engines.Policy.IsEnabled())

	// Modify config file (simulating user editing config)
	updatedConfig := `
version: 1
severity_threshold: error
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  policy:
    enabled: true
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// "Restart" server - create new instance (simulates VSCode "Restart Server" command)
	server2 := NewServer(strings.NewReader(""), &bytes.Buffer{})

	err = server2.handleInitialize(msg)
	require.NoError(t, err)

	// Verify new config values are loaded
	assert.Equal(t, "error", server2.config.SeverityThreshold)
	assert.True(t, server2.config.Engines.Policy.IsEnabled())
}

// TestVSCodeSettings_EmptyValues verifies that empty/default VSCode settings
// are handled correctly (matching VSCode behavior of sending undefined).
func TestVSCodeSettings_EmptyValues(t *testing.T) {
	// VSCode sends undefined (omitted) for empty strings and default booleans
	vsCodePayload := `{
		"engines": {
			"fmt": true,
			"style": true,
			"lint": true,
			"policy": false
		}
	}`

	var opts InitializationOptions
	err := json.Unmarshal([]byte(vsCodePayload), &opts)
	require.NoError(t, err)

	// Empty optional strings should be empty
	assert.Empty(t, opts.Profile)
	assert.Empty(t, opts.ConfigPath)
	assert.Empty(t, opts.SeverityThreshold)

	// Default engine toggles
	assert.True(t, opts.Engines.Fmt)
	assert.True(t, opts.Engines.Style)
	assert.True(t, opts.Engines.Lint)
	assert.False(t, opts.Engines.Policy)
}

func TestServer_HandleInitialize_NilOptions(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: "file:///tmp/test-project",
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)

	// Nil options should not crash
	assert.Nil(t, server.initOptions)
	assert.NotNil(t, server.config)
}

func TestServer_HandleInitialize_WorkspaceValidation(t *testing.T) {
	t.Run("valid directory", func(t *testing.T) {
		// Create a real temp directory
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		params := InitializeParams{
			RootURI: pathToFileURI(tmpDir),
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "initialize",
			Params:  paramsJSON,
		}

		err := server.handleInitialize(msg)
		require.NoError(t, err)
		// Normalize paths for comparison (handles slash differences on Windows)
		assert.Equal(t, filepath.Clean(tmpDir), filepath.Clean(server.workspaceRoot))
	})

	t.Run("non-existent path continues", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		params := InitializeParams{
			RootURI: "file:///nonexistent/path/that/does/not/exist",
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "initialize",
			Params:  paramsJSON,
		}

		// Should not error - logs warning but continues
		err := server.handleInitialize(msg)
		require.NoError(t, err)
		assert.Equal(t, "/nonexistent/path/that/does/not/exist", server.workspaceRoot)
	})

	t.Run("file instead of directory continues", func(t *testing.T) {
		// Create a temp file (not directory)
		tmpFile, err := os.CreateTemp("", "test-workspace-*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		params := InitializeParams{
			RootURI: pathToFileURI(tmpFile.Name()),
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "initialize",
			Params:  paramsJSON,
		}

		// Should not error - logs warning but continues
		err = server.handleInitialize(msg)
		require.NoError(t, err)
		// Normalize paths for comparison (handles slash differences on Windows)
		assert.Equal(t, filepath.Clean(tmpFile.Name()), filepath.Clean(server.workspaceRoot))
	})
}

func TestServer_HandleInitialized(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})

	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	err := server.handleInitialized(msg)
	require.NoError(t, err)
	assert.True(t, server.initialized)
}

func TestServer_HandleFormatting(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectEdits bool
	}{
		{
			name:        "unformatted HCL gets formatted",
			content:     "resource \"aws_instance\" \"test\" {\nami = \"ami-123\"\n  instance_type=\"t2.micro\"\n}\n",
			expectEdits: true,
		},
		{
			name:        "already formatted HCL returns empty",
			content:     "resource \"aws_instance\" \"test\" {\n  ami           = \"ami-123\"\n  instance_type = \"t2.micro\"\n}\n",
			expectEdits: false,
		},
		{
			name:        "multi-block HCL with comments",
			content:     "# Main config\nresource \"aws_s3_bucket\" \"b\" {\nbucket=\"my-bucket\"\n}\n\n# Second block\nvariable \"name\" {\ntype=string\n}\n",
			expectEdits: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			server := NewServer(strings.NewReader(""), out)

			// Add document to the server
			uri := "file:///tmp/test.tf"
			server.documents[uri] = &Document{
				URI:     uri,
				Content: tt.content,
				Version: 1,
			}

			params := DocumentFormattingParams{
				TextDocument: TextDocumentIdentifier{URI: uri},
			}
			paramsJSON, _ := json.Marshal(params)

			msg := RequestMessage{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "textDocument/formatting",
				Params:  paramsJSON,
			}

			err := server.handleFormatting(msg)
			require.NoError(t, err)

			output := out.String()
			if tt.expectEdits {
				assert.Contains(t, output, `"newText"`)
				assert.Contains(t, output, `"range"`)
			} else {
				assert.Contains(t, output, `[]`)
			}
		})
	}
}

func TestServer_HandleFormatting_UnknownDocument(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///tmp/unknown.tf"},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/formatting",
		Params:  paramsJSON,
	}

	err := server.handleFormatting(msg)
	require.NoError(t, err)

	output := out.String()
	// nil result means no edits and no "newText" in the response
	assert.NotContains(t, output, `"newText"`)
}

func TestServer_HandleShutdown(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "shutdown",
	}

	err := server.handleShutdown(msg)
	require.NoError(t, err)
	assert.True(t, server.shutdown)
}

func TestServer_HandleDidOpen(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.lintEngine = nil  // Disable for this test
	server.styleEngine = nil // Disable for this test

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///test.go", // Non-tf file to skip diagnostics
			LanguageID: "go",
			Version:    1,
			Text:       "package main",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  paramsJSON,
	}
	content, _ := json.Marshal(msg)

	err := server.handleMessage(content)
	require.NoError(t, err)

	server.docMu.RLock()
	doc, ok := server.documents["file:///test.go"]
	server.docMu.RUnlock()

	assert.True(t, ok)
	assert.Equal(t, "package main", doc.Content)
	assert.Equal(t, 1, doc.Version)
}

func TestServer_HandleDidClose(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// First add a document
	server.docMu.Lock()
	server.documents["file:///test.tf"] = &Document{
		URI:     "file:///test.tf",
		Content: "resource {}",
		Version: 1,
	}
	server.docMu.Unlock()

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test.tf",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didClose",
		Params:  paramsJSON,
	}
	content, _ := json.Marshal(msg)

	err := server.handleMessage(content)
	require.NoError(t, err)

	server.docMu.RLock()
	_, ok := server.documents["file:///test.tf"]
	server.docMu.RUnlock()

	assert.False(t, ok)
}

func TestServer_HandleCodeAction(t *testing.T) {
	t.Run("returns fix edits for unformatted document", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		uri := "file:///test.tf"
		server.documents[uri] = &Document{
			URI:     uri,
			Content: "resource \"test\" \"x\" {\nami=\"val\"\n}\n",
			Version: 1,
		}

		params := CodeActionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Range:        Range{Start: Position{Line: 0}, End: Position{Line: 0}},
			Context: CodeActionContext{
				Diagnostics: []Diagnostic{
					{
						Range:    Range{Start: Position{Line: 0}, End: Position{Line: 0}},
						Code:     "style.blank-lines",
						Message:  "Missing blank line",
						Severity: 2,
					},
				},
			},
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "textDocument/codeAction",
			Params:  paramsJSON,
		}

		err := server.handleCodeAction(msg)
		require.NoError(t, err)

		output := out.String()
		// styleEngine is nil here so buildFixerEdits short-circuits and the
		// diagnostic bucket falls through to the shared format-fallback
		// CodeAction, which now uses the generic "Format document" title.
		assert.Contains(t, output, "Format document")
		assert.Contains(t, output, `"edit"`)
		assert.Contains(t, output, `"changes"`)
	})

	t.Run("returns empty for no diagnostics", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		uri := "file:///test.tf"
		server.documents[uri] = &Document{
			URI:     uri,
			Content: "resource {}\n",
			Version: 1,
		}

		params := CodeActionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Range:        Range{Start: Position{Line: 0}, End: Position{Line: 0}},
			Context:      CodeActionContext{Diagnostics: []Diagnostic{}},
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "textDocument/codeAction",
			Params:  paramsJSON,
		}

		err := server.handleCodeAction(msg)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, `[]`)
	})

	t.Run("returns empty for unknown document", func(t *testing.T) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(""), out)

		params := CodeActionParams{
			TextDocument: TextDocumentIdentifier{URI: "file:///unknown.tf"},
			Range:        Range{Start: Position{Line: 0}, End: Position{Line: 0}},
			Context: CodeActionContext{
				Diagnostics: []Diagnostic{
					{Code: "some-rule", Message: "something"},
				},
			},
		}
		paramsJSON, _ := json.Marshal(params)

		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "textDocument/codeAction",
			Params:  paramsJSON,
		}

		err := server.handleCodeAction(msg)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, `[]`)
	})
}

// TestServer_HandleCodeAction_PerFindingEdits verifies that handleCodeAction
// emits one CodeAction per diagnostic (rather than a single global fix), each
// tagged with its originating rule code and tied back to its diagnostic.
//
// Rationale for what is — and isn't — asserted: Fixer.Fix returns
// *sdk.FixResult (whose Edits field carries []TextEdit), but rule bodies
// currently still emit a single whole-file edit via WholeFileEdit. So both
// CodeActions in this test carry a whole-file LSP TextEdit covering the
// entire document, and asserting "ranges don't overlap" would be a
// guaranteed false negative until rules emit narrow per-rule edits. The
// behavior tested here — N diagnostics → N CodeActions, each tied to its
// own rule + diagnostic — is the architectural change LSP-side; the
// narrowness assertion belongs on the later narrow-edits work.
func TestServer_HandleCodeAction_PerFindingEdits(t *testing.T) {
	tmpDir := t.TempDir()

	// Fixture triggers two rules:
	//   - style.comment-syntax  (// comment on line 1)
	//   - style.tags-at-end     (ami attribute appears after tags inside the resource)
	tfContent := `// header comment
resource "aws_instance" "x" {
  tags = {
    Name = "test"
  }
  ami = "ami-123"
}
`
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.workspaceRoot = tmpDir
	server.styleEngine = style.New(nil)

	uri := pathToFileURI(testFile)
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}

	// Diagnostic.Message values are placeholders — the LSP code-action path
	// keys on Code, not Message, so synthetic strings keep the test decoupled
	// from rule output wording.
	commentDiag := Diagnostic{
		Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 17}},
		Code:     "style.comment-syntax",
		Message:  "synthetic: comment-syntax finding",
		Severity: 3,
	}
	tagsDiag := Diagnostic{
		Range:    Range{Start: Position{Line: 2, Character: 2}, End: Position{Line: 2, Character: 6}},
		Code:     "style.tags-at-end",
		Message:  "synthetic: tags-at-end finding",
		Severity: 2,
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 6}},
		Context:      CodeActionContext{Diagnostics: []Diagnostic{commentDiag, tagsDiag}},
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, server.handleCodeAction(RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}))

	actions := extractCodeActionsFromResponse(t, out.String())
	require.Len(t, actions, 2, "expected one CodeAction per diagnostic")

	// Map by rule code so assertions don't depend on iteration order of the
	// diagnostics slice.
	byCode := make(map[string]CodeAction, len(actions))
	for _, a := range actions {
		require.Len(t, a.Diagnostics, 1, "each CodeAction should carry its single originating diagnostic")
		byCode[a.Diagnostics[0].Code] = a
	}

	for _, code := range []string{"style.comment-syntax", "style.tags-at-end"} {
		action, ok := byCode[code]
		require.Truef(t, ok, "missing CodeAction for diagnostic %q", code)
		assert.Containsf(t, action.Title, code, "CodeAction title should reference the originating rule")
		assert.Equal(t, "quickfix", action.Kind)
		assert.True(t, action.IsPreferred)
		require.NotNil(t, action.Edit, "CodeAction must carry a WorkspaceEdit")
		edits := action.Edit.Changes[uri]
		require.NotEmpty(t, edits, "CodeAction must carry at least one TextEdit for the document URI")
	}
}

// TestServer_HandleCodeAction_FallbackToFormatForUnknownCode verifies that
// when a diagnostic's Code does not match any registered Fixer rule, the
// handler falls back to the legacy whole-file format quickfix. This preserves
// the pre-byte-range-textedits behavior for lint codes and any other
// diagnostics whose origin isn't a style.Fixer (see handleCodeAction godoc).
//
// The fixture uses unformatted HCL (no spaces around '=') so the format
// engine produces a non-trivial result; the diagnostic carries a made-up
// Code so Engine.Fixer returns nil and the fallback branch fires.
func TestServer_HandleCodeAction_FallbackToFormatForUnknownCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Unformatted: missing spaces around '='. format.Format will canonicalize.
	tfContent := "resource \"test\" \"x\" {\n  ami=\"val\"\n}\n"
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.workspaceRoot = tmpDir
	// Engine is wired so the fallback is hit via the lookup-miss branch
	// rather than the styleEng==nil short-circuit.
	server.styleEngine = style.New(nil)

	uri := pathToFileURI(testFile)
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}

	// Code is intentionally not a registered style rule. Anything that
	// Engine.Fixer can't resolve will do; using a lint-style code
	// mirrors the real-world case (lint findings never have a style Fixer).
	unknownDiag := Diagnostic{
		Range:    Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 5}},
		Code:     "lint.no-such-rule",
		Message:  "synthetic: diagnostic without a Fixer",
		Severity: 2,
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 2}},
		Context:      CodeActionContext{Diagnostics: []Diagnostic{unknownDiag}},
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, server.handleCodeAction(RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}))

	actions := extractCodeActionsFromResponse(t, out.String())
	require.Len(t, actions, 1, "fallback should emit exactly one CodeAction for the diagnostic")

	action := actions[0]
	assert.Equal(t, "Format document", action.Title, "format fallback uses a generic title, not per-diagnostic")
	assert.Equal(t, "quickfix", action.Kind)
	assert.True(t, action.IsPreferred)
	require.Len(t, action.Diagnostics, 1)
	assert.Equal(t, "lint.no-such-rule", action.Diagnostics[0].Code)

	require.NotNil(t, action.Edit, "CodeAction must carry a WorkspaceEdit")
	edits := action.Edit.Changes[uri]
	require.Len(t, edits, 1, "format fallback always emits exactly one whole-file TextEdit")

	edit := edits[0]
	wantRange := wholeFileRange(tfContent)
	assert.Equal(t, wantRange, edit.Range, "fallback edit must cover the whole document")
	wantText := string(format.Format([]byte(tfContent)))
	assert.Equal(t, wantText, edit.NewText, "fallback NewText must be the format-canonicalized content")
}

// TestServer_HandleCodeAction_FallbackAggregatesMultipleDiagnostics verifies
// that when multiple diagnostics in one request all fall back to the format
// quickfix (no matching Fixer for any of their codes), the response carries
// a SINGLE "Format document" action whose Diagnostics field collects all of
// them, rather than N duplicate format-quickfix actions. Avoids cluttering
// the editor's quick-fix menu with N copies of the same edit.
func TestServer_HandleCodeAction_FallbackAggregatesMultipleDiagnostics(t *testing.T) {
	tmpDir := t.TempDir()

	// Unformatted: missing spaces around '='. format.Format will canonicalize,
	// so formatEdits is non-nil for the aggregation path to fire.
	tfContent := "resource \"test\" \"x\" {\n  ami=\"val\"\n  type=\"t2.micro\"\n}\n"
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.workspaceRoot = tmpDir
	server.styleEngine = style.New(nil)

	uri := pathToFileURI(testFile)
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}

	// Three lint-style diagnostics, none mapped to a style Fixer; all three
	// must fall through to the shared format fallback.
	diags := []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 5}},
			Code:     "lint.rule-a",
			Message:  "synthetic: first unresolved",
			Severity: 2,
		},
		{
			Range:    Range{Start: Position{Line: 2, Character: 2}, End: Position{Line: 2, Character: 6}},
			Code:     "lint.rule-b",
			Message:  "synthetic: second unresolved",
			Severity: 2,
		},
		{
			Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 21}},
			Code:     "lint.rule-c",
			Message:  "synthetic: third unresolved",
			Severity: 1,
		},
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 3}},
		Context:      CodeActionContext{Diagnostics: diags},
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, server.handleCodeAction(RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}))

	actions := extractCodeActionsFromResponse(t, out.String())
	require.Len(t, actions, 1, "three unresolved diagnostics should aggregate into one format-fallback action")

	action := actions[0]
	assert.Equal(t, "Format document", action.Title)
	assert.Equal(t, "quickfix", action.Kind)
	require.Len(t, action.Diagnostics, 3, "fallback action must reference every diagnostic it resolves")

	gotCodes := make([]string, len(action.Diagnostics))
	for i, d := range action.Diagnostics {
		gotCodes[i] = d.Code
	}
	assert.ElementsMatch(t, []string{"lint.rule-a", "lint.rule-b", "lint.rule-c"}, gotCodes)

	require.NotNil(t, action.Edit)
	edits := action.Edit.Changes[uri]
	require.Len(t, edits, 1, "single shared whole-file format edit")
}

// TestServer_HandleCodeAction_FixerNoOpFallsThrough verifies the path where a
// rule's Fixer is found and invoked but returns no edits (nil FixResult). In
// that case buildFixerEdits returns nil and the handler falls back to the
// whole-file format quickfix. When the format engine is also a no-op (already
// canonical content) no CodeAction is emitted for the diagnostic at all.
//
// This covers the `result == nil || len(result.Edits) == 0` branch of
// buildFixerEdits which is otherwise only exercised via direct unit tests of
// the helper. Using already-canonical content lets us trigger it end-to-end
// through a real registered Fixer without needing to inject a stub.
func TestServer_HandleCodeAction_FixerNoOpFallsThrough(t *testing.T) {
	tmpDir := t.TempDir()

	// Canonical content: blank line between blocks, no formatting drift.
	// BlankLineBetweenBlocksRule.Fix returns nil FixResult for this input
	// (already-correctly-spaced); format.Format is also a no-op.
	tfContent := `resource "aws_instance" "a" {
  ami = "ami-123"
}

resource "aws_instance" "b" {
  ami = "ami-456"
}
`
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.workspaceRoot = tmpDir
	server.styleEngine = style.New(nil)

	uri := pathToFileURI(testFile)
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}

	// Synthetic diagnostic carrying a real rule code so the Fixer lookup
	// succeeds. Fix will return nil because content is already canonical.
	noopDiag := Diagnostic{
		Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 28}},
		Code:     "style.blank-line-between-blocks",
		Message:  "synthetic: rule that is a no-op on this content",
		Severity: 3,
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 6}},
		Context:      CodeActionContext{Diagnostics: []Diagnostic{noopDiag}},
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, server.handleCodeAction(RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}))

	actions := extractCodeActionsFromResponse(t, out.String())
	assert.Empty(t, actions, "no CodeAction should be emitted when both Fixer and format fallback are no-ops")
}

// errorStubFixer is an sdk.Fixer whose Fix returns a predetermined error and an
// explicit nil *sdk.FixResult. It records that Fix ran so tests can assert the
// shim path actually invoked the underlying Fixer (and didn't short-circuit
// before reaching it).
type errorStubFixer struct {
	err       error
	fixCalled bool
}

func (f *errorStubFixer) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	f.fixCalled = true
	return nil, f.err
}

// TestServer_BuildFixerEdits_ReturnsNilWhenFixerErrors pins the unit-level
// "no edits on error" contract of (*Server).buildFixerEdits: when the
// registered Fixer's Fix returns an error, the helper must return nil/empty
// edits regardless of what else is in the FixResult. A regression that leaked
// a partially-built edits slice on error would fail this test before the
// LSP-boundary integration test below runs.
func TestServer_BuildFixerEdits_ReturnsNilWhenFixerErrors(t *testing.T) {
	const ruleCode = "test.simulated-failure"
	tfContent := "resource \"test\" \"x\" {\n  ami=\"val\"\n}\n"

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.styleEngine = style.New(nil)

	stub := &errorStubFixer{err: errors.New("simulated fix failure")}
	server.styleEngine.RegisterFixerForTesting(ruleCode, stub)

	content := []byte(tfContent)
	parser := hclparse.NewParser()
	parsedFile, parseDiags := parser.ParseHCL(content, "test.tf")
	require.False(t, parseDiags.HasErrors(), "fixture must parse cleanly: %v", parseDiags)

	edits := server.buildFixerEdits(server.styleEngine, parsedFile, "test.tf", content, ruleCode)
	assert.Empty(t, edits, "buildFixerEdits must return nil/empty when the Fixer returns an error")
	assert.True(t, stub.fixCalled, "the registered Fixer's Fix must actually run — buildFixerEdits must not short-circuit before invoking it")
}

// TestServer_HandleCodeAction_FallbackWhenFixerErrors covers the err != nil
// branch of buildFixerEdits end-to-end through handleCodeAction — the only
// branch not exercised by the other code-action tests in this file. A stub
// Fixer registered via the test seam returns an error from Fix; the handler
// must drop that diagnostic's per-rule CodeAction and fall through to the
// shared "Format document" quickfix.
//
// Three assertions pin the LSP-boundary contract:
//
//  1. The stub's Fix ran — handleCodeAction reached the shim and did not
//     short-circuit before invoking the registered Fixer.
//  2. The response contains exactly one "Format document" quickfix
//     CodeAction whose edits are the whole-file format result.
//  3. No CodeAction's title equals "Fix: <ruleCode>" — guards against a
//     regression where buildFixerEdits leaks partial edits on error and
//     codeActionFor turns them into a sibling action next to the fallback.
//
// assert.Len (not require.Len) on the action count is intentional so
// assertion (3)'s loop still runs in the regression case where two
// actions appear instead of one — that's the exact scenario assertion
// (3) is meant to catch.
func TestServer_HandleCodeAction_FallbackWhenFixerErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Unformatted content so the format-fallback path produces a non-trivial
	// edit (missing spaces around '='). If formatting were a no-op the
	// fallback would emit no CodeAction at all and assertion (2) would not
	// distinguish "fallback fired" from "fallback was a no-op".
	tfContent := "resource \"test\" \"x\" {\n  ami=\"val\"\n}\n"
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.workspaceRoot = tmpDir
	server.styleEngine = style.New(nil)

	const ruleCode = "test.simulated-failure"
	stub := &errorStubFixer{err: errors.New("simulated fix failure")}
	server.styleEngine.RegisterFixerForTesting(ruleCode, stub)

	uri := pathToFileURI(testFile)
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}

	diag := Diagnostic{
		Range:    Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 5}},
		Code:     ruleCode,
		Message:  "synthetic: stub Fixer that errors",
		Severity: 2,
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 2}},
		Context:      CodeActionContext{Diagnostics: []Diagnostic{diag}},
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, server.handleCodeAction(RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}))

	// Assertion (1): the LSP path reached the stub (no upstream bypass).
	assert.True(t, stub.fixCalled, "handleCodeAction must invoke the registered Fixer's Fix even when it errors")

	actions := extractCodeActionsFromResponse(t, out.String())

	// Assertion (2): exactly one "Format document" quickfix CodeAction.
	// assert (not require) so assertion (3)'s loop still runs if a leak
	// causes len(actions) == 2.
	assert.Len(t, actions, 1, "errored Fixer should bucket into the single shared format-fallback action")

	// Assertion (3): no CodeAction title is "Fix: <ruleCode>" (the shape
	// codeActionFor produces when buildFixerEdits returns non-empty edits).
	// The stub never returns edits, so such an action could only appear
	// via a buildFixerEdits leak on error.
	leakedTitle := fmt.Sprintf("Fix: %s", ruleCode)
	var fallback *CodeAction
	for i := range actions {
		assert.NotEqualf(t, leakedTitle, actions[i].Title,
			"no CodeAction should derive from the errored Fixer (would indicate buildFixerEdits leaked partial edits on error)")
		if actions[i].Title == "Format document" {
			fallback = &actions[i]
		}
	}

	require.NotNil(t, fallback, "response must contain a Format document fallback action")
	assert.Equal(t, "quickfix", fallback.Kind)
	assert.True(t, fallback.IsPreferred)
	require.Len(t, fallback.Diagnostics, 1)
	assert.Equal(t, ruleCode, fallback.Diagnostics[0].Code, "fallback action must reference the originating diagnostic")

	require.NotNil(t, fallback.Edit, "fallback CodeAction must carry a WorkspaceEdit")
	edits := fallback.Edit.Changes[uri]
	require.Len(t, edits, 1, "format fallback emits exactly one whole-file TextEdit")
	wantRange := wholeFileRange(tfContent)
	assert.Equal(t, wantRange, edits[0].Range, "fallback edit must cover the whole document")
	wantText := string(format.Format([]byte(tfContent)))
	assert.Equal(t, wantText, edits[0].NewText, "fallback NewText must be the format-canonicalized content")
}

// extractCodeActionsFromResponse parses the LSP-framed response written to the
// server's writer and returns the []CodeAction in the response's result field.
func extractCodeActionsFromResponse(t *testing.T, framed string) []CodeAction {
	t.Helper()
	sep := "\r\n\r\n"
	idx := strings.Index(framed, sep)
	require.GreaterOrEqual(t, idx, 0, "response missing LSP header separator: %q", framed)
	body := framed[idx+len(sep):]

	var resp ResponseMessage
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	require.Nil(t, resp.Error, "expected successful response, got error: %+v", resp.Error)

	raw, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var actions []CodeAction
	require.NoError(t, json.Unmarshal(raw, &actions))
	return actions
}

func TestServer_Run_EOF(t *testing.T) {
	// Test that Run returns nil on EOF
	server := NewServer(io.LimitReader(strings.NewReader(""), 0), &bytes.Buffer{})

	err := server.Run()
	assert.NoError(t, err)
}

func TestServer_Run_Shutdown(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.shutdown = true

	err := server.Run()
	assert.NoError(t, err)
}

func TestServer_HandleDidChange(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Add document first
	server.documents["file:///test.tf"] = &Document{
		URI:     "file:///test.tf",
		Content: "initial content",
		Version: 1,
	}

	// Create didChange message
	msgJSON := `{
		"jsonrpc": "2.0",
		"method": "textDocument/didChange",
		"params": {
			"textDocument": {"uri": "file:///test.tf"},
			"contentChanges": [{"text": "updated content"}]
		}
	}`
	msg := RequestMessage{}
	err := json.Unmarshal([]byte(msgJSON), &msg)
	require.NoError(t, err)

	err = server.handleDidChange(msg)
	require.NoError(t, err)

	// Document should be updated
	doc := server.documents["file:///test.tf"]
	assert.Equal(t, "updated content", doc.Content)
}

func TestServer_HandleDidSave(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Add document
	server.documents["file:///test.tf"] = &Document{
		URI: "file:///test.tf",
		Content: `resource "aws_instance" "example" {
  ami = "ami-12345"
}`,
		Version: 1,
	}

	msgJSON := `{
		"jsonrpc": "2.0",
		"method": "textDocument/didSave",
		"params": {
			"textDocument": {"uri": "file:///test.tf"}
		}
	}`
	msg := RequestMessage{}
	err := json.Unmarshal([]byte(msgJSON), &msg)
	require.NoError(t, err)

	err = server.handleDidSave(msg)
	require.NoError(t, err)

	// Should publish diagnostics - check output has notification
	output := out.String()
	assert.Contains(t, output, "textDocument/publishDiagnostics")
}

// Note: handleExit calls os.Exit() which terminates the process,
// so it cannot be unit tested. It's tested via integration tests.

func TestServer_HandleMessage_UnknownMethod(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	msgJSON := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "unknown/method"
	}`

	err := server.handleMessage(json.RawMessage(msgJSON))
	require.NoError(t, err) // sendError returns nil after writing error response

	// Check that error response was sent
	output := out.String()
	assert.Contains(t, output, "Method not found")
}

func TestServer_HandleMessage_UnknownNotification(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Notification (no ID)
	msgJSON := `{
		"jsonrpc": "2.0",
		"method": "unknown/notification"
	}`

	err := server.handleMessage(json.RawMessage(msgJSON))
	require.NoError(t, err)

	// No response for notifications
	output := out.String()
	assert.Empty(t, output)
}

func TestServer_PublishDiagnostics_EmptyDocument(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Add empty document
	server.documents["file:///empty.tf"] = &Document{
		URI:     "file:///empty.tf",
		Content: "",
		Version: 1,
	}

	err := server.publishDiagnostics("file:///empty.tf")
	require.NoError(t, err)

	output := out.String()
	// Should still publish (with empty diagnostics)
	assert.Contains(t, output, "textDocument/publishDiagnostics")
	assert.Contains(t, output, "file:///empty.tf")
}

func TestServer_PublishDiagnostics_WithFindings(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Add document with style violations
	server.documents["file:///test.tf"] = &Document{
		URI: "file:///test.tf",
		Content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}`,
		Version: 1,
	}

	err := server.publishDiagnostics("file:///test.tf")
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "textDocument/publishDiagnostics")
}

func TestServer_PublishDiagnostics_TempFileReuse(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	uri := "file:///tmp/reuse.tf"
	doc := &Document{
		URI:     uri,
		Content: "resource {}\n",
		Version: 1,
	}
	server.documents[uri] = doc

	// First call creates the temp file
	err := server.publishDiagnostics(uri)
	require.NoError(t, err)
	assert.NotEmpty(t, doc.tempFile, "temp file should be created")
	firstTempFile := doc.tempFile

	// Second call reuses the same temp file
	out.Reset()
	doc.Content = "resource {}\nvariable {}\n"
	err = server.publishDiagnostics(uri)
	require.NoError(t, err)
	assert.Equal(t, firstTempFile, doc.tempFile, "temp file should be reused")

	// Cleanup
	_ = os.Remove(doc.tempFile)
}

func TestServer_PublishDiagnostics_TempFileHCLExtension(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	uri := "file:///tmp/config.hcl"
	doc := &Document{
		URI:     uri,
		Content: "variable {}\n",
		Version: 1,
	}
	server.documents[uri] = doc

	err := server.publishDiagnostics(uri)
	require.NoError(t, err)
	assert.NotEmpty(t, doc.tempFile)
	assert.Contains(t, doc.tempFile, ".hcl")

	_ = os.Remove(doc.tempFile)
}

func TestServer_PublishDiagnostics_NonTerraformSkipped(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	uri := "file:///tmp/readme.md"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "# readme",
		Version: 1,
	}

	err := server.publishDiagnostics(uri)
	require.NoError(t, err)

	// Non-terraform files publish empty diagnostics (clears stale diagnostics)
	output := out.String()
	assert.Contains(t, output, "textDocument/publishDiagnostics")
	assert.Contains(t, output, `"diagnostics":[]`)
}

func TestServer_HandleDidClose_CleansUpTempFile(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Create a temp file to simulate diagnostics having run
	tmpFile, err := os.CreateTemp("", "terratidy-test-*.tf")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	tmpPath := tmpFile.Name()

	uri := "file:///tmp/cleanup.tf"
	server.documents[uri] = &Document{
		URI:      uri,
		Content:  "resource {}\n",
		Version:  1,
		tempFile: tmpPath,
	}

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didClose",
		Params:  paramsJSON,
	}

	err = server.handleDidClose(msg)
	require.NoError(t, err)

	// Verify temp file was removed
	_, statErr := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(statErr), "temp file should be deleted on close")

	// Verify document was removed
	server.docMu.RLock()
	_, exists := server.documents[uri]
	server.docMu.RUnlock()
	assert.False(t, exists, "document should be removed")
}

func TestServer_HandleDidClose_NoTempFile(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	uri := "file:///tmp/no-temp.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "resource {}\n",
		Version: 1,
		// tempFile intentionally empty (document never analyzed)
	}

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didClose",
		Params:  paramsJSON,
	}

	err := server.handleDidClose(msg)
	require.NoError(t, err)

	server.docMu.RLock()
	_, exists := server.documents[uri]
	server.docMu.RUnlock()
	assert.False(t, exists)
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"info", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"error", LogLevelError},
		{"off", LogLevelOff},
		{"DEBUG", LogLevelDebug},
		{"unknown", LogLevelInfo},
		{"", LogLevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLogLevel(tt.input))
		})
	}
}

func TestServer_SetLogLevel(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	assert.Equal(t, LogLevelInfo, server.logLevel)

	server.SetLogLevel(LogLevelDebug)
	assert.Equal(t, LogLevelDebug, server.logLevel)

	server.SetLogLevel(LogLevelOff)
	assert.Equal(t, LogLevelOff, server.logLevel)
}

func TestServer_SetLogFile(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})

	logFile := filepath.Join(t.TempDir(), "test.log")
	err := server.SetLogFile(logFile)
	require.NoError(t, err)

	server.SetLogLevel(LogLevelDebug)
	server.logDebug("test message %d", 42)

	// Close before reading so Windows releases the file handle
	require.NoError(t, server.Close())

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message 42")
	assert.Contains(t, string(content), "[DEBUG]")
}

func TestServer_SetLogFile_InvalidPath(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	err := server.SetLogFile("/nonexistent/dir/file.log")
	assert.Error(t, err)
}

func TestServer_Close_NoLogFile(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	err := server.Close()
	assert.NoError(t, err)
}

func TestServer_LogError(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})

	logFile := filepath.Join(t.TempDir(), "error.log")
	require.NoError(t, server.SetLogFile(logFile))
	server.SetLogLevel(LogLevelError)

	server.logError("something went wrong: %s", "test")
	// logDebug should be suppressed at error level
	server.logDebug("this should not appear")

	require.NoError(t, server.Close())

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "[ERROR]")
	assert.Contains(t, string(content), "something went wrong: test")
	assert.NotContains(t, string(content), "[DEBUG]")
}

func TestServer_HandleMessage_BeforeInitialize(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Try to send textDocument/didOpen before initialize
	msgJSON := `{
		"jsonrpc": "2.0",
		"method": "textDocument/didOpen",
		"params": {}
	}`

	err := server.handleMessage(json.RawMessage(msgJSON))
	// Before initialize, the server should handle messages gracefully.
	// An error is acceptable (not initialized), but it must not panic.
	assert.NoError(t, err, "handleMessage before initialize should not error")
}

func TestServer_HandleShutdown_ReturnsNull(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`42`),
		Method:  "shutdown",
	}

	err := server.handleShutdown(msg)
	require.NoError(t, err)
	assert.True(t, server.shutdown)

	// LSP spec requires result: null (not omitted)
	output := out.String()
	assert.Contains(t, output, `"result":null`)
	assert.Contains(t, output, `"id":42`)
}

func TestServer_HandleDiagnostic(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Add a document
	uri := "file:///tmp/test.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "resource \"test\" \"example\" {}\n",
		Version: 1,
	}

	params := DocumentDiagnosticParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/diagnostic",
		Params:  paramsJSON,
	}

	err := server.handleDiagnostic(msg)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, `"kind":"full"`)
	assert.Contains(t, output, `"items"`)
}

func TestServer_HandleDiagnostic_UnknownDocument(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := DocumentDiagnosticParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///unknown.tf"},
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "textDocument/diagnostic",
		Params:  paramsJSON,
	}

	err := server.handleDiagnostic(msg)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, `"kind":"full"`)
	assert.Contains(t, output, `"items":[]`)
}

func TestServer_BuildStyleConfig_NilConfig(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.config = nil

	cfg := server.buildStyleConfig()

	require.NotNil(t, cfg)
	assert.NotNil(t, cfg.Rules)
	assert.Empty(t, cfg.Rules)
}

func TestServer_BuildStyleConfig_WithEngineStyleRules(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.config = &config.Config{
		Engines: config.Engines{
			Style: config.StyleEngineConfig{
				Rules: map[string]config.RuleConfig{
					"style.terraform-block-first": {
						Enabled:  config.BoolPtr(false),
						Severity: "info",
					},
					"style.blank-line-between-blocks": {
						Enabled:  config.BoolPtr(true),
						Severity: "warning",
					},
				},
			},
		},
	}

	cfg := server.buildStyleConfig()

	require.NotNil(t, cfg)
	assert.Len(t, cfg.Rules, 2)

	rule1 := cfg.Rules["style.terraform-block-first"]
	assert.False(t, *rule1.Enabled)
	assert.Equal(t, "info", rule1.Severity)

	rule2 := cfg.Rules["style.blank-line-between-blocks"]
	assert.True(t, *rule2.Enabled)
	assert.Equal(t, "warning", rule2.Severity)
}

func TestServer_BuildStyleConfig_WithRuleOptions(t *testing.T) {
	// Tests that Config options are passed through as Options
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.config = &config.Config{
		Engines: config.Engines{
			Style: config.StyleEngineConfig{
				Rules: map[string]config.RuleConfig{
					"style.naming": {
						Enabled:  config.BoolPtr(true),
						Severity: "warning",
						Config: map[string]any{
							"pattern": "^[a-z_]+$",
						},
					},
				},
			},
		},
	}

	cfg := server.buildStyleConfig()

	require.NotNil(t, cfg)
	require.Len(t, cfg.Rules, 1)

	rule := cfg.Rules["style.naming"]
	assert.True(t, *rule.Enabled)
	assert.Equal(t, "warning", rule.Severity)
	require.NotNil(t, rule.Options)
	assert.Equal(t, "^[a-z_]+$", rule.Options["pattern"])
}

func TestServer_GetSeverityThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
		expected  sdk.Severity
	}{
		{"default when nil config", "", sdk.SeverityInfo},
		{"error threshold", "error", sdk.SeverityError},
		{"warning threshold", "warning", sdk.SeverityWarning},
		{"info threshold", "info", sdk.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(strings.NewReader(""), &bytes.Buffer{})
			if tt.threshold != "" {
				server.config = &config.Config{SeverityThreshold: tt.threshold}
			}
			assert.Equal(t, tt.expected, server.getSeverityThreshold())
		})
	}
}

func TestMeetsThreshold(t *testing.T) {
	tests := []struct {
		severity  sdk.Severity
		threshold sdk.Severity
		expected  bool
	}{
		// Error threshold: only errors pass
		{sdk.SeverityError, sdk.SeverityError, true},
		{sdk.SeverityWarning, sdk.SeverityError, false},
		{sdk.SeverityInfo, sdk.SeverityError, false},
		// Warning threshold: errors and warnings pass
		{sdk.SeverityError, sdk.SeverityWarning, true},
		{sdk.SeverityWarning, sdk.SeverityWarning, true},
		{sdk.SeverityInfo, sdk.SeverityWarning, false},
		// Info threshold: all pass
		{sdk.SeverityError, sdk.SeverityInfo, true},
		{sdk.SeverityWarning, sdk.SeverityInfo, true},
		{sdk.SeverityInfo, sdk.SeverityInfo, true},
	}

	for _, tt := range tests {
		name := string(tt.severity) + "_with_" + string(tt.threshold) + "_threshold"
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, meetsThreshold(tt.severity, tt.threshold))
		})
	}
}

func TestServer_ResourceLimits_DocumentSize(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Create content that exceeds maxDocumentSize
	largeContent := strings.Repeat("x", maxDocumentSize+1)

	// Try to open a document that's too large
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///test.tf",
			LanguageID: "terraform",
			Version:    1,
			Text:       largeContent,
		},
	}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  paramsJSON,
	}

	err := server.handleDidOpen(msg)
	require.NoError(t, err) // Should not error, just ignore

	// Document should NOT be added
	server.docMu.RLock()
	_, exists := server.documents["file:///test.tf"]
	server.docMu.RUnlock()
	assert.False(t, exists, "oversized document should not be added")
}

func TestServer_ResourceLimits_DocumentCount(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// Fill up the document map to the limit
	server.docMu.Lock()
	for i := range maxDocuments {
		uri := "file:///doc" + strings.Repeat("0", 4) + string(rune('a'+i/26/26%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i%26)) + ".tf"
		server.documents[uri] = &Document{URI: uri, Content: ""}
	}
	docCount := len(server.documents)
	server.docMu.Unlock()

	require.Equal(t, maxDocuments, docCount, "should have exactly maxDocuments")

	// Try to open one more document
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///overflow.tf",
			LanguageID: "terraform",
			Version:    1,
			Text:       "resource \"test\" \"x\" {}",
		},
	}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  paramsJSON,
	}

	err := server.handleDidOpen(msg)
	require.NoError(t, err) // Should not error, just ignore

	// New document should NOT be added
	server.docMu.RLock()
	_, exists := server.documents["file:///overflow.tf"]
	server.docMu.RUnlock()
	assert.False(t, exists, "document should not be added when at limit")
}

func TestServer_ResourceLimits_ChangeSize(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	// First open a normal document
	server.docMu.Lock()
	server.documents["file:///test.tf"] = &Document{
		URI:     "file:///test.tf",
		Content: "small content",
		Version: 1,
	}
	server.docMu.Unlock()

	// Try to change it to something too large
	largeContent := strings.Repeat("x", maxDocumentSize+1)
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     "file:///test.tf",
			Version: 2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: largeContent},
		},
	}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didChange",
		Params:  paramsJSON,
	}

	err := server.handleDidChange(msg)
	require.NoError(t, err) // Should not error, just ignore

	// Document content should NOT be changed
	server.docMu.RLock()
	doc := server.documents["file:///test.tf"]
	server.docMu.RUnlock()
	assert.Equal(t, "small content", doc.Content, "content should not be updated with oversized change")
}

func TestServer_ResourceLimits_SemaphoreInitialized(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Verify semaphore is initialized with correct capacity
	assert.NotNil(t, server.diagSem)
	assert.Equal(t, maxConcurrentDiagnostics, cap(server.diagSem))
}

func TestServer_ResourceLimits_SemaphoreUsedInDiagnostics(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(`resource "test" "x" {}`), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.workspaceRoot = tmpDir

	// Add a document
	uri := pathToFileURI(testFile)
	server.docMu.Lock()
	server.documents[uri] = &Document{
		URI:     uri,
		Content: `resource "test" "x" {}`,
		Version: 1,
	}
	server.docMu.Unlock()

	// Verify semaphore is empty before
	assert.Equal(t, 0, len(server.diagSem))

	// Call getDiagnostics - this exercises the semaphore acquire/release
	diagnostics := server.getDiagnostics(uri)

	// Semaphore should be released after getDiagnostics returns
	assert.Equal(t, 0, len(server.diagSem))

	// Should return diagnostics (may be empty if no engines configured)
	assert.NotNil(t, diagnostics)
}

func TestServer_InitSessionTempDir(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	server.initSessionTempDir()

	// Should have created a session temp directory
	assert.NotEmpty(t, server.sessionTempDir)
	assert.DirExists(t, server.sessionTempDir)

	// Directory should be under cache path
	assert.Contains(t, server.sessionTempDir, "terratidy")

	// Clean up
	err := server.Close()
	require.NoError(t, err)

	// Directory should be removed after Close
	assert.NoDirExists(t, server.sessionTempDir)
}

func TestServer_SessionTempDir_UsedForDocs(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(`resource "test" "x" {}`), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.workspaceRoot = tmpDir

	// Initialize session temp dir
	server.initSessionTempDir()

	// Add a document
	uri := pathToFileURI(testFile)
	server.docMu.Lock()
	server.documents[uri] = &Document{
		URI:     uri,
		Content: `resource "test" "x" {}`,
		Version: 1,
	}
	server.docMu.Unlock()

	// Get diagnostics - this creates temp file
	_ = server.getDiagnostics(uri)

	// Check that temp file is in session directory
	server.docMu.RLock()
	doc := server.documents[uri]
	server.docMu.RUnlock()

	if doc.tempFile != "" {
		assert.True(t, strings.HasPrefix(doc.tempFile, server.sessionTempDir),
			"temp file %s should be under session dir %s", doc.tempFile, server.sessionTempDir)
	}

	// Clean up
	_ = server.Close()
}

func TestGetSessionTempBaseDir(t *testing.T) {
	// Test that it returns a valid path
	baseDir := getSessionTempBaseDir()
	assert.NotEmpty(t, baseDir)
	assert.Contains(t, baseDir, "terratidy")
}

func TestGetSessionTempBaseDir_XDGCacheHome(t *testing.T) {
	// Save and restore env var
	orig := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		if orig == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", orig)
		}
	}()

	// Set XDG_CACHE_HOME
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	baseDir := getSessionTempBaseDir()
	assert.True(t, strings.HasPrefix(baseDir, testDir))
	assert.Contains(t, baseDir, "terratidy")
	assert.Contains(t, baseDir, "lsp-tmp")
}

func TestServer_CleanupOldSessions(t *testing.T) {
	// Create a temp base directory
	baseDir := t.TempDir()

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Create an "old" session directory with old mtime
	oldSession := filepath.Join(baseDir, "old-session")
	require.NoError(t, os.MkdirAll(oldSession, 0o700))

	// Note: We can't easily set mtime to be old on all platforms,
	// so we just verify the function doesn't crash
	server.cleanupOldSessions(baseDir)

	// Function should complete without error
	// (actual cleanup would require manipulating mtimes)
}

func TestServer_CleanupOldSessions_SkipsFiles(t *testing.T) {
	baseDir := t.TempDir()

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Create a file (not a directory) in the base directory
	testFile := filepath.Join(baseDir, "not-a-directory.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

	// Should not crash when encountering files
	server.cleanupOldSessions(baseDir)

	// File should still exist (not deleted)
	assert.FileExists(t, testFile)
}

func TestServer_CleanupOldSessions_NonexistentDir(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Should not crash when directory doesn't exist
	server.cleanupOldSessions("/nonexistent/path/that/does/not/exist")
}

// TestServer_HandleInitialize_RootPath verifies that the deprecated rootPath
// field is used when rootUri is absent.
func TestServer_HandleInitialize_RootPath(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootPath: "/tmp/test-via-root-path",
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test-via-root-path", server.workspaceRoot)
}

// TestServer_HandleInitialize_WithPluginsEnabled verifies the plugin-loading
// branch inside handleInitialize when plugins.enabled is true in the config.
func TestServer_HandleInitialize_WithPluginsEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a YAML plugin rule so the manager loads something real.
	pluginDir := filepath.Join(tmpDir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	yamlRule := `name: lsp-init-plugin-rule
description: Rule loaded during initialize
severity: warning
enabled: true
message: "Test finding"
patterns:
  required_attributes:
    - test_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "init-rule.yaml"), []byte(yamlRule), 0o644))

	// Write a .terratidy.yaml that enables plugins pointing at the plugin dir.
	cfgContent := `version: 1
plugins:
  enabled: true
  directories:
    - ` + pluginDir + `
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".terratidy.yaml"), []byte(cfgContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err := server.handleInitialize(msg)
	require.NoError(t, err)

	// Plugin rules should have been loaded into the server.
	assert.Len(t, server.pluginRules, 1, "should have loaded 1 plugin rule via initialize")
}

// TestServer_HandleInitialize_WithPluginLoadError verifies that a plugin load
// error is non-fatal: the server continues without plugin rules.
func TestServer_HandleInitialize_WithPluginLoadError(t *testing.T) {
	tmpDir := t.TempDir()

	// Config points plugins at a path that does not exist.
	cfgContent := `version: 1
plugins:
  enabled: true
  directories:
    - /nonexistent/plugin/directory/that/will/fail
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".terratidy.yaml"), []byte(cfgContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
	}
	paramsJSON, _ := json.Marshal(params)

	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	// Non-existent directory: loadFromDirectory returns nil (skips missing dirs),
	// so pluginRules should be empty but not nil and no error returned.
	err := server.handleInitialize(msg)
	require.NoError(t, err, "plugin load failure should not prevent initialization")
}

// TestServer_HandleDidSave_WithTextContent verifies the branch where didSave
// includes the document text (params.Text != "").
func TestServer_HandleDidSave_WithTextContent(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true

	uri := "file:///tmp/save-with-text.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "initial content",
		Version: 1,
	}

	newText := `resource "aws_instance" "updated" {
  ami = "ami-99999"
}
`
	msgJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didSave",
		"params": map[string]any{
			"textDocument": map[string]string{"uri": uri},
			"text":         newText,
		},
	})

	msg := RequestMessage{}
	require.NoError(t, json.Unmarshal(msgJSON, &msg))

	err := server.handleDidSave(msg)
	require.NoError(t, err)

	// Content should be updated from the text field.
	server.docMu.RLock()
	doc := server.documents[uri]
	server.docMu.RUnlock()
	assert.Equal(t, newText, doc.Content, "document content should be updated from didSave text field")

	// Diagnostics notification should have been published.
	output := out.String()
	assert.Contains(t, output, "textDocument/publishDiagnostics")
}

func TestServer_LintEngineWithPluginRules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a YAML plugin rule that requires 'tags' attribute
	yamlRule := `name: lsp-require-tags
description: Resources must have tags
severity: warning
enabled: true
message: "Resource is missing 'tags' attribute"
patterns:
  required_attributes:
    - tags
`
	pluginDir := filepath.Join(tmpDir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "require-tags.yaml"), []byte(yamlRule), 0o644))

	// Create a test TF file without tags (will trigger the rule)
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	// Create server and configure with plugin rules
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.workspaceRoot = tmpDir

	// Load plugin rules using the plugin manager
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}

	mgr := plugins.NewManager(cfg.Plugins.Directories, false)
	require.NoError(t, mgr.LoadAll())

	rulesMap := mgr.GetRules()
	server.pluginRules = make([]sdk.Rule, 0, len(rulesMap))
	for _, rule := range rulesMap {
		server.pluginRules = append(server.pluginRules, rule)
	}
	require.Len(t, server.pluginRules, 1, "should have loaded 1 plugin rule")

	// Initialize engines with plugin rules
	server.lintEngine = lint.New(nil, server.pluginRules...)
	server.styleEngine = style.New(nil, server.pluginRules...)

	// Add document
	uri := pathToFileURI(testFile)
	server.docMu.Lock()
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}
	server.docMu.Unlock()

	// Get diagnostics
	diagnostics := server.getDiagnostics(uri)

	// Should have at least one diagnostic from the plugin rule
	var foundPluginDiagnostic bool
	for _, diag := range diagnostics {
		if strings.Contains(diag.Message, "tags") {
			foundPluginDiagnostic = true
			break
		}
	}
	assert.True(t, foundPluginDiagnostic, "should have diagnostic from plugin rule requiring tags")
}

// TestServer_GetDiagnostics_EnginesDisabledViaToggles verifies that when both lint
// and style engines are initialized but disabled via InitializationOptions, the
// engine Run methods are not called and no diagnostics are returned.
func TestServer_GetDiagnostics_EnginesDisabledViaToggles(t *testing.T) {
	tmpDir := t.TempDir()

	tfContent := `resource "aws_instance" "test" {
  ami = "ami-123"
}
`
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte(tfContent), 0o644))

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.workspaceRoot = tmpDir

	// Initialize both engines (non-nil) so they would normally produce diagnostics.
	server.lintEngine = lint.New(nil)
	server.styleEngine = style.New(nil)

	// Disable both engines via initOptions.
	server.initOptions = &InitializationOptions{
		Engines: EngineToggles{
			Fmt:    false,
			Style:  false,
			Lint:   false,
			Policy: false,
		},
	}

	uri := pathToFileURI(testFile)
	server.docMu.Lock()
	server.documents[uri] = &Document{
		URI:     uri,
		Content: tfContent,
		Version: 1,
	}
	server.docMu.Unlock()

	diagnostics := server.getDiagnostics(uri)

	// Both engines are disabled, so no diagnostics should be produced even
	// though the engines are initialized and the file would otherwise trigger findings.
	assert.Empty(t, diagnostics, "disabled engines should produce no diagnostics")
}

func TestServer_Debounce_RapidChanges(t *testing.T) {
	// Test that rapid changes within debounce window trigger single diagnostics
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	// Generous delay so the "timer still pending" check below is race-safe: the
	// five sends finish in microseconds, far inside this window, so no scheduler
	// stall short of the full delay can let the timer fire early and flake it.
	server.debounceDelay = 200 * time.Millisecond

	// Add document
	uri := "file:///test.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "initial",
		Version: 1,
	}

	// Simulate 5 rapid changes (within debounce window)
	for i := range 5 {
		msgJSON := fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"method": "textDocument/didChange",
			"params": {
				"textDocument": {"uri": "%s", "version": %d},
				"contentChanges": [{"text": "content %d"}]
			}
		}`, uri, i+2, i)
		msg := RequestMessage{}
		require.NoError(t, json.Unmarshal([]byte(msgJSON), &msg))
		require.NoError(t, server.handleDidChange(msg))
	}

	// Verify timer is pending
	server.debounceMu.Lock()
	_, hasPending := server.debounceTimers[uri]
	server.debounceMu.Unlock()
	assert.True(t, hasPending, "should have pending debounce timer")

	// Wait for the single coalesced fire instead of guessing a sleep duration.
	require.Eventually(t, func() bool {
		return strings.Count(out.String(), "textDocument/publishDiagnostics") == 1
	}, 2*time.Second, 5*time.Millisecond, "coalesced debounce should fire once")

	// The map entry is deleted before the diagnostic is published, so once the
	// fire is observed the timer is already cleaned up.
	server.debounceMu.Lock()
	_, stillPending := server.debounceTimers[uri]
	server.debounceMu.Unlock()
	assert.False(t, stillPending, "debounce timer should be cleaned up after firing")

	// Regression guard: with coalescing intact only one timer is ever live, so
	// a second publish appearing here would mean the rapid changes failed to
	// coalesce.
	require.Never(t, func() bool {
		return strings.Count(out.String(), "textDocument/publishDiagnostics") > 1
	}, 100*time.Millisecond, 10*time.Millisecond, "rapid changes should trigger single diagnostics")
}

func TestServer_Debounce_SlowChanges(t *testing.T) {
	// Test that changes separated by more than debounce delay trigger separate diagnostics
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.debounceDelay = 30 * time.Millisecond // Short delay for test

	// Add document
	uri := "file:///test.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "initial",
		Version: 1,
	}

	// First change
	msg1JSON := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "textDocument/didChange",
		"params": {
			"textDocument": {"uri": "%s", "version": 2},
			"contentChanges": [{"text": "content 1"}]
		}
	}`, uri)
	msg1 := RequestMessage{}
	require.NoError(t, json.Unmarshal([]byte(msg1JSON), &msg1))
	require.NoError(t, server.handleDidChange(msg1))

	// Wait for the first debounce to actually fire before sending the second
	// change, so the two changes are guaranteed to debounce separately
	// regardless of scheduler jitter. A fixed sleep here flakes under CPU
	// contention and the race detector, where the 30ms timer can drift past a
	// 50ms window.
	require.Eventually(t, func() bool {
		return strings.Count(out.String(), "textDocument/publishDiagnostics") == 1
	}, 2*time.Second, 5*time.Millisecond, "first debounce should fire")

	// Second change (after first debounce completed)
	msg2JSON := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "textDocument/didChange",
		"params": {
			"textDocument": {"uri": "%s", "version": 3},
			"contentChanges": [{"text": "content 2"}]
		}
	}`, uri)
	msg2 := RequestMessage{}
	require.NoError(t, json.Unmarshal([]byte(msg2JSON), &msg2))
	require.NoError(t, server.handleDidChange(msg2))

	// Wait for the second debounce to fire.
	require.Eventually(t, func() bool {
		return strings.Count(out.String(), "textDocument/publishDiagnostics") == 2
	}, 2*time.Second, 5*time.Millisecond, "second debounce should fire")

	// Regression guard: it should settle at exactly two. A third publish here
	// would mean the second change spilled an extra fire (symmetric with the
	// no-coalescing guard in RapidChanges).
	require.Never(t, func() bool {
		return strings.Count(out.String(), "textDocument/publishDiagnostics") > 2
	}, 100*time.Millisecond, 10*time.Millisecond, "slow changes should trigger exactly two diagnostics")
}

func TestServer_Debounce_CancelOnClose(t *testing.T) {
	// Test that closing document cancels pending diagnostics
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.debounceDelay = 100 * time.Millisecond // Long enough to close before firing

	// Add document
	uri := "file:///test.tf"
	server.documents[uri] = &Document{
		URI:     uri,
		Content: "initial",
		Version: 1,
	}

	// Trigger change (starts debounce timer)
	changeJSON := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "textDocument/didChange",
		"params": {
			"textDocument": {"uri": "%s", "version": 2},
			"contentChanges": [{"text": "new content"}]
		}
	}`, uri)
	changeMsg := RequestMessage{}
	require.NoError(t, json.Unmarshal([]byte(changeJSON), &changeMsg))
	require.NoError(t, server.handleDidChange(changeMsg))

	// Verify timer is pending
	server.debounceMu.Lock()
	_, hasPending := server.debounceTimers[uri]
	server.debounceMu.Unlock()
	assert.True(t, hasPending, "should have pending debounce timer")

	// Close document immediately (before debounce fires)
	closeJSON := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "textDocument/didClose",
		"params": {
			"textDocument": {"uri": "%s"}
		}
	}`, uri)
	closeMsg := RequestMessage{}
	require.NoError(t, json.Unmarshal([]byte(closeJSON), &closeMsg))
	require.NoError(t, server.handleDidClose(closeMsg))

	// Verify timer was canceled
	server.debounceMu.Lock()
	_, stillPending := server.debounceTimers[uri]
	server.debounceMu.Unlock()
	assert.False(t, stillPending, "debounce timer should be canceled on close")

	// Wait past debounce delay
	time.Sleep(150 * time.Millisecond)

	// Verify no diagnostics were published (only clear diagnostics on close)
	output := out.String()
	// The didClose sends a clear diagnostics message, but we should NOT have the debounced one
	// Count only content-type headers followed by publishDiagnostics
	publishCount := strings.Count(output, "textDocument/publishDiagnostics")
	// The close sends one publishDiagnostics with empty diagnostics
	assert.Equal(t, 1, publishCount, "should only have clear diagnostics from close, not debounced diagnostics")
}

// TestServer_ConfigReload_FileChange tests that modifying the config file
// triggers an automatic reload of the configuration.
func TestServer_ConfigReload_FileChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Initial config
	initialConfig := `version: 1
severity_threshold: info
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err)

	// Initialize server with config
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Verify initial config (use mutex to avoid race)
	server.engineMu.RLock()
	initialThreshold := server.config.SeverityThreshold
	server.engineMu.RUnlock()
	require.Equal(t, "info", initialThreshold)

	// Verify config watcher is running
	server.configWatcherMu.Lock()
	hasWatcher := server.configWatcher != nil
	server.configWatcherMu.Unlock()
	require.True(t, hasWatcher, "config watcher should be initialized")

	// Modify config file
	updatedConfig := `version: 1
severity_threshold: error
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Wait for fsnotify to detect and process the change (use Eventually for reliability)
	require.Eventually(t, func() bool {
		server.engineMu.RLock()
		threshold := server.config.SeverityThreshold
		server.engineMu.RUnlock()
		return threshold == "error"
	}, 2*time.Second, 10*time.Millisecond, "config should reload with new severity_threshold")
}

// TestServer_ConfigReload_ImportedFile tests that modifying an imported config file
// triggers a reload of the configuration.
func TestServer_ConfigReload_ImportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	importedPath := filepath.Join(tmpDir, "imported.yaml")

	// Create imported config
	importedConfig := `severity_threshold: info
`
	err := os.WriteFile(importedPath, []byte(importedConfig), 0o644)
	require.NoError(t, err)

	// Main config imports the other file
	mainConfig := `version: 1
imports:
  - imported.yaml
`
	err = os.WriteFile(configPath, []byte(mainConfig), 0o644)
	require.NoError(t, err)

	// Initialize server
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Verify initial config loaded from import (use mutex to avoid race)
	server.engineMu.RLock()
	initialThreshold := server.config.SeverityThreshold
	server.engineMu.RUnlock()
	require.Equal(t, "info", initialThreshold)

	// Verify both files are being watched
	server.configWatcherMu.Lock()
	watchedFiles := server.configFiles
	server.configWatcherMu.Unlock()
	require.Len(t, watchedFiles, 2, "should watch main config and imported file")

	// Modify imported config file
	updatedImport := `severity_threshold: warning
`
	err = os.WriteFile(importedPath, []byte(updatedImport), 0o644)
	require.NoError(t, err)

	// Wait for fsnotify to detect and process the change (use Eventually for reliability)
	require.Eventually(t, func() bool {
		server.engineMu.RLock()
		threshold := server.config.SeverityThreshold
		server.engineMu.RUnlock()
		return threshold == "warning"
	}, 2*time.Second, 10*time.Millisecond, "config should reload with new severity from imported file")
}

// TestServer_ConfigReload_EnginesReinitialized tests that when config is reloaded,
// the engines are reinitialized with the new configuration.
func TestServer_ConfigReload_EnginesReinitialized(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Initial config with style enabled and a rule configured
	initialConfig := `version: 1
engines:
  style:
    enabled: true
    rules:
      style.blank-line-between-blocks:
        enabled: true
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err)

	// Create a test document
	tfFile := filepath.Join(tmpDir, "main.tf")
	err = os.WriteFile(tfFile, []byte(`resource "null" "a" {}
resource "null" "b" {}`), 0o644)
	require.NoError(t, err)

	// Initialize server
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Verify style engine is initialized (use mutex to avoid race)
	server.engineMu.RLock()
	require.NotNil(t, server.styleEngine)
	// Save reference to old style engine
	oldStyleEngine := server.styleEngine
	server.engineMu.RUnlock()

	// Update config to change style rule settings
	updatedConfig := `version: 1
engines:
  style:
    enabled: true
    rules:
      style.blank-line-between-blocks:
        enabled: false
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Wait for reload (use Eventually for reliability)
	require.Eventually(t, func() bool {
		server.engineMu.RLock()
		newStyleEngine := server.styleEngine
		server.engineMu.RUnlock()
		return newStyleEngine != oldStyleEngine
	}, 2*time.Second, 10*time.Millisecond, "style engine should be reinitialized")
}

// TestServer_ConfigReload_Debounce tests that rapid config file changes are debounced.
func TestServer_ConfigReload_Debounce(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	logFile := filepath.Join(tmpDir, "server.log")

	// Initial config
	initialConfig := `version: 1
severity_threshold: info
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err)

	// Initialize server with log file to capture reload messages
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)
	require.NoError(t, server.SetLogFile(logFile))
	// Use longer debounce delay for test reliability
	server.configReloadDelay = 200 * time.Millisecond

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Write to config file multiple times rapidly (simulating editor save with multiple events)
	// All writes happen within the 200ms debounce window
	for i := range 5 {
		config := fmt.Sprintf(`version: 1
severity_threshold: warning
# change %d
`, i)
		err = os.WriteFile(configPath, []byte(config), 0o644)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond) // Small delay between writes, total ~100ms
	}

	// Wait for debounced reload to complete
	require.Eventually(t, func() bool {
		server.engineMu.RLock()
		threshold := server.config.SeverityThreshold
		server.engineMu.RUnlock()
		return threshold == "warning"
	}, 2*time.Second, 10*time.Millisecond, "config should reload with final value")

	// Verify that only one "config reloaded successfully" message appears
	// (The debouncing should coalesce the 5 writes into 1 reload)
	logContent, err := os.ReadFile(logFile)
	require.NoError(t, err)
	reloadCount := strings.Count(string(logContent), "config reloaded successfully")
	assert.Equal(t, 1, reloadCount, "should have exactly one reload for multiple rapid writes, got log: %s", string(logContent))
}

// TestServer_DidChangeConfiguration tests that workspace/didChangeConfiguration
// updates server settings and triggers a config reload.
func TestServer_DidChangeConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Initial config with info threshold
	initialConfig := `version: 1
severity_threshold: info
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err)

	// Initialize server
	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			SeverityThreshold: "info",
		},
	}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Verify initial threshold
	server.engineMu.RLock()
	initialThreshold := server.initOptions.SeverityThreshold
	server.engineMu.RUnlock()
	assert.Equal(t, "info", initialThreshold)

	// Send didChangeConfiguration with new settings
	changeParams := DidChangeConfigurationParams{
		Settings: &InitializationOptions{
			SeverityThreshold: "warning",
			Profile:           "production",
		},
	}
	changeParamsJSON, _ := json.Marshal(changeParams)
	changeMsg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "workspace/didChangeConfiguration",
		Params:  changeParamsJSON,
	}

	err = server.handleDidChangeConfiguration(changeMsg)
	require.NoError(t, err)

	// Verify initOptions were updated
	server.engineMu.RLock()
	newThreshold := server.initOptions.SeverityThreshold
	newProfile := server.initOptions.Profile
	// Also verify that reloadConfig was called and config was updated
	configThreshold := server.config.SeverityThreshold
	server.engineMu.RUnlock()

	assert.Equal(t, "warning", newThreshold, "initOptions severity threshold should be updated")
	assert.Equal(t, "production", newProfile, "initOptions profile should be updated")
	assert.Equal(t, "warning", configThreshold, "config severity threshold should be updated via reloadConfig")
}

// TestServer_DidChangeConfiguration_NilSettings tests that nil settings are handled gracefully.
func TestServer_DidChangeConfiguration_NilSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644)
	require.NoError(t, err)

	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Send didChangeConfiguration with nil settings
	changeParams := DidChangeConfigurationParams{Settings: nil}
	changeParamsJSON, _ := json.Marshal(changeParams)
	changeMsg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "workspace/didChangeConfiguration",
		Params:  changeParamsJSON,
	}

	// Should not error
	err = server.handleDidChangeConfiguration(changeMsg)
	assert.NoError(t, err, "nil settings should be handled gracefully")
}

// TestServer_DidChangeConfiguration_InvalidJSON tests that invalid JSON params return an error.
func TestServer_DidChangeConfiguration_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644)
	require.NoError(t, err)

	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{RootURI: pathToFileURI(tmpDir)}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Send didChangeConfiguration with invalid JSON
	changeMsg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "workspace/didChangeConfiguration",
		Params:  json.RawMessage(`{invalid json`),
	}

	// Should return an error
	err = server.handleDidChangeConfiguration(changeMsg)
	require.Error(t, err, "invalid JSON should return an error")
	assert.Contains(t, err.Error(), "parsing didChangeConfiguration params")
}

// TestServer_DidChangeConfiguration_EngineToggles tests that engine toggles are updated.
func TestServer_DidChangeConfiguration_EngineToggles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644)
	require.NoError(t, err)

	out := &safeBuffer{}
	server := NewServer(strings.NewReader(""), out)

	params := InitializeParams{
		RootURI: pathToFileURI(tmpDir),
		InitializationOptions: &InitializationOptions{
			Engines: EngineToggles{
				Style: true,
				Lint:  true,
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)
	msg := RequestMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	err = server.handleInitialize(msg)
	require.NoError(t, err)
	defer server.Close()

	// Send didChangeConfiguration to disable lint
	changeParams := DidChangeConfigurationParams{
		Settings: &InitializationOptions{
			Engines: EngineToggles{
				Style: true,
				Lint:  false,
			},
		},
	}
	changeParamsJSON, _ := json.Marshal(changeParams)
	changeMsg := RequestMessage{
		JSONRPC: "2.0",
		Method:  "workspace/didChangeConfiguration",
		Params:  changeParamsJSON,
	}

	err = server.handleDidChangeConfiguration(changeMsg)
	require.NoError(t, err)

	// Verify engine toggles were updated
	server.engineMu.RLock()
	engines := server.initOptions.Engines
	server.engineMu.RUnlock()

	assert.True(t, engines.Style, "style should remain enabled")
	assert.False(t, engines.Lint, "lint should be disabled")
}
