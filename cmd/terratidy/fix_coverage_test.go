package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestCountFormattedFiles(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"empty", []sdk.Finding{}, 0},
		{"no formatted", []sdk.Finding{{Rule: "style.something"}}, 0},
		{"one formatted", []sdk.Finding{{Rule: "fmt.formatted"}}, 1},
		{"mixed", []sdk.Finding{
			{Rule: "fmt.formatted"},
			{Rule: "style.blank-line"},
			{Rule: "fmt.formatted"},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countFormattedFiles(tt.findings))
		})
	}
}

func TestCountFixedStyleIssues(t *testing.T) {
	fixResult := &sdk.FixResult{Content: []byte("fixed")}

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"not fixable", []sdk.Finding{{Fix: nil}}, 0},
		{"fixable with fix", []sdk.Finding{{Fix: fixResult}}, 1},
		{"mixed", []sdk.Finding{
			{Fix: fixResult},
			{Fix: nil},
			{Fix: fixResult},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countFixedStyleIssues(tt.findings))
		})
	}
}

func TestCountRemainingIssues(t *testing.T) {
	fixResult := &sdk.FixResult{Content: []byte("fixed")}

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"all fixable", []sdk.Finding{{Fix: fixResult}}, 0},
		{"not fixable", []sdk.Finding{{Fix: nil}}, 1},
		{"mixed", []sdk.Finding{
			{Fix: fixResult},
			{Fix: nil},
			{Fix: nil},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countRemainingIssues(tt.findings))
		})
	}
}
