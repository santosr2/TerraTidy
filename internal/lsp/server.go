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

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/santosr2/TerraTidy/internal/buildinfo"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/internal/plugins"
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

// DefaultDebounceDelay is the default delay before running diagnostics after a document change.
// This prevents running expensive diagnostics on every keystroke.
const DefaultDebounceDelay = 500 * time.Millisecond

// ConfigReloadDebounceDelay is the delay before reloading config after a file change.
// This coalesces rapid file events (e.g., editor save with multiple writes).
const ConfigReloadDebounceDelay = 100 * time.Millisecond

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
	pluginRules    []sdk.Rule // plugin rules loaded from configured directories
	workspaceRoot  string
	initialized    bool
	shutdown       bool
	logger         *log.Logger
	logLevel       LogLevel
	logFile        *os.File
	diagSem        chan struct{} // semaphore for concurrent diagnostics
	sessionTempDir string        // private temp directory for this session

	// Debouncing: defer diagnostics until typing pauses
	debounceTimers map[string]*time.Timer // URI -> pending timer
	debounceMu     sync.Mutex             // protects debounceTimers
	debounceDelay  time.Duration          // configurable delay (default: DefaultDebounceDelay)

	// Config watching: auto-reload on file changes
	configWatcher     *fsnotify.Watcher // watches config files for changes
	configFiles       []string          // list of config files being watched
	configWatcherMu   sync.Mutex        // protects configWatcher, configFiles, configReloadTimer, closing
	configPath        string            // path to main config file (for reload)
	configReloadTimer *time.Timer       // debounce timer for config reload
	configReloadDelay time.Duration     // configurable delay (default: ConfigReloadDebounceDelay)
	closing           bool              // true when server is shutting down
	republishWg       sync.WaitGroup    // tracks in-flight republishAllDiagnostics goroutines

	// Engine mutex: protects engine and config fields during reload
	engineMu sync.RWMutex // protects config, lintEngine, styleEngine, pluginRules
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
		reader:            bufio.NewReader(in),
		writer:            out,
		documents:         make(map[string]*Document),
		logger:            log.New(os.Stderr, "terratidy-lsp: ", log.Ltime),
		logLevel:          LogLevelInfo,
		diagSem:           make(chan struct{}, maxConcurrentDiagnostics),
		debounceTimers:    make(map[string]*time.Timer),
		debounceDelay:     DefaultDebounceDelay,
		configReloadDelay: ConfigReloadDebounceDelay,
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
	// Mark server as closing to prevent new operations
	s.configWatcherMu.Lock()
	s.closing = true
	s.configWatcherMu.Unlock()

	// Stop config watcher (cancels pending reload timer)
	s.stopConfigWatcher()

	// Wait for any in-flight republish operations to complete
	s.republishWg.Wait()

	// Clean up session temp directory
	if s.sessionTempDir != "" {
		_ = os.RemoveAll(s.sessionTempDir)
	}
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}

// initConfigWatcher starts watching config files for changes.
// When a config file changes, the server reloads configuration and reinitializes engines.
func (s *Server) initConfigWatcher(configPath string, configFiles []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating config watcher: %w", err)
	}

	s.configWatcherMu.Lock()
	s.configWatcher = watcher
	s.configFiles = configFiles
	s.configPath = configPath
	s.configWatcherMu.Unlock()

	// Add all config files to watcher
	for _, file := range configFiles {
		if err := watcher.Add(file); err != nil {
			s.logWarn("failed to watch config file %s: %v", file, err)
			// Continue watching other files
		} else {
			s.logDebug("watching config file: %s", file)
		}
	}

	// Start event handler goroutine
	go s.handleConfigWatchEvents()

	return nil
}

// stopConfigWatcher stops watching config files and releases resources.
func (s *Server) stopConfigWatcher() {
	s.configWatcherMu.Lock()
	defer s.configWatcherMu.Unlock()

	// Cancel any pending reload
	if s.configReloadTimer != nil {
		s.configReloadTimer.Stop()
		s.configReloadTimer = nil
	}

	if s.configWatcher != nil {
		if err := s.configWatcher.Close(); err != nil {
			s.logWarn("failed to close config watcher: %v", err)
		}
		s.configWatcher = nil
		s.configFiles = nil
	}
}

