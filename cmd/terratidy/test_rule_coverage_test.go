package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestMatchesFinding(t *testing.T) {
	finding := sdk.Finding{
		Rule:     "style.blank-line-between-blocks",
		Severity: sdk.SeverityWarning,
		Message:  "Missing blank line between blocks",
	}

	tests := []struct {
		name     string
		expected ExpectedFinding
		want     bool
	}{
		{"exact rule match", ExpectedFinding{Rule: "blank-line-between-blocks"}, true},
		{"rule prefix no match", ExpectedFinding{Rule: "other-rule"}, false},
		{"severity match", ExpectedFinding{Severity: "warning"}, true},
		{"severity mismatch", ExpectedFinding{Severity: "error"}, false},
		{"message contains", ExpectedFinding{Message: "blank line"}, true},
		{"message not contains", ExpectedFinding{Message: "something else"}, false},
		{"all fields match", ExpectedFinding{Rule: "blank-line-between-blocks", Severity: "warning", Message: "blank line"}, true},
		{"empty expected matches all", ExpectedFinding{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesFinding(tt.expected, finding))
		})
	}
}
