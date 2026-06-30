package lsp

import (
	"encoding/json"
	"sort"
	"unicode/utf16"
	"unicode/utf8"
)

// RequestMessage represents an LSP request message
type RequestMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ResponseMessage represents an LSP response message
type ResponseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// NotificationMessage represents an LSP notification message
type NotificationMessage struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// ResponseError represents an LSP error
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// InitializeParams represents the parameters for the initialize request
type InitializeParams struct {
	ProcessID             int                    `json:"processId,omitempty"`
	RootPath              string                 `json:"rootPath,omitempty"`
	RootURI               string                 `json:"rootUri,omitempty"`
	Capabilities          ClientCapabilities     `json:"capabilities"`
	InitializationOptions *InitializationOptions `json:"initializationOptions,omitempty"`
}

// InitializationOptions represents client-provided options from the editor.
// The server only ever unmarshals this type (it is wire-format input from the
// LSP client), so json marshal-time tag options like omitzero/omitempty are
// intentionally omitted on the engines field — they would have no effect.
type InitializationOptions struct {
	Profile           string        `json:"profile,omitempty"`
	ConfigPath        string        `json:"configPath,omitempty"`
	Engines           EngineToggles `json:"engines"`
	SeverityThreshold string        `json:"severityThreshold,omitempty"`
}

// EngineToggles represents which engines are enabled
type EngineToggles struct {
	Fmt    bool `json:"fmt"`
	Style  bool `json:"style"`
	Lint   bool `json:"lint"`
	Policy bool `json:"policy"`
}

// ClientCapabilities represents client capabilities. The server only
// unmarshals this type; marshal-time tag options are intentionally omitted.
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace"`
}

// TextDocumentClientCapabilities represents text document capabilities
type TextDocumentClientCapabilities struct {
	Synchronization    *TextDocumentSyncClientCapabilities   `json:"synchronization,omitempty"`
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
}

// TextDocumentSyncClientCapabilities represents synchronization capabilities
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// PublishDiagnosticsClientCapabilities represents diagnostic capabilities
type PublishDiagnosticsClientCapabilities struct {
	RelatedInformation     bool                  `json:"relatedInformation,omitempty"`
	TagSupport             *DiagnosticTagSupport `json:"tagSupport,omitempty"`
	VersionSupport         bool                  `json:"versionSupport,omitempty"`
	CodeDescriptionSupport bool                  `json:"codeDescriptionSupport,omitempty"`
	DataSupport            bool                  `json:"dataSupport,omitempty"`
}

// DiagnosticTagSupport represents supported diagnostic tags
type DiagnosticTagSupport struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities
type WorkspaceClientCapabilities struct {
	ApplyEdit        bool `json:"applyEdit,omitempty"`
	WorkspaceFolders bool `json:"workspaceFolders,omitempty"`
	Configuration    bool `json:"configuration,omitempty"`
}

// InitializeResult represents the result of the initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	TextDocumentSync           *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	DocumentFormattingProvider bool                     `json:"documentFormattingProvider,omitempty"`
	CodeActionProvider         bool                     `json:"codeActionProvider,omitempty"`
	DiagnosticProvider         *DiagnosticOptions       `json:"diagnosticProvider,omitempty"`
}

// TextDocumentSyncOptions represents text document sync options
type TextDocumentSyncOptions struct {
	OpenClose bool         `json:"openClose,omitempty"`
	Change    int          `json:"change,omitempty"` // 0: None, 1: Full, 2: Incremental
	Save      *SaveOptions `json:"save,omitempty"`
}

