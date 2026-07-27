package lsp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializationOptions_JSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected InitializationOptions
	}{
		{
			name: "full options",
			json: `{
				"profile": "production",
				"configPath": "/path/to/config.yaml",
				"engines": {
					"fmt": true,
					"style": false,
					"lint": true,
					"policy": false
				},
				"severityThreshold": "error"
			}`,
			expected: InitializationOptions{
				Profile:           "production",
				ConfigPath:        "/path/to/config.yaml",
				Engines:           EngineToggles{Fmt: true, Style: false, Lint: true, Policy: false},
				SeverityThreshold: "error",
			},
		},
		{
			name: "empty options",
			json: `{}`,
			expected: InitializationOptions{
				Engines: EngineToggles{}, // all false
			},
		},
		{
			name: "partial options",
			json: `{
				"profile": "dev",
				"engines": {"style": true}
			}`,
			expected: InitializationOptions{
				Profile: "dev",
				Engines: EngineToggles{Style: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts InitializationOptions
			err := json.Unmarshal([]byte(tt.json), &opts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, opts)
		})
	}
}

func TestEngineToggles_JSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected EngineToggles
	}{
		{
			name:     "all true",
			json:     `{"fmt": true, "style": true, "lint": true, "policy": true}`,
			expected: EngineToggles{Fmt: true, Style: true, Lint: true, Policy: true},
		},
		{
			name:     "all false",
			json:     `{"fmt": false, "style": false, "lint": false, "policy": false}`,
			expected: EngineToggles{Fmt: false, Style: false, Lint: false, Policy: false},
		},
		{
			name:     "empty object defaults to false",
			json:     `{}`,
			expected: EngineToggles{},
		},
		{
			name:     "mixed values",
			json:     `{"fmt": true, "policy": true}`,
			expected: EngineToggles{Fmt: true, Policy: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var toggles EngineToggles
			err := json.Unmarshal([]byte(tt.json), &toggles)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, toggles)
		})
	}
}

func TestInitializeParams_JSONParsing(t *testing.T) {
	input := `{
		"processId": 12345,
		"rootUri": "file:///workspace",
		"capabilities": {
			"textDocument": {
				"synchronization": {
					"didSave": true
				}
			}
		},
		"initializationOptions": {
			"profile": "ci",
			"severityThreshold": "warning"
		}
	}`

	var params InitializeParams
	err := json.Unmarshal([]byte(input), &params)
	require.NoError(t, err)

	assert.Equal(t, 12345, params.ProcessID)
	assert.Equal(t, "file:///workspace", params.RootURI)
	assert.True(t, params.Capabilities.TextDocument.Synchronization.DidSave)
	require.NotNil(t, params.InitializationOptions)
	assert.Equal(t, "ci", params.InitializationOptions.Profile)
	assert.Equal(t, "warning", params.InitializationOptions.SeverityThreshold)
}

func TestDiagnostic_JSONRoundtrip(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 10, Character: 5},
			End:   Position{Line: 10, Character: 20},
		},
		Severity: 2, // Warning
		Code:     "style.resource-name-convention",
		Source:   "terratidy",
		Message:  "Resource name should be snake_case",
	}

	// Marshal to JSON
	data, err := json.Marshal(diag)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Diagnostic
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, diag, decoded)
}

func TestTextEdit_JSONRoundtrip(t *testing.T) {
	edit := TextEdit{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 100, Character: 0},
		},
		NewText: "formatted content\n",
	}

	data, err := json.Marshal(edit)
	require.NoError(t, err)

	var decoded TextEdit
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, edit, decoded)
}

func TestCodeAction_JSONRoundtrip(t *testing.T) {
	action := CodeAction{
		Title:       "Fix: style.resource-name-convention",
		Kind:        "quickfix",
		IsPreferred: true,
		Edit: &WorkspaceEdit{
			Changes: map[string][]TextEdit{
				"file:///test.tf": {
					{
						Range: Range{
							Start: Position{Line: 0, Character: 0},
							End:   Position{Line: 10, Character: 0},
						},
						NewText: "fixed content",
					},
				},
			},
		},
	}

	data, err := json.Marshal(action)
	require.NoError(t, err)

	var decoded CodeAction
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, action.Title, decoded.Title)
	assert.Equal(t, action.Kind, decoded.Kind)
	assert.Equal(t, action.IsPreferred, decoded.IsPreferred)
	require.NotNil(t, decoded.Edit)
	assert.Len(t, decoded.Edit.Changes, 1)
}