// handleConfigWatchEvents processes fsnotify events for config file changes.
func (s *Server) handleConfigWatchEvents() {
	s.configWatcherMu.Lock()
	watcher := s.configWatcher
	s.configWatcherMu.Unlock()

	if watcher == nil {
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return // Watcher closed
			}

			// Only handle Write and Create events (file modified or recreated)
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				s.logDebug("config file changed: %s (debouncing)", event.Name)
				s.scheduleConfigReload()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return // Watcher closed
			}
			s.logError("config watcher error: %v", err)
		}
	}
}

// scheduleConfigReload schedules a config reload after a debounce delay.
// This coalesces rapid file events from editors that write multiple times per save.
func (s *Server) scheduleConfigReload() {
	s.configWatcherMu.Lock()
	defer s.configWatcherMu.Unlock()

	// Don't schedule if server is closing
	if s.closing {
		return
	}

	// Cancel any pending reload
	if s.configReloadTimer != nil {
		s.configReloadTimer.Stop()
	}

	// Schedule new reload after debounce delay
	s.configReloadTimer = time.AfterFunc(s.configReloadDelay, func() {
		// Check closing flag again before executing reload
		s.configWatcherMu.Lock()
		isClosing := s.closing
		s.configWatcherMu.Unlock()
		if isClosing {
			return
		}

		s.logInfo("config file changed, reloading...")
		if err := s.reloadConfig(); err != nil {
			s.logError("failed to reload config: %v", err)
		} else {
			s.logInfo("config reloaded successfully")
			// Track the republish goroutine so Close() can wait for it
			s.republishWg.Go(s.republishAllDiagnostics)
		}
	})
}

