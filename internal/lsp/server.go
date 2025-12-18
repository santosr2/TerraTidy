// Package lsp implements a Language Server Protocol server for TerraTidy.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/santosr2/terratidy/internal/config"
	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Server represents an LSP server instance
type Server struct {
	reader        *bufio.Reader
	writer        io.Writer
	config        *config.Config
	documents     map[string]*Document
	docMu         sync.RWMutex
	lintEngine    *lint.Engine
	styleEngine   *style.Engine
	workspaceRoot string
	initialized   bool
	shutdown      bool
}

// Document represents an open document
type Document struct {
	URI     string
	Content string
	Version int
}

// NewServer creates a new LSP server
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		reader:    bufio.NewReader(in),
		writer:    out,
		documents: make(map[string]*Document),
	}
}

// Run starts the LSP server main loop
func (s *Server) Run() error {
	for {
		if s.shutdown {
			return nil
		}

		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading message: %w", err)
		}

		if err := s.handleMessage(msg); err != nil {
			// Log error but continue processing
			fmt.Fprintf(os.Stderr, "Error handling message: %v\n", err)
		}
	}
}

// readMessage reads an LSP message from stdin
func (s *Server) readMessage() (json.RawMessage, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid content length: %w", err)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("no content length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, content)
	if err != nil {
		return nil, fmt.Errorf("reading content: %w", err)
	}

	return content, nil
}

// writeMessage writes an LSP message to stdout
func (s *Server) writeMessage(msg interface{}) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	if _, err := io.WriteString(s.writer, header); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := s.writer.Write(content); err != nil {
		return fmt.Errorf("writing content: %w", err)
	}

	return nil
}

// handleMessage processes an incoming LSP message
func (s *Server) handleMessage(content json.RawMessage) error {
	var msg RequestMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return fmt.Errorf("parsing message: %w", err)
	}

	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return s.handleInitialized(msg)
	case "shutdown":
		return s.handleShutdown(msg)
	case "exit":
		return s.handleExit()
	case "textDocument/didOpen":
		return s.handleDidOpen(msg)
	case "textDocument/didChange":
		return s.handleDidChange(msg)
	case "textDocument/didClose":
		return s.handleDidClose(msg)
	case "textDocument/didSave":
		return s.handleDidSave(msg)
	case "textDocument/formatting":
		return s.handleFormatting(msg)
	case "textDocument/codeAction":
		return s.handleCodeAction(msg)
	default:
		// Unknown method - respond with method not found for requests
		if msg.ID != nil {
			return s.sendError(msg.ID, -32601, fmt.Sprintf("Method not found: %s", msg.Method))
		}
		return nil
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(msg RequestMessage) error {
	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	// Store workspace root
	if params.RootURI != "" {
		s.workspaceRoot = uriToPath(params.RootURI)
	} else if params.RootPath != "" {
		s.workspaceRoot = params.RootPath
	}

	// Load configuration
	configPath := filepath.Join(s.workspaceRoot, ".terratidy.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	s.config = cfg

	// Initialize engines
	s.lintEngine = lint.New(nil)
	s.styleEngine = style.New(nil)

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full sync
				Save:      &SaveOptions{IncludeText: true},
			},
			DocumentFormattingProvider: true,
			CodeActionProvider:         true,
			DiagnosticProvider: &DiagnosticOptions{
				InterFileDependencies: false,
				WorkspaceDiagnostics:  false,
			},
		},
		ServerInfo: &ServerInfo{
			Name:    "terratidy-lsp",
			Version: "0.1.0",
		},
	}

	return s.sendResult(msg.ID, result)
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(msg RequestMessage) error {
	s.initialized = true
	return nil
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(msg RequestMessage) error {
	s.shutdown = true
	return s.sendResult(msg.ID, nil)
}

// handleExit handles the exit notification
func (s *Server) handleExit() error {
	if s.shutdown {
		os.Exit(0)
	}
	os.Exit(1)
	return nil
}

// handleDidOpen handles textDocument/didOpen notification
func (s *Server) handleDidOpen(msg RequestMessage) error {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didOpen params: %w", err)
	}

	s.docMu.Lock()
	s.documents[params.TextDocument.URI] = &Document{
		URI:     params.TextDocument.URI,
		Content: params.TextDocument.Text,
		Version: params.TextDocument.Version,
	}
	s.docMu.Unlock()

	// Run diagnostics
	return s.publishDiagnostics(params.TextDocument.URI)
}

