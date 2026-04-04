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
	"time"

	"github.com/santosr2/TerraTidy/internal/buildinfo"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// lspTempBasePath is the base directory for LSP session temp files.
// Uses XDG cache directory structure.
const lspTempBasePath = ".cache/terratidy/lsp-tmp"

// sessionTempMaxAge is the maximum age of session temp directories before cleanup.
// Directories older than this are removed on server start.
const sessionTempMaxAge = 24 * time.Hour

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

// maxContentLength is the maximum allowed Content-Length for LSP messages (10 MB).
// This prevents denial-of-service via memory exhaustion from malicious clients.
const maxContentLength = 10 * 1024 * 1024

// maxDocumentSize is the maximum allowed size for a single document (10 MB).
// Documents larger than this will be rejected on didOpen/didChange.
const maxDocumentSize = maxContentLength

// maxDocuments is the maximum number of open documents the server will track.
// This prevents memory exhaustion from opening too many files.
const maxDocuments = 1000

// maxConcurrentDiagnostics is the maximum number of concurrent diagnostic computations.
// This prevents CPU exhaustion from computing diagnostics for many files simultaneously.
const maxConcurrentDiagnostics = 10

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
	reader         *bufio.Reader
	writer         io.Writer
	writeMu        sync.Mutex // protects writer from concurrent writeMessage calls
	config         *config.Config
	initOptions    *InitializationOptions
	documents      map[string]*Document
	docMu          sync.RWMutex
	lintEngine     *lint.Engine
	styleEngine    *style.Engine
	workspaceRoot  string
	initialized    bool
	shutdown       bool
	logger         *log.Logger
	logLevel       LogLevel
	logFile        *os.File
	diagSem        chan struct{} // semaphore for concurrent diagnostics
	sessionTempDir string        // private temp directory for this session
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
		diagSem:   make(chan struct{}, maxConcurrentDiagnostics),
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
	// Clean up session temp directory
	if s.sessionTempDir != "" {
		_ = os.RemoveAll(s.sessionTempDir)
	}
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}

// initSessionTempDir creates a private temp directory for this server session.
// It also cleans up stale session directories older than sessionTempMaxAge.
func (s *Server) initSessionTempDir() error {
	baseDir := getSessionTempBaseDir()

	// Clean up old session directories
	s.cleanupOldSessions(baseDir)

	// Create session directory with PID + timestamp
	sessionID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	sessionDir := filepath.Join(baseDir, sessionID)

	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		// Fall back to system temp dir
		s.logWarn("failed to create session temp dir %s, using system temp: %v", sessionDir, err)
		s.sessionTempDir = os.TempDir()
		return nil
	}

	s.sessionTempDir = sessionDir
	s.logDebug("using session temp directory: %s", sessionDir)
	return nil
}

// getSessionTempBaseDir returns the base directory for LSP session temp files.
// Uses XDG cache directory (~/.cache/terratidy/lsp-tmp).
func getSessionTempBaseDir() string {
	// Try XDG_CACHE_HOME first
	if cacheDir := os.Getenv("XDG_CACHE_HOME"); cacheDir != "" {
		return filepath.Join(cacheDir, "terratidy", "lsp-tmp")
	}

	// Fall back to ~/.cache
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, lspTempBasePath)
	}

	// Last resort: system temp
	return filepath.Join(os.TempDir(), "terratidy-lsp")
}

// cleanupOldSessions removes session temp directories older than sessionTempMaxAge.
// This prevents accumulation of stale temp files from crashed sessions.
func (s *Server) cleanupOldSessions(baseDir string) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return // Directory doesn't exist or can't be read
	}

	cutoff := time.Now().Add(-sessionTempMaxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove directories older than cutoff
		if info.ModTime().Before(cutoff) {
			sessionPath := filepath.Join(baseDir, entry.Name())
			if err := os.RemoveAll(sessionPath); err != nil {
				s.logWarn("failed to cleanup old session %s: %v", entry.Name(), err)
			} else {
				s.logDebug("cleaned up old session directory: %s", entry.Name())
			}
		}
	}
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