// reloadConfig reloads the configuration from disk and reinitializes engines.
func (s *Server) reloadConfig() error {
	s.configWatcherMu.Lock()
	configPath := s.configPath
	s.configWatcherMu.Unlock()

	if configPath == "" {
		return fmt.Errorf("no config path set")
	}

	// Snapshot initOptions under engineMu to avoid race with handleInitialize
	// (initOptions is written once during initialize, but we need safe access)
	s.engineMu.RLock()
	initOpts := s.initOptions
	s.engineMu.RUnlock()

	// Load config with file list (outside lock - I/O bound)
	cfg, configFiles, err := config.LoadWithFiles(configPath)
	if err != nil {
		// Keep using existing config on error
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply profile from init options
	if initOpts != nil && initOpts.Profile != "" {
		if profileErr := cfg.ApplyProfile(initOpts.Profile); profileErr != nil {
			s.logInfo("profile %q not found: %v", initOpts.Profile, profileErr)
		}
	}

	// Apply severity threshold from init options
	if initOpts != nil && initOpts.SeverityThreshold != "" {
		cfg.SeverityThreshold = initOpts.SeverityThreshold
	}

	// Reload plugin rules if plugins are enabled (outside lock - I/O bound)
	var pluginRules []sdk.Rule
	if cfg.Plugins.Enabled {
		mgr := plugins.NewManager(cfg.Plugins.Directories, cfg.Plugins.ShouldVerifyIntegrity())
		if err := mgr.LoadAll(); err != nil {
			s.logWarn("failed to load plugins: %v", err)
		} else {
			rulesMap := mgr.GetRules()
			pluginRules = make([]sdk.Rule, 0, len(rulesMap))
			for _, rule := range rulesMap {
				pluginRules = append(pluginRules, rule)
			}
		}
	}

	// Build style config before acquiring lock
	styleConfig := buildStyleConfigFromCfg(cfg)

	// Create new engines before acquiring lock
	newLintEngine := lint.New(nil, pluginRules...)
	newStyleEngine := style.New(styleConfig, pluginRules...)

	// Update all engine-related fields atomically
	s.engineMu.Lock()
	s.config = cfg
	s.pluginRules = pluginRules
	s.lintEngine = newLintEngine
	s.styleEngine = newStyleEngine
	s.engineMu.Unlock()

	// Update watched files (new imports may have been added)
	s.updateConfigWatcher(configFiles)

	return nil
}

// updateConfigWatcher updates the watched config files after a reload.
func (s *Server) updateConfigWatcher(newFiles []string) {
	s.configWatcherMu.Lock()
	defer s.configWatcherMu.Unlock()

	if s.configWatcher == nil {
		return
	}

	// Build set of current files
	currentSet := make(map[string]bool)
	for _, f := range s.configFiles {
		currentSet[f] = true
	}

	// Build set of new files
	newSet := make(map[string]bool)
	for _, f := range newFiles {
		newSet[f] = true
	}

	// Remove watches for files no longer in config
	for f := range currentSet {
		if !newSet[f] {
			if err := s.configWatcher.Remove(f); err != nil {
				s.logWarn("failed to unwatch config file %s: %v", f, err)
			} else {
				s.logDebug("unwatched config file: %s", f)
			}
		}
	}

	// Add watches for new files
	for f := range newSet {
		if !currentSet[f] {
			if err := s.configWatcher.Add(f); err != nil {
				s.logWarn("failed to watch config file %s: %v", f, err)
			} else {
				s.logDebug("watching config file: %s", f)
			}
		}
	}

	s.configFiles = newFiles
}

// republishAllDiagnostics republishes diagnostics for all open documents.
// Called after config reload to apply new settings.
func (s *Server) republishAllDiagnostics() {
	s.docMu.RLock()
	uris := make([]string, 0, len(s.documents))
	for uri := range s.documents {
		uris = append(uris, uri)
	}
	s.docMu.RUnlock()

	for _, uri := range uris {
		if err := s.publishDiagnostics(uri); err != nil {
			s.logError("failed to republish diagnostics for %s: %v", uri, err)
		}
	}
}

// initSessionTempDir creates a private temp directory for this server session.
// It also cleans up stale session directories older than sessionTempMaxAge.
func (s *Server) initSessionTempDir() {
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
		return
	}

	s.sessionTempDir = sessionDir
	s.logDebug("using session temp directory: %s", sessionDir)
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
			// Check for exit request - return to allow cleanup
			if errors.Is(err, ErrServerExit) {
				return err
			}
			// Log other errors but continue processing
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
		if after, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			lengthStr := strings.TrimSpace(after)
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid content length: %w", err)
			}
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("invalid or missing content length header")
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
	case "workspace/didChangeConfiguration":
		return s.handleDidChangeConfiguration(msg)
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
	cfg, configFiles, err := config.LoadWithFiles(configPath)
	if err != nil {
		s.logDebug("Config load error (using defaults): %v", err)
		cfg = config.DefaultConfig()
		configFiles = nil
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

	// Initialize session temp directory for document analysis.
	// Falls back to the system temp dir internally if creation fails.
	s.initSessionTempDir()

	// Load plugin rules if plugins are enabled
	if cfg.Plugins.Enabled {
		mgr := plugins.NewManager(cfg.Plugins.Directories, cfg.Plugins.ShouldVerifyIntegrity())
		if err := mgr.LoadAll(); err != nil {
			s.logWarn("failed to load plugins: %v", err)
			// Not fatal - continue without plugin rules
		} else {
			rulesMap := mgr.GetRules()
			s.pluginRules = make([]sdk.Rule, 0, len(rulesMap))
			for _, rule := range rulesMap {
				s.pluginRules = append(s.pluginRules, rule)
			}
		}
	}

	// Initialize engines with config and plugin rules
	s.lintEngine = lint.New(nil, s.pluginRules...)
	s.styleEngine = style.New(s.buildStyleConfig(), s.pluginRules...)

	// Start config file watcher if config files exist
	if len(configFiles) > 0 {
		if err := s.initConfigWatcher(configPath, configFiles); err != nil {
			s.logWarn("failed to start config watcher: %v", err)
			// Not fatal - continue without auto-reload
		}
	}

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

// handleDidChangeConfiguration handles workspace/didChangeConfiguration notification.
// This is called when the client pushes configuration changes to the server.
func (s *Server) handleDidChangeConfiguration(msg RequestMessage) error {
	var params DidChangeConfigurationParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didChangeConfiguration params: %w", err)
	}

	// If no settings provided, nothing to do
	if params.Settings == nil {
		s.logDebug("configuration change received with no settings, skipping")
		return nil
	}

	// Replace initOptions with new settings.
	// LSP clients (including VSCode) send complete settings on every change,
	// so we don't need partial merging. This avoids inconsistent behavior
	// between string fields (which could be conditionally merged) and boolean
	// fields (which can't distinguish zero from explicit false).
	s.engineMu.Lock()
	s.initOptions = params.Settings
	s.engineMu.Unlock()

	// Trigger a config reload to apply the new settings
	if err := s.reloadConfig(); err != nil {
		s.logError("failed to reload config after didChangeConfiguration: %v", err)
		return nil // Don't return error for notifications
	}

	s.logInfo("configuration updated successfully")
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

// ErrServerExit is returned when the LSP server should exit.
// The exit code is determined by the Shutdown field.
var ErrServerExit = errors.New("server exit requested")

// handleExit handles the exit notification
func (s *Server) handleExit() error {
	// Return sentinel error instead of os.Exit to allow proper cleanup
	return ErrServerExit
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

	// Debounce diagnostics: cancel any pending timer and schedule new one
	s.scheduleDebouncedDiagnostics(params.TextDocument.URI)
	return nil
}

// scheduleDebouncedDiagnostics schedules diagnostics to run after the debounce delay.
// If called again before the delay expires, the previous timer is canceled.
func (s *Server) scheduleDebouncedDiagnostics(uri string) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()

	// Cancel any existing timer for this URI
	if timer, ok := s.debounceTimers[uri]; ok {
		timer.Stop()
	}

	// Schedule new timer
	s.debounceTimers[uri] = time.AfterFunc(s.debounceDelay, func() {
		// Clean up timer from map
		s.debounceMu.Lock()
		delete(s.debounceTimers, uri)
		s.debounceMu.Unlock()

		// Run diagnostics (ignore error, already logged internally)
		_ = s.publishDiagnostics(uri)
	})
}