func TestByteRangeToLSPRange(t *testing.T) {
	tests := []struct {
		name    string
		content string
		start   int
		end     int
		want    Range
	}{
		{
			name:    "single-line edit",
			content: "resource \"x\" {}",
			start:   9,
			end:     12,
			want: Range{
				Start: Position{Line: 0, Character: 9},
				End:   Position{Line: 0, Character: 12},
			},
		},
		{
			name:    "multi-line edit",
			content: "line0\nline1\nline2\n",
			start:   3,
			end:     14,
			want: Range{
				Start: Position{Line: 0, Character: 3},
				End:   Position{Line: 2, Character: 2},
			},
		},
		{
			name:    "edit at file start",
			content: "abc\ndef",
			start:   0,
			end:     0,
			want: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
		},
		{
			name:    "edit spanning final character",
			content: "abc\ndef",
			start:   4,
			end:     7,
			want: Range{
				Start: Position{Line: 1, Character: 0},
				End:   Position{Line: 1, Character: 3},
			},
		},
		{
			name:    "empty file",
			content: "",
			start:   0,
			end:     0,
			want: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
		},
		{
			name:    "CRLF content treats CR as line-trailing byte",
			content: "abc\r\nxyz",
			start:   3, // CR position — still on line 0, character 3
			end:     5, // first byte of line 1 ('x') — character 0
			want: Range{
				Start: Position{Line: 0, Character: 3},
				End:   Position{Line: 1, Character: 0},
			},
		},
		{
			name:    "end-of-line position with no trailing newline",
			content: "alpha",
			start:   5,
			end:     5,
			want: Range{
				Start: Position{Line: 0, Character: 5},
				End:   Position{Line: 0, Character: 5},
			},
		},
		{
			name:    "offset at line start of newline-terminated file",
			content: "abc\n",
			start:   4,
			end:     4,
			want: Range{
				Start: Position{Line: 1, Character: 0},
				End:   Position{Line: 1, Character: 0},
			},
		},
		{
			name:    "negative offset clamps to file start",
			content: "abc",
			start:   -5,
			end:     -1,
			want: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
		},
		{
			// U+2014 em-dash is 3 UTF-8 bytes but 1 UTF-16 code unit. Byte
			// offset 8 is the start of "dash" — UTF-16 character is 6
			// ("// em" is 5 + em-dash is 1).
			name:    "BMP rune (em-dash) counts as one UTF-16 code unit",
			content: "// em—dash\nresource",
			start:   8,
			end:     12,
			want: Range{
				Start: Position{Line: 0, Character: 6},
				End:   Position{Line: 0, Character: 10},
			},
		},
		{
			// U+1F600 (grinning face) is 4 UTF-8 bytes and 2 UTF-16 code
			// units (surrogate pair). Byte offset 7 is the start of "!" —
			// UTF-16 character is 5 ("hi " is 3 + emoji is 2).
			name:    "supplementary-plane rune (emoji) counts as two UTF-16 code units",
			content: "hi \U0001F600!\n",
			start:   7,
			end:     8,
			want: Range{
				Start: Position{Line: 0, Character: 5},
				End:   Position{Line: 0, Character: 6},
			},
		},
		{
			// CJK ideograph U+4E2D ("中") is 3 UTF-8 bytes but 1 UTF-16
			// code unit. End-of-line (the '\n' itself) is byte 8 → UTF-16
			// character 4 ("a中文b" is 4 code units).
			name:    "CJK characters count one UTF-16 code unit each",
			content: "a中文b\n",
			start:   0,
			end:     8,
			want: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 4},
			},
		},
		{
			// Offset past EOF clamps; UTF-16 character on the final line is
			// computed correctly even when offset overshoots.
			name:    "offset past end-of-file clamps",
			content: "abc",
			start:   100,
			end:     200,
			want: Range{
				Start: Position{Line: 0, Character: 3},
				End:   Position{Line: 0, Character: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := byteRangeToLSPRange([]byte(tt.content), tt.start, tt.end)
			assert.Equal(t, tt.want, got)
		})
	}
}
