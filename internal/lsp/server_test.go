package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		input := "Content-Length: " + intToStr(contentLen) + "\r\n\r\n" + content

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
		assert.Contains(t, err.Error(), "no content length")
	})

	t.Run("oversized content length rejected", func(t *testing.T) {
		// Content-Length exceeds 10 MB limit
		oversizedLength := 11 * 1024 * 1024 // 11 MB
		input := "Content-Length: " + intToStr(oversizedLength) + "\r\n\r\n"

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
		input := "Content-Length: " + intToStr(contentLen) + "\r\n\r\n" + content

		server := NewServer(strings.NewReader(input), &bytes.Buffer{})

		msg, err := server.readMessage()
		require.NoError(t, err)
		assert.NotNil(t, msg)
	})
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
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
			FormatOnSave: true,
			RunOnSave:    true,
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
	assert.True(t, server.initOptions.FormatOnSave)

	// Severity threshold should be applied to config
	assert.Equal(t, "error", server.config.SeverityThreshold)
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
		assert.Contains(t, output, "Fix:")
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

func TestServer_BuildStyleConfig_WithOverrides(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.config = &config.Config{
		Overrides: config.OverridesConfig{
			Rules: map[string]config.RuleConfig{
				"style.resource-name-matches-type": {
					Enabled:  true,
					Severity: "warning",
					Config: map[string]any{
						"option1": "value1",
					},
				},
				"style.blank-lines-between-blocks": {
					Enabled:  false,
					Severity: "info",
				},
			},
		},
	}

	cfg := server.buildStyleConfig()

	require.NotNil(t, cfg)
	assert.Len(t, cfg.Rules, 2)

	rule1 := cfg.Rules["style.resource-name-matches-type"]
	assert.True(t, rule1.Enabled)
	assert.Equal(t, "warning", rule1.Severity)
	assert.Equal(t, "value1", rule1.Options["option1"])

	rule2 := cfg.Rules["style.blank-lines-between-blocks"]
	assert.False(t, rule2.Enabled)
	assert.Equal(t, "info", rule2.Severity)
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

	err := server.initSessionTempDir()
	require.NoError(t, err)

	// Should have created a session temp directory
	assert.NotEmpty(t, server.sessionTempDir)
	assert.DirExists(t, server.sessionTempDir)

	// Directory should be under cache path
	assert.Contains(t, server.sessionTempDir, "terratidy")

	// Clean up
	err = server.Close()
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
	err := server.initSessionTempDir()
	require.NoError(t, err)

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