// cancelDebouncedDiagnostics cancels any pending debounced diagnostics for a URI.
// Called when a document is closed to prevent diagnostics for non-existent documents.
func (s *Server) cancelDebouncedDiagnostics(uri string) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()

	if timer, ok := s.debounceTimers[uri]; ok {
		timer.Stop()
		delete(s.debounceTimers, uri)
	}
}

// handleDidClose handles textDocument/didClose notification
func (s *Server) handleDidClose(msg RequestMessage) error {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return fmt.Errorf("parsing didClose params: %w", err)
	}

	// Cancel any pending debounced diagnostics for this document
	s.cancelDebouncedDiagnostics(params.TextDocument.URI)

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
	var original string
	if ok {
		original = doc.Content
	}
	s.docMu.RUnlock()

	if !ok {
		return s.sendResult(msg.ID, nil)
	}

	// Format the document content using hclwrite
	formatted := string(format.Format([]byte(original)))

	// If content is unchanged, return empty edits
	if formatted == original {
		return s.sendResult(msg.ID, []TextEdit{})
	}

	edits := []TextEdit{{
		Range:   wholeFileRange(original),
		NewText: formatted,
	}}
	return s.sendResult(msg.ID, edits)
}

// wholeFileRange returns the LSP Range covering the entire document. Delegates
// to byteRangeToLSPRange so the final-line character count is in UTF-16 code
// units, matching the LSP spec for any document whose last line contains
// multi-byte runes.
func wholeFileRange(original string) Range {
	return byteRangeToLSPRange([]byte(original), 0, len(original))
}