// handleDidChange handles textDocument/didChange notification
func (s *Server) handleDidChange(msg RequestMessage) error {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didChange params: %w", err)
	}

	s.docMu.Lock()
	if doc, ok := s.documents[params.TextDocument.URI]; ok {
		for _, change := range params.ContentChanges {
			doc.Content = change.Text
		}
		doc.Version = params.TextDocument.Version
	}
	s.docMu.Unlock()

	// Run diagnostics
	return s.publishDiagnostics(params.TextDocument.URI)
}

// handleDidClose handles textDocument/didClose notification
func (s *Server) handleDidClose(msg RequestMessage) error {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didClose params: %w", err)
	}

	s.docMu.Lock()
	delete(s.documents, params.TextDocument.URI)
	s.docMu.Unlock()

	// Clear diagnostics
	return s.writeMessage(NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []Diagnostic{},
		},
	})
}

// handleDidSave handles textDocument/didSave notification
func (s *Server) handleDidSave(msg RequestMessage) error {
	var params DidSaveTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didSave params: %w", err)
	}

	// Update content if included
	if params.Text != "" {
		s.docMu.Lock()
		if doc, ok := s.documents[params.TextDocument.URI]; ok {
			doc.Content = params.Text
		}
		s.docMu.Unlock()
	}

	return s.publishDiagnostics(params.TextDocument.URI)
}

// handleFormatting handles textDocument/formatting request
func (s *Server) handleFormatting(msg RequestMessage) error {
	var params DocumentFormattingParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	s.docMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.docMu.RUnlock()

	if !ok {
		return s.sendResult(msg.ID, nil)
	}

	// For now, return empty edits (formatting would require hclwrite)
	// A full implementation would format the content and return TextEdit[]
	_ = doc
	return s.sendResult(msg.ID, []TextEdit{})
}

// handleCodeAction handles textDocument/codeAction request
func (s *Server) handleCodeAction(msg RequestMessage) error {
	var params CodeActionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	// Return code actions for fixable diagnostics
	var actions []CodeAction
	for _, diag := range params.Context.Diagnostics {
		if diag.Code != "" {
			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Fix: %s", diag.Code),
				Kind:        "quickfix",
				Diagnostics: []Diagnostic{diag},
				// Command or Edit would be set for actual fixes
			})
		}
	}

	return s.sendResult(msg.ID, actions)
}

// publishDiagnostics runs TerraTidy and publishes diagnostics
func (s *Server) publishDiagnostics(uri string) error {
	s.docMu.RLock()
	doc, ok := s.documents[uri]
	s.docMu.RUnlock()

	if !ok {
		return nil
	}

	filePath := uriToPath(uri)

	// Only process .tf and .hcl files
	ext := filepath.Ext(filePath)
	if ext != ".tf" && ext != ".hcl" && ext != ".tfvars" {
		return nil
	}

	// Write content to temp file for analysis
	tempFile, err := os.CreateTemp("", "terratidy-*.tf")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(doc.Content); err != nil {
		tempFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	tempFile.Close()

	// Run lint and style checks
	ctx := context.Background()
	var findings []sdk.Finding

	if s.lintEngine != nil {
		lintFindings, err := s.lintEngine.Run(ctx, []string{tempFile.Name()})
		if err == nil {
			findings = append(findings, lintFindings...)
		}
	}

	if s.styleEngine != nil {
		styleFindings, err := s.styleEngine.Run(ctx, []string{tempFile.Name()})
		if err == nil {
			findings = append(findings, styleFindings...)
		}
	}

	// Convert findings to diagnostics
	diagnostics := make([]Diagnostic, 0, len(findings))
	for _, f := range findings {
		diag := Diagnostic{
			Range: Range{
				Start: Position{
					Line:      max(0, f.Location.Start.Line-1),
					Character: max(0, f.Location.Start.Column-1),
				},
				End: Position{
					Line:      max(0, f.Location.End.Line-1),
					Character: max(0, f.Location.End.Column-1),
				},
			},
			Severity: severityToLSP(f.Severity),
			Code:     f.Rule,
			Source:   "terratidy",
			Message:  f.Message,
		}
		diagnostics = append(diagnostics, diag)
	}

	return s.writeMessage(NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	})
}

// sendResult sends a successful response
func (s *Server) sendResult(id json.RawMessage, result interface{}) error {
	return s.writeMessage(ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// sendError sends an error response
func (s *Server) sendError(id json.RawMessage, code int, message string) error {
	return s.writeMessage(ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
		},
	})
}

// uriToPath converts a file URI to a file path
func uriToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

// severityToLSP converts SDK severity to LSP diagnostic severity
func severityToLSP(severity sdk.Severity) int {
	switch severity {
	case sdk.SeverityError:
		return 1 // Error
	case sdk.SeverityWarning:
		return 2 // Warning
	case sdk.SeverityInfo:
		return 3 // Information
	default:
		return 4 // Hint
	}
}