func (s *Server) logWarn(format string, args ...any) {
	if s.logLevel >= LogLevelWarn {
		s.logger.Printf("[WARN] "+format, args...)
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

	// Reject oversized messages to prevent memory exhaustion
	if contentLength > maxContentLength {
		return nil, fmt.Errorf("content length %d exceeds maximum %d bytes", contentLength, maxContentLength)
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
	case "textDocument/diagnostic":
		return s.handleDiagnostic(msg)
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

	// Validate workspace root is a real directory
	if s.workspaceRoot != "" {
		if info, err := os.Stat(s.workspaceRoot); err != nil {
			s.logError("workspace root does not exist: %s", s.workspaceRoot)
			// Continue anyway - the path might be created later
		} else if !info.IsDir() {
			s.logError("workspace root is not a directory: %s", s.workspaceRoot)
		}
	}

	// Store initialization options from the client
	s.initOptions = params.InitializationOptions

	// Load configuration, preferring client-provided config path
	configPath := filepath.Join(s.workspaceRoot, ".terratidy.yaml")
	if s.initOptions != nil && s.initOptions.ConfigPath != "" {
		configPath = s.initOptions.ConfigPath
	}
	s.logDebug("Loading config from: %s", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		s.logDebug("Config load error (using defaults): %v", err)
		cfg = config.DefaultConfig()
	} else {
		s.logDebug("Config loaded: severity_threshold=%s", cfg.SeverityThreshold)
	}

	// Apply profile from client options
	if s.initOptions != nil && s.initOptions.Profile != "" {
		if profileErr := cfg.ApplyProfile(s.initOptions.Profile); profileErr != nil {
			// Profile not found is not fatal; log and continue
			s.logInfo("profile %q not found: %v", s.initOptions.Profile, profileErr)
		}
	}

	// Apply severity threshold from client options (overrides config file)
	if s.initOptions != nil && s.initOptions.SeverityThreshold != "" {
		s.logDebug("Overriding config severity_threshold with client setting: %s", s.initOptions.SeverityThreshold)
		cfg.SeverityThreshold = s.initOptions.SeverityThreshold
	}

	s.config = cfg

	// Initialize session temp directory for document analysis
	if err := s.initSessionTempDir(); err != nil {
		s.logWarn("failed to initialize session temp dir: %v", err)
		// Not fatal - will fall back to system temp
	}

	// Initialize engines with config
	s.lintEngine = lint.New(nil)
	s.styleEngine = style.New(s.buildStyleConfig())

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full sync
				Save:      &SaveOptions{IncludeText: true},
			},
			DocumentFormattingProvider: true,
			CodeActionProvider:         true,
			// Note: We use push diagnostics (publishDiagnostics) instead of pull
			// diagnostics to avoid duplication. Don't advertise DiagnosticProvider.
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
	// LSP spec requires result: null (not omitted)
	return s.writeMessage(ResponseMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  json.RawMessage("null"),
	})
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

	// Check document size limit
	if len(params.TextDocument.Text) > maxDocumentSize {
		s.logWarn("document %s exceeds size limit (%d > %d bytes), ignoring",
			params.TextDocument.URI, len(params.TextDocument.Text), maxDocumentSize)
		return nil
	}

	s.docMu.Lock()
	// Check document count limit (only for new documents)
	if _, exists := s.documents[params.TextDocument.URI]; !exists && len(s.documents) >= maxDocuments {
		s.docMu.Unlock()
		s.logWarn("document limit reached (%d), cannot open %s", maxDocuments, params.TextDocument.URI)
		return nil
	}
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

	// Check document size limit for each change
	for _, change := range params.ContentChanges {
		if len(change.Text) > maxDocumentSize {
			s.logWarn("document %s change exceeds size limit (%d > %d bytes), ignoring",
				params.TextDocument.URI, len(change.Text), maxDocumentSize)
			return nil
		}
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
	diagnostics := s.getDiagnostics(uri)
	return s.writeMessage(NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	})
}

// handleDiagnostic handles textDocument/diagnostic request (pull diagnostics)
func (s *Server) handleDiagnostic(msg RequestMessage) error {
	var params DocumentDiagnosticParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	diagnostics := s.getDiagnostics(params.TextDocument.URI)
	return s.sendResult(msg.ID, DocumentDiagnosticReport{
		Kind:  "full",
		Items: diagnostics,
	})
}