// handleCodeAction handles textDocument/codeAction request.
//
// For each incoming diagnostic the handler tries to find the originating
// style rule via Engine.Fixer and ask it for byte-range edits against the
// unsaved buffer. Diagnostics whose code doesn't map to a
// Fixer-implementing rule (or whose Fix returns nothing) fall back to the
// whole-file format quickfix the LSP has always offered.
//
// Parse failures and Fix errors are silent — the diagnostic simply yields no
// code action, matching the spec acceptance criterion for malformed HCL.
func (s *Server) handleCodeAction(msg RequestMessage) error {
	var params CodeActionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params")
	}

	uri := params.TextDocument.URI

	// Snapshot the document content under docMu. The unlock before computing
	// edits is safe: a concurrent didChange may update doc.Content, but the
	// parsed AST and the byte-offset translation below both reference this
	// `original` snapshot, so the edits remain self-consistent. The editor
	// reconciles against its current buffer via the versioned document
	// sync the LSP spec already mandates.
	s.docMu.RLock()
	doc, ok := s.documents[uri]
	var original string
	if ok {
		original = doc.Content
	}
	s.docMu.RUnlock()

	if !ok || len(params.Context.Diagnostics) == 0 {
		return s.sendResult(msg.ID, []CodeAction{})
	}

	content := []byte(original)

	// Snapshot the style engine pointer once under the engine lock so
	// concurrent config reloads can't swap the rule set mid-iteration.
	s.engineMu.RLock()
	styleEng := s.styleEngine
	s.engineMu.RUnlock()

	// Persist the unsaved buffer to a request-private temp file so rule
	// Fix() bodies that read ctx.File from disk see the in-memory content,
	// and so that concurrent getDiagnostics on the same URI (which writes
	// to the document's shared tempFile) cannot race the bytes out from
	// underneath us. Parse the same bytes so AST and disk match. If any
	// step fails, parsedFile is nil and every diagnostic falls through to
	// the format fallback.
	parsedFile, ctxFile, cleanup := s.prepareFixerInputs(uri, content)
	defer cleanup()

	formatEdits := buildFormatFallbackEdits(original)

	// Per-diagnostic Fixer actions are emitted as we go. Diagnostics with no
	// matching Fixer bucket into a single shared format-fallback action so
	// the response carries one "Format document" entry rather than N copies
	// of the same whole-file edit (one per unresolved diagnostic).
	actions := make([]CodeAction, 0, len(params.Context.Diagnostics))
	var fallbackDiags []Diagnostic
	for i := range params.Context.Diagnostics {
		diag := params.Context.Diagnostics[i]
		if diag.Code == "" {
			continue
		}

		edits := s.buildFixerEdits(styleEng, parsedFile, ctxFile, content, diag.Code)
		if edits != nil {
			actions = append(actions, codeActionFor(uri, diag, edits))
			continue
		}
		if formatEdits != nil {
			fallbackDiags = append(fallbackDiags, diag)
		}
	}
	if len(fallbackDiags) > 0 {
		actions = append(actions, formatFallbackAction(uri, fallbackDiags, formatEdits))
	}

	return s.sendResult(msg.ID, actions)
}

// formatFallbackAction wraps the shared whole-file format edit as a single
// CodeAction carrying every diagnostic that had no per-rule Fixer. Aggregating
// here keeps the response from emitting one "Format document" action per
// unresolved diagnostic, which would clutter the editor's quick-fix menu with
// N copies of the same fix.
func formatFallbackAction(uri string, diags []Diagnostic, edits []TextEdit) CodeAction {
	return CodeAction{
		Title:       "Format document",
		Kind:        "quickfix",
		Diagnostics: diags,
		IsPreferred: true,
		Edit: &WorkspaceEdit{
			Changes: map[string][]TextEdit{uri: edits},
		},
	}
}

