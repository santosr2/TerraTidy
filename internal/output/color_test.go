package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveColor(t *testing.T) {
	tests := []struct {
		name              string
		requested         bool
		requestedExplicit bool
		noColorEnv        string
		forceColorEnv     string
		stdoutIsTerminal  bool
		want              bool
	}{
		{
			name:             "not a terminal, no env: color disabled",
			requested:        true,
			stdoutIsTerminal: false,
			want:             false,
		},
		{
			name:             "is a terminal, no env: color enabled",
			requested:        true,
			stdoutIsTerminal: true,
			want:             true,
		},
		{
			name:             "NO_COLOR set on a terminal: color disabled",
			noColorEnv:       "1",
			stdoutIsTerminal: true,
			want:             false,
		},
		{
			name:             "NO_COLOR set, not a terminal: color disabled",
			noColorEnv:       "1",
			stdoutIsTerminal: false,
			want:             false,
		},
		{
			name:             "NO_COLOR set to empty string: does not disable",
			noColorEnv:       "",
			stdoutIsTerminal: true,
			want:             true,
		},
		{
			name:             "FORCE_COLOR set, not a terminal: color enabled",
			forceColorEnv:    "1",
			stdoutIsTerminal: false,
			want:             true,
		},
		{
			name:             "NO_COLOR wins over FORCE_COLOR when both set",
			noColorEnv:       "1",
			forceColorEnv:    "1",
			stdoutIsTerminal: false,
			want:             false,
		},
		{
			name:              "explicit --color=true wins over NO_COLOR",
			requested:         true,
			requestedExplicit: true,
			noColorEnv:        "1",
			stdoutIsTerminal:  false,
			want:              true,
		},
		{
			name:              "explicit --color=false wins over a terminal and FORCE_COLOR",
			requested:         false,
			requestedExplicit: true,
			forceColorEnv:     "1",
			stdoutIsTerminal:  true,
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveColor(tt.requested, tt.requestedExplicit, tt.noColorEnv, tt.forceColorEnv, tt.stdoutIsTerminal)
			assert.Equal(t, tt.want, got)
		})
	}
}