// getDiagnostics generates diagnostics for a document
func (s *Server) getDiagnostics(uri string) []Diagnostic {
	s.docMu.RLock()
	doc, ok := s.documents[uri]
	s.docMu.RUnlock()

	if !ok {
		return []Diagnostic{}
	}

	filePath := uriToPath(uri)
	if filePath == "" {
		return []Diagnostic{} // Invalid URI
	}

	// Validate path is within workspace to prevent path traversal
	validPath, err := s.validateWorkspacePath(filePath)
	if err != nil {
		s.logDebug("path validation failed for %s: %v", uri, err)
		return []Diagnostic{}
	}
	filePath = validPath

	// Only process .tf and .hcl files
	ext := filepath.Ext(filePath)
	if ext != ".tf" && ext != ".hcl" && ext != ".tfvars" {
		return []Diagnostic{}
	}

	// Acquire semaphore to limit concurrent diagnostics
	s.diagSem <- struct{}{}
	defer func() { <-s.diagSem }()

	// Reuse a temp file per document to avoid creating one per keystroke
	if doc.tempFile == "" {
		tmpExt := filepath.Ext(filePath)
		if tmpExt == "" {
			tmpExt = ".tf"
		}
		// Use session temp directory if available, otherwise fall back to system temp
		tempDir := s.sessionTempDir
		if tempDir == "" {
			tempDir = os.TempDir()
		}
		f, err := os.CreateTemp(tempDir, "terratidy-*"+tmpExt)
		if err != nil {
			return []Diagnostic{}
		}
		doc.tempFile = f.Name()
		_ = f.Close()
	}

	if err := os.WriteFile(doc.tempFile, []byte(doc.Content), 0o600); err != nil {
		return []Diagnostic{}
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

	// Filter findings by severity threshold
	threshold := s.getSeverityThreshold()
	filteredFindings := make([]sdk.Finding, 0, len(findings))
	for _, f := range findings {
		if meetsThreshold(f.Severity, threshold) {
			filteredFindings = append(filteredFindings, f)
		}
	}

	// Convert findings to diagnostics
	diagnostics := make([]Diagnostic, 0, len(filteredFindings))
	for _, f := range filteredFindings {
		diag := Diagnostic{
			Range: Range{
				Start: Position{
					Line:      max(0, f.Location.StartLine-1),
					Character: max(0, f.Location.StartColumn-1),
				},
				End: Position{
					Line:      max(0, f.Location.EndLine-1),
					Character: max(0, f.Location.EndColumn-1),
				},
			},
			Severity: severityToLSP(f.Severity),
			Code:     f.Rule,
			Source:   "terratidy",
			Message:  f.Message,
		}
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics
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
// Handles URL encoding, Windows paths, and UNC paths correctly.
// Returns empty string for invalid URIs (fail-secure).
func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "" // Invalid URI, fail secure
	}

	// Decode all percent-encoding in the path.
	// url.Parse().Path decodes most sequences but NOT %2F (slash) or %5C (backslash).
	// We need full decoding before any path validation to prevent traversal via encoded sequences.
	p, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "" // Invalid encoding, fail secure
	}

	// Handle UNC paths: file://server/share/path -> //server/share/path
	// The Host contains the server name for UNC paths.
	if parsed.Host != "" {
		p = "//" + parsed.Host + p
	}

	// Handle Windows drive letters.
	// file:///C:/path gives Path="/C:/path", we want "C:/path".
	// Check for /X: pattern where X is a drive letter (a-z, A-Z).
	if len(p) >= 3 && p[0] == '/' && isASCIILetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}

	return p
}

// isASCIILetter returns true if b is an ASCII letter (a-z, A-Z).
func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// validateWorkspacePath validates that a path is within the workspace root.
// It resolves symlinks (if the file exists) and checks the path doesn't escape.
// Returns the validated path or an error if the path is outside the workspace.
func (s *Server) validateWorkspacePath(path string) (string, error) {
	if s.workspaceRoot == "" {
		return path, nil // No workspace root set, skip validation
	}

	// Clean both paths for consistent comparison
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(s.workspaceRoot)

	// Resolve symlinks in workspace root for accurate comparison
	resolvedRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		// If workspace root can't be resolved, use cleaned version
		resolvedRoot = cleanRoot
	}

	// Resolve symlinks in the path to catch symlink-based escapes
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// File doesn't exist yet or can't be resolved.
		// Validate the logical path instead.
		resolvedPath = cleanPath
	}

	// Check if the resolved path is within the workspace using filepath.Rel.
	// Rel returns a relative path from resolvedRoot to resolvedPath.
	// If the path is outside, the relative path will start with "..".
	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("path outside workspace: %s", path)
	}

	// Reject paths that escape the workspace (relative path starts with "..")
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}

	return resolvedPath, nil
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

// getSeverityThreshold returns the configured severity threshold
func (s *Server) getSeverityThreshold() sdk.Severity {
	if s.config != nil && s.config.SeverityThreshold != "" {
		return sdk.ParseSeverity(s.config.SeverityThreshold, sdk.SeverityInfo)
	}
	return sdk.SeverityInfo // Default: show all
}

// meetsThreshold returns true if the finding severity meets or exceeds the threshold
func meetsThreshold(severity, threshold sdk.Severity) bool {
	// Severity order: error > warning > info
	severityRank := map[sdk.Severity]int{
		sdk.SeverityError:   3,
		sdk.SeverityWarning: 2,
		sdk.SeverityInfo:    1,
	}
	return severityRank[severity] >= severityRank[threshold]
}

// buildStyleConfig creates a style.Config from the server's config
func (s *Server) buildStyleConfig() *style.Config {
	styleCfg := &style.Config{
		Rules: make(map[string]style.RuleConfig),
	}

	if s.config == nil {
		return styleCfg
	}

	// Merge override rules from config
	for ruleName, ruleCfg := range s.config.Overrides.Rules {
		rc := style.RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
		}
		if ruleCfg.Config != nil {
			rc.Options = ruleCfg.Config
		}
		styleCfg.Rules[ruleName] = rc
	}

	return styleCfg
}
