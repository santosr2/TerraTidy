package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// FuzzLSPMessage tests LSP message handling with arbitrary JSON bodies.
// The header is always well-formed; this exercises handleMessage dispatch and JSON parsing.
// For header parsing specifically, see FuzzLSPReadMessage.
func FuzzLSPMessage(f *testing.F) {
	// Valid message seeds
	f.Add([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"shutdown","id":2}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/didChange"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/didClose"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/didSave"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/formatting","id":3}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"textDocument/codeAction","id":4}`))

	// Edge cases
	f.Add([]byte{})                                              // Empty
	f.Add([]byte(`{}`))                                          // Empty object
	f.Add([]byte(`{"method":""}`))                               // Empty method
	f.Add([]byte(`null`))                                        // JSON null
	f.Add([]byte(`[]`))                                          // Array instead of object
	f.Add([]byte(`"string"`))                                    // Just a string
	f.Add([]byte(`123`))                                         // Just a number
	f.Add([]byte(`{`))                                           // Invalid JSON
	f.Add([]byte(`{"jsonrpc`))                                   // Truncated JSON
	f.Add([]byte(`{"jsonrpc":"2.0","method":"unknown/method"}`)) // Unknown method

	// Special characters in method names
	f.Add([]byte(`{"method":"../../../etc/passwd"}`))
	f.Add([]byte(`{"method":"<script>alert(1)</script>"}`))
	f.Add([]byte(`{"method":"\u0000\u0001\u0002"}`))

	// Unicode
	f.Add([]byte(`{"jsonrpc":"2.0","method":"日本語"}`))

	// Large ID values
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","id":999999999999}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","id":"string-id"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","id":null}`))

	// With params
	f.Add([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":"invalid"}`))

	f.Fuzz(func(t *testing.T, content []byte) {
		// Wrap content in proper LSP message format
		header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
		input := header + string(content)

		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(input), out)

		// Read message - should not panic
		msg, err := server.readMessage()
		if err != nil {
			return // Reading failed, which is acceptable
		}

		// Handle message - should not panic
		_ = server.handleMessage(msg)
	})
}

// FuzzLSPDidChange tests the textDocument/didChange handler with arbitrary content.
// Exercises incremental text synchronization code paths.
func FuzzLSPDidChange(f *testing.F) {
	// Valid didChange params
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"text":"resource \"test\" {}"}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"text":"hello"}]}`))

	// Multiple content changes
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":3},"contentChanges":[{"text":"a"},{"text":"b"},{"text":"c"}]}`))

	// Edge cases
	f.Add([]byte{})                                                  // Empty
	f.Add([]byte(`{}`))                                              // Empty object
	f.Add([]byte(`{"textDocument":{}}`))                             // Missing fields
	f.Add([]byte(`{"contentChanges":[]}`))                           // Missing textDocument
	f.Add([]byte(`{"textDocument":{"uri":""},"contentChanges":[]}`)) // Empty URI

	// Invalid URIs
	f.Add([]byte(`{"textDocument":{"uri":"not-a-uri","version":1},"contentChanges":[{"text":""}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file://","version":1},"contentChanges":[{"text":""}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"http://example.com/test.tf","version":1},"contentChanges":[{"text":""}]}`))

	// Large content
	largeContent := strings.Repeat("resource \"test\" \"r\" {}\n", 1000)
	f.Add([]byte(fmt.Sprintf(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"text":%s}]}`, jsonQuote(largeContent))))

	// Special characters in content
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"text":"<script>alert(1)</script>"}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"text":"\u0000\u0001\u0002"}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"text":"日本語テスト"}]}`))

	// Invalid range values
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"range":{"start":{"line":-1,"character":-1},"end":{"line":-1,"character":-1}},"text":"x"}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":1},"contentChanges":[{"range":{"start":{"line":999999,"character":999999},"end":{"line":999999,"character":999999}},"text":"x"}]}`))

	// Version edge cases
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":0},"contentChanges":[{"text":""}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":-1},"contentChanges":[{"text":""}]}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf","version":9999999999},"contentChanges":[{"text":""}]}`))

	f.Fuzz(func(t *testing.T, paramsData []byte) {
		// Build a valid LSP didChange request with fuzzed params
		msg := RequestMessage{
			JSONRPC: "2.0",
			Method:  "textDocument/didChange",
			Params:  json.RawMessage(paramsData),
		}

		content, err := json.Marshal(msg)
		if err != nil {
			return
		}

		header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
		input := header + string(content)

		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(input), out)

		// Initialize server (required for message handling)
		// NewServer already creates documents map, just set initialized flag
		server.initialized = true

		// Read and handle the message - should not panic
		msgBytes, err := server.readMessage()
		if err != nil {
			return
		}
		_ = server.handleMessage(msgBytes)
	})
}

