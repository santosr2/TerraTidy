// Package lsp implements a Language Server Protocol server for TerraTidy.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/santosr2/terratidy/internal/buildinfo"
	"github.com/santosr2/terratidy/internal/config"
	"github.com/santosr2/terratidy/internal/engines/format"
	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// LogLevel represents the logging verbosity
type LogLevel int

// LogLevel values control server logging verbosity.
const (
	LogLevelOff LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "error":
		return LogLevelError
	case "warn", "warning":
		return LogLevelWarn
	case "info":
		return LogLevelInfo
	case "debug":
		return LogLevelDebug
	case "off":
		return LogLevelOff
	default:
		return LogLevelInfo
	}
}

// Server represents an LSP server instance
type Server struct {
	reader        *bufio.Reader
	writer        io.Writer
	writeMu       sync.Mutex // protects writer from concurrent writeMessage calls
	config        *config.Config
	initOptions   *InitializationOptions
	documents     map[string]*Document
	docMu         sync.RWMutex
	lintEngine    *lint.Engine
	styleEngine   *style.Engine
	workspaceRoot string
	initialized   bool
	shutdown      bool
	logger        *log.Logger
	logLevel      LogLevel
	logFile       *os.File
}

// Document represents an open document
type Document struct {
	URI      string
	Content  string
	Version  int
	tempFile string // cached temp file path for analysis
}

// NewServer creates a new LSP server
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		reader:    bufio.NewReader(in),
		writer:    out,
		documents: make(map[string]*Document),
		logger:    log.New(os.Stderr, "terratidy-lsp: ", log.Ltime),
		logLevel:  LogLevelInfo,
	}
}

// SetLogLevel sets the logging verbosity
func (s *Server) SetLogLevel(level LogLevel) {
	s.logLevel = level
}

// SetLogFile redirects log output to a file (in addition to or instead of stderr)
func (s *Server) SetLogFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	s.logFile = f
	s.logger.SetOutput(f)
	return nil
}

// Close releases resources held by the server (e.g., log file handle)
func (s *Server) Close() error {
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}

func (s *Server) logDebug(format string, args ...any) {
	if s.logLevel >= LogLevelDebug {
		s.logger.Printf("[DEBUG] "+format, args...)
	}
}

func (s *Server) logInfo(format string, args ...any) {
	if s.logLevel >= LogLevelInfo {
		s.logger.Printf("[INFO] "+format, args...)
	}
}

func (s *Server) logError(format string, args ...any) {
	if s.logLevel >= LogLevelError {
		s.logger.Printf("[ERROR] "+format, args...)
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
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading message: %w", err)
		}

		if err := s.handleMessage(msg); err != nil {
			// Log error but continue processing
			s.logError("handling message: %v", err)
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

// writeMessage writes an LSP message to stdout.
// The mutex ensures concurrent publishDiagnostics calls don't interleave
// the header and content of different messages.
func (s *Server) writeMessage(msg any) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

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

	// Store initialization options from the client
	s.initOptions = params.InitializationOptions

	// Load configuration, preferring client-provided config path
	configPath := filepath.Join(s.workspaceRoot, ".terratidy.yaml")
	if s.initOptions != nil && s.initOptions.ConfigPath != "" {
		configPath = s.initOptions.ConfigPath
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Apply profile from client options
	if s.initOptions != nil && s.initOptions.Profile != "" {
		if profileErr := cfg.ApplyProfile(s.initOptions.Profile); profileErr != nil {
			// Profile not found is not fatal; log and continue
			s.logInfo("profile %q not found: %v", s.initOptions.Profile, profileErr)
		}
	}

	// Apply severity threshold from client options
	if s.initOptions != nil && s.initOptions.SeverityThreshold != "" {
		cfg.SeverityThreshold = s.initOptions.SeverityThreshold
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
			Version: buildinfo.GetVersion(),
		},
	}

	return s.sendResult(msg.ID, result)
}

// handleInitialized handles the initialized notification.
func (s *Server) handleInitialized(_ RequestMessage) error {
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
	if doc, ok := s.documents[params.TextDocument.URI]; ok {
		if doc.tempFile != "" {
			_ = os.Remove(doc.tempFile)
		}
	}
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

	// Format the document content using hclwrite
	original := doc.Content
	formatted := string(format.Format([]byte(original)))

	// If content is unchanged, return empty edits
	if formatted == original {
		return s.sendResult(msg.ID, []TextEdit{})
	}

	// Return a single edit replacing the entire document
	lines := strings.Count(original, "\n")
	lastLineLen := len(original)
	if idx := strings.LastIndex(original, "\n"); idx >= 0 {
		lastLineLen = len(original) - idx - 1
	}

	edits := []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: lines, Character: lastLineLen},
			},
			NewText: formatted,
		},
	}
	return s.sendResult(msg.ID, edits)
}

// handleCodeAction handles textDocument/codeAction request
func (s *Server) handleCodeAction(msg RequestMessage) error {
	var params CodeActionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	uri := params.TextDocument.URI

	s.docMu.RLock()
	doc, ok := s.documents[uri]
	s.docMu.RUnlock()

	if !ok || len(params.Context.Diagnostics) == 0 {
		return s.sendResult(msg.ID, []CodeAction{})
	}

	// Compute the formatted version once for all fix actions
	original := doc.Content
	formatted := string(format.Format([]byte(original)))
	hasFormatFix := formatted != original

	var actions []CodeAction
	for i := range params.Context.Diagnostics {
		if params.Context.Diagnostics[i].Code == "" {
			continue
		}

		// Offer a format-based fix for diagnostics when formatting changes the file
		if hasFormatFix {
			lines := strings.Count(original, "\n")
			lastLineLen := len(original)
			if idx := strings.LastIndex(original, "\n"); idx >= 0 {
				lastLineLen = len(original) - idx - 1
			}

			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Fix: %s", params.Context.Diagnostics[i].Code),
				Kind:        "quickfix",
				Diagnostics: []Diagnostic{params.Context.Diagnostics[i]},
				IsPreferred: true,
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						uri: {
							{
								Range: Range{
									Start: Position{Line: 0, Character: 0},
									End:   Position{Line: lines, Character: lastLineLen},
								},
								NewText: formatted,
							},
						},
					},
				},
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

	// Reuse a temp file per document to avoid creating one per keystroke
	if doc.tempFile == "" {
		tmpExt := filepath.Ext(filePath)
		if tmpExt == "" {
			tmpExt = ".tf"
		}
		f, err := os.CreateTemp("", "terratidy-*"+tmpExt)
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		doc.tempFile = f.Name()
		_ = f.Close()
	}

	if err := os.WriteFile(doc.tempFile, []byte(doc.Content), 0o600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Run lint and style checks
	ctx := context.Background()
	var findings []sdk.Finding

	if s.lintEngine != nil {
		lintFindings, err := s.lintEngine.Run(ctx, []string{doc.tempFile})
		if err == nil {
			findings = append(findings, lintFindings...)
		}
	}

	if s.styleEngine != nil {
		styleFindings, err := s.styleEngine.Run(ctx, []string{doc.tempFile})
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
func (s *Server) sendResult(id json.RawMessage, result any) error {
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

// uriToPath converts a file URI to a file path.
// Handles Windows paths correctly (file:///C:/path becomes C:/path).
func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		// Fallback: strip prefix manually
		return strings.TrimPrefix(uri, "file://")
	}
	// On Windows, url.Parse("file:///C:/path") gives Path="/C:/path".
	// Strip the leading slash before a drive letter.
	p := parsed.Path
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return p
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