// prepareFixerInputs writes the unsaved buffer to a request-private temp file
// and parses it so per-finding Fixers receive an AST and an on-disk ctx.File
// that match. The temp file is created via os.CreateTemp inside the session
// temp dir (or the system temp dir as fallback) so it cannot race with the
// document's long-lived diagnostics temp file. The returned cleanup is always
// safe to call — it is a no-op when no file was created.
//
// Returns (nil, "", noop) if the URI is invalid, path validation fails, the
// temp file can't be created or written, or the buffer is not parseable
// HCL/JSON. Callers must treat a nil parsed file as "no Fixer path available;
// use the format fallback for every diagnostic."
func (s *Server) prepareFixerInputs(uri string, content []byte) (*hcl.File, string, func()) {
	noop := func() {}

	filePath := uriToPath(uri)
	if filePath == "" {
		return nil, "", noop
	}
	if _, err := s.validateWorkspacePath(filePath); err != nil {
		s.logDebug("code action path validation failed for %s: %v", uri, err)
		return nil, "", noop
	}

	ext := filepath.Ext(filePath)
	if ext == "" {
		ext = ".tf"
	}
	tempDir := s.sessionTempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	f, err := os.CreateTemp(tempDir, "terratidy-ca-*"+ext)
	if err != nil {
		return nil, "", noop
	}
	tempFile := f.Name()
	_ = f.Close()
	cleanup := func() { _ = os.Remove(tempFile) }

	if err := os.WriteFile(tempFile, content, 0o600); err != nil {
		cleanup()
		return nil, "", noop
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, tempFile)
	if diags.HasErrors() {
		file, diags = parser.ParseJSON(content, tempFile)
	}
	if diags.HasErrors() || file == nil {
		cleanup()
		return nil, "", noop
	}
	return file, tempFile, cleanup
}

// buildFixerEdits looks up the rule named by ruleCode on styleEng, invokes
// its Fixer against the parsed buffer, and translates the resulting byte-range
// edits to LSP TextEdits. Returns nil if any precondition is missing (no
// engine, no parsed file, no Fixer for this rule), if Fix errors, or if Fix
// returns no edits — callers then fall through to the format fallback.
//
// ctx.WorkDir is the LSP server's workspaceRoot rather than ".". Unlike the
// CLI's checkFile path (which runs from the user's shell cwd and treats "."
// as authoritative), the LSP knows the project root from its initialize
// params, so passing it through gives rule fixers a meaningful absolute path
// to resolve relative references against. Empty workspaceRoot is acceptable
// — no in-tree Fixer currently reads ctx.WorkDir, so the field is best-effort.
func (s *Server) buildFixerEdits(styleEng *style.Engine, parsedFile *hcl.File, ctxFile string, content []byte, ruleCode string) []TextEdit {
	if styleEng == nil || parsedFile == nil {
		return nil
	}
	fixer := styleEng.Fixer(ruleCode)
	if fixer == nil {
		return nil
	}
	ctx := &sdk.Context{
		Context: context.Background(),
		Options: make(map[string]any),
		WorkDir: s.workspaceRoot,
		File:    ctxFile,
	}
	result, err := fixer.Fix(ctx, parsedFile)
	if err != nil || result == nil || len(result.Edits) == 0 {
		return nil
	}
	edits := make([]TextEdit, 0, len(result.Edits))
	for _, te := range result.Edits {
		edits = append(edits, TextEdit{
			Range:   byteRangeToLSPRange(content, te.Start, te.End),
			NewText: string(te.Replacement),
		})
	}
	return edits
}

// buildFormatFallbackEdits computes the legacy whole-file format quickfix.
// Returns nil when formatting leaves the content unchanged so the caller can
// distinguish "no fallback available" from "fallback is a no-op."
func buildFormatFallbackEdits(original string) []TextEdit {
	formatted := string(format.Format([]byte(original)))
	if formatted == original {
		return nil
	}
	return []TextEdit{{
		Range:   wholeFileRange(original),
		NewText: formatted,
	}}
}