// SaveOptions represents save options
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// DiagnosticOptions represents diagnostic provider options
type DiagnosticOptions struct {
	Identifier            string `json:"identifier,omitempty"`
	InterFileDependencies bool   `json:"interFileDependencies"`
	WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentItem represents a text document item
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidOpenTextDocumentParams represents didOpen parameters
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams represents didChange parameters
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent represents a content change event
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams represents didClose parameters
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams represents didSave parameters
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// DocumentFormattingParams represents formatting parameters
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions represents formatting options
type FormattingOptions struct {
	TabSize                int  `json:"tabSize"`
	InsertSpaces           bool `json:"insertSpaces"`
	TrimTrailingWhitespace bool `json:"trimTrailingWhitespace,omitempty"`
	InsertFinalNewline     bool `json:"insertFinalNewline,omitempty"`
	TrimFinalNewlines      bool `json:"trimFinalNewlines,omitempty"`
}

// TextEdit represents a text edit
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a diagnostic
type Diagnostic struct {
	Range              Range               `json:"range"`
	Severity           int                 `json:"severity,omitempty"`
	Code               string              `json:"code,omitempty"`
	CodeDescription    *CodeDescription    `json:"codeDescription,omitempty"`
	Source             string              `json:"source,omitempty"`
	Message            string              `json:"message"`
	Tags               []int               `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelated `json:"relatedInformation,omitempty"`
	Data               any                 `json:"data,omitempty"`
}

// CodeDescription represents a code description
type CodeDescription struct {
	Href string `json:"href"`
}

// DiagnosticRelated represents related diagnostic information
type DiagnosticRelated struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// PublishDiagnosticsParams represents publish diagnostics parameters
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// DocumentDiagnosticParams represents textDocument/diagnostic request params
type DocumentDiagnosticParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentDiagnosticReport represents the response to textDocument/diagnostic
type DocumentDiagnosticReport struct {
	Kind  string       `json:"kind"` // "full" or "unchanged"
	Items []Diagnostic `json:"items"`
}

// CodeActionParams represents code action parameters
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext represents the context for code actions
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

// CodeAction represents a code action
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
	Data        any            `json:"data,omitempty"`
}

// WorkspaceEdit represents a workspace edit
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// Command represents a command
type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

// DidChangeConfigurationParams represents workspace/didChangeConfiguration parameters.
// The settings field contains client configuration pushed to the server.
type DidChangeConfigurationParams struct {
	// Settings contains the actual configuration values.
	// For TerraTidy, this has the same shape as InitializationOptions.
	Settings *InitializationOptions `json:"settings,omitempty"`
}

// byteRangeToLSPRange converts the half-open byte range [start, end) in content
// to an LSP Range. The LSP spec defines Position.character as a UTF-16
// code-unit offset from the line start, so multi-byte runes (e.g. an em-dash
// in a comment, CJK in a string literal, or a supplementary-plane emoji that
// occupies a surrogate pair) are counted correctly rather than as raw bytes.
// CRLF counts as two UTF-16 code units; negative offsets clamp at zero;
// offsets past len(content) clamp at end-of-file.
//
// Builds a newline-offset table per call (O(N)) and resolves each offset by
// binary search (O(log L)). Per-line UTF-16 conversion is O(byteOffsetInLine);
// safe to invoke in loops over many TextEdits because each line is decoded
// once per Position computation, not once per code unit.
func byteRangeToLSPRange(content []byte, start, end int) Range {
	lineStarts := lineStartOffsets(content)
	return Range{
		Start: byteOffsetToPosition(content, start, lineStarts),
		End:   byteOffsetToPosition(content, end, lineStarts),
	}
}

func lineStartOffsets(content []byte) []int {
	starts := []int{0}
	for i, b := range content {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// byteOffsetToPosition converts a byte offset into an LSP Position whose
// character field is a UTF-16 code-unit count, per the LSP spec. Out-of-range
// offsets clamp to file boundaries. Malformed UTF-8 inside the line is treated
// as one code unit per byte so the returned character count stays monotonically
// aligned with the byte offset rather than drifting silently.
func byteOffsetToPosition(content []byte, offset int, lineStarts []int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(content) {
		offset = len(content)
	}
	// SearchInts returns the first index whose lineStart is > offset; the
	// containing line is therefore one index earlier.
	line := sort.SearchInts(lineStarts, offset+1) - 1
	if line < 0 {
		line = 0
	}
	lineStart := lineStarts[line]

	character := 0
	for i := lineStart; i < offset; {
		r, size := utf8.DecodeRune(content[i:])
		if r == utf8.RuneError && size == 1 {
			// Malformed byte: count as 1 code unit to keep alignment.
			character++
			i++
			continue
		}
		character += utf16.RuneLen(r)
		i += size
	}
	return Position{Line: line, Character: character}
}
