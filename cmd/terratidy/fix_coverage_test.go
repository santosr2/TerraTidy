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
	fixFunc := func() ([]byte, error) { return nil, nil }

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"not fixable", []sdk.Finding{{Fixable: false}}, 0},
		{"fixable but no func", []sdk.Finding{{Fixable: true, FixFunc: nil}}, 0},
		{"fixable with func", []sdk.Finding{{Fixable: true, FixFunc: fixFunc}}, 1},
		{"mixed", []sdk.Finding{
			{Fixable: true, FixFunc: fixFunc},
			{Fixable: false},
			{Fixable: true, FixFunc: fixFunc},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countFixedStyleIssues(tt.findings))
		})
	}
}

func TestCountRemainingIssues(t *testing.T) {
	fixFunc := func() ([]byte, error) { return nil, nil }

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"all fixable", []sdk.Finding{{Fixable: true, FixFunc: fixFunc}}, 0},
		{"not fixable", []sdk.Finding{{Fixable: false}}, 1},
		{"fixable no func", []sdk.Finding{{Fixable: true, FixFunc: nil}}, 1},
		{"mixed", []sdk.Finding{
			{Fixable: true, FixFunc: fixFunc},
			{Fixable: false},
			{Fixable: true, FixFunc: nil},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countRemainingIssues(tt.findings))
		})
	}
}