// jsonQuote properly JSON-quotes a string for embedding in JSON.
// json.Marshal on a plain string cannot fail, so the error is ignored.
func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// FuzzLSPReadMessage tests the Content-Length header parsing with arbitrary input.
// This specifically tests the header parsing logic.
func FuzzLSPReadMessage(f *testing.F) {
	// Valid headers (first seed has trailing null to match length=10, tests trailing garbage)
	f.Add("Content-Length: 10\r\n\r\n" + `{"x":"yy"}`)
	f.Add("Content-Length: 2\r\n\r\n{}")
	f.Add("content-length: 2\r\n\r\n{}") // lowercase (some clients)
	f.Add("Content-Length: 0\r\n\r\n")   // Zero length

	// Multiple headers
	f.Add("Content-Length: 2\r\nContent-Type: application/json\r\n\r\n{}")

	// Edge cases
	f.Add("")                                        // Empty
	f.Add("\r\n")                                    // Just separator
	f.Add("Content-Length:\r\n\r\n")                 // Empty length
	f.Add("Content-Length: abc\r\n\r\n")             // Non-numeric
	f.Add("Content-Length: -1\r\n\r\n")              // Negative
	f.Add("Content-Length: 999999999999999\r\n\r\n") // Overflow
	f.Add("Content-Length: 10\r\n\r\n")              // Length mismatch (content shorter)

	// Malformed headers
	f.Add("Content-Length 10\r\n\r\n{}") // Missing colon
	f.Add("Content-Length:10\r\n\r\n{}") // No space after colon
	f.Add("Content-Length: 10\n\n{}")    // LF only (not CRLF)

	// Injection attempts
	f.Add("Content-Length: 2\r\nX-Injected: value\r\n\r\n{}")

	f.Fuzz(func(t *testing.T, input string) {
		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(input), out)

		// Should not panic regardless of input
		_, _ = server.readMessage()
	})
}

// FuzzLSPFormatting tests the textDocument/formatting handler.
func FuzzLSPFormatting(f *testing.F) {
	// Valid formatting params
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf"},"options":{"tabSize":2,"insertSpaces":true}}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf"},"options":{"tabSize":4,"insertSpaces":false}}`))

	// Edge cases
	f.Add([]byte{})
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"textDocument":{}}`))
	f.Add([]byte(`{"options":{}}`))

	// Invalid values
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf"},"options":{"tabSize":-1}}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf"},"options":{"tabSize":999999}}`))
	f.Add([]byte(`{"textDocument":{"uri":"file:///test.tf"},"options":{"insertSpaces":"yes"}}`))

	f.Fuzz(func(t *testing.T, paramsData []byte) {
		msg := RequestMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "textDocument/formatting",
			Params:  json.RawMessage(paramsData),
		}

		content, err := json.Marshal(msg)
		if err != nil {
			return
		}

		header := "Content-Length: " + strconv.Itoa(len(content)) + "\r\n\r\n"
		input := header + string(content)

		out := &bytes.Buffer{}
		server := NewServer(strings.NewReader(input), out)
		server.initialized = true
		// Pre-seed a document so formatting path is exercised (not just early-return)
		server.documents["file:///test.tf"] = &Document{
			URI:     "file:///test.tf",
			Content: `resource "test" "example" {}`,
			Version: 1,
		}

		msgBytes, err := server.readMessage()
		if err != nil {
			return
		}
		_ = server.handleMessage(msgBytes)
	})
}