// codeActionFor wraps a set of edits into a quickfix CodeAction tied to the
// diagnostic that produced them.
func codeActionFor(uri string, diag Diagnostic, edits []TextEdit) CodeAction {
	return CodeAction{
		Title:       fmt.Sprintf("Fix: %s", diag.Code),
		Kind:        "quickfix",
		Diagnostics: []Diagnostic{diag},
		IsPreferred: true,
		Edit: &WorkspaceEdit{
			Changes: map[string][]TextEdit{uri: edits},
		},
	}
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

// getOrCreateTempFile creates or returns the temp file path for a document.
// Returns empty string on failure.
func (s *Server) getOrCreateTempFile(uri, filePath string) string {
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
		return ""
	}
	tempFile := f.Name()
	_ = f.Close()

	// Update document with tempFile under write lock
	s.docMu.Lock()
	if d, exists := s.documents[uri]; exists {
		d.tempFile = tempFile
		s.documents[uri] = d
	}
	s.docMu.Unlock()

	return tempFile
}

// getDiagnostics generates diagnostics for a document
func (s *Server) getDiagnostics(uri string) []Diagnostic {
	// Copy document content and tempFile under lock to avoid data races
	s.docMu.RLock()
	doc, ok := s.documents[uri]
	if !ok {
		s.docMu.RUnlock()
		return []Diagnostic{}
	}
	content := doc.Content
	tempFile := doc.tempFile
	s.docMu.RUnlock()

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
	if tempFile == "" {
		tempFile = s.getOrCreateTempFile(uri, filePath)
		if tempFile == "" {
			return []Diagnostic{}
		}
	}

	if err := os.WriteFile(tempFile, []byte(content), 0o600); err != nil {
		return []Diagnostic{}
	}

	// Run lint and style checks based on engine toggles from InitializationOptions
	ctx := context.Background()
	var findings []sdk.Finding

	// Copy engine pointers under lock to avoid race with config reload
	s.engineMu.RLock()
	lintEng := s.lintEngine
	styleEng := s.styleEngine
	s.engineMu.RUnlock()

	if lintEng != nil && s.isEngineEnabled("lint") {
		lintFindings, err := lintEng.Run(ctx, []string{tempFile})
		if err == nil {
			findings = append(findings, lintFindings...)
		}
	}

	if styleEng != nil && s.isEngineEnabled("style") {
		styleFindings, err := styleEng.Run(ctx, []string{tempFile})
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
	s.engineMu.RLock()
	cfg := s.config
	s.engineMu.RUnlock()

	if cfg != nil && cfg.SeverityThreshold != "" {
		return sdk.ParseSeverity(cfg.SeverityThreshold, sdk.SeverityInfo)
	}
	return sdk.SeverityInfo // Default: show all
}

// isEngineEnabled checks if an engine should run based on InitializationOptions.
// If no InitializationOptions are set, engines default to enabled (except policy).
// This respects the VSCode terratidy.engines.* settings.
func (s *Server) isEngineEnabled(engine string) bool {
	if s.initOptions == nil {
		// No client options - use defaults (all enabled except policy)
		return engine != "policy"
	}

	switch engine {
	case "fmt":
		return s.initOptions.Engines.Fmt
	case "style":
		return s.initOptions.Engines.Style
	case "lint":
		return s.initOptions.Engines.Lint
	case "policy":
		return s.initOptions.Engines.Policy
	default:
		return true
	}
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
	s.engineMu.RLock()
	cfg := s.config
	s.engineMu.RUnlock()
	return buildStyleConfigFromCfg(cfg)
}

// buildStyleConfigFromCfg builds a style.Config from a config.Config.
// This is a separate function to allow building config before acquiring locks.
func buildStyleConfigFromCfg(cfg *config.Config) *style.Config {
	styleCfg := &style.Config{
		Rules: make(map[string]style.RuleConfig),
	}

	if cfg == nil {
		return styleCfg
	}

	// Apply style rules from engines.style.rules
	for ruleName, ruleCfg := range cfg.Engines.Style.Rules {
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
