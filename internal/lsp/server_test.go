package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		{"file:///tmp/test.tf", "/tmp/test.tf"},
		{"file:///home/user/main.tf", "/home/user/main.tf"},
		{"/direct/path.tf", "/direct/path.tf"},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			result := uriToPath(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
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

func TestServer_HandleFormatting(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Add a document
	server.docMu.Lock()
	server.documents["file:///test.tf"] = &Document{
		URI:     "file:///test.tf",
		Content: "resource {}",
		Version: 1,
	}
	server.docMu.Unlock()

	params := DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test.tf",
		},
		Options: FormattingOptions{
			TabSize:      2,
			InsertSpaces: true,
		},
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
	assert.Contains(t, output, `"result"`)
}

func TestServer_HandleCodeAction(t *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file:///test.tf",
		},
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 10},
		},
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
	assert.Contains(t, output, `"result"`)
	assert.Contains(t, output, "Fix:")
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

func TestServer_HandleMessage_BeforeInitialize(_ *testing.T) {
	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)

	// Try to send textDocument/didOpen before initialize
	msgJSON := `{
		"jsonrpc": "2.0",
		"method": "textDocument/didOpen",
		"params": {}
	}`

	err := server.handleMessage(json.RawMessage(msgJSON))
	// Should handle gracefully (may return error or nil)
	// The important thing is it doesn't panic
	_ = err
}
