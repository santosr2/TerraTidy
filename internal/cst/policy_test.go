package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultPolicies pins the two preset policies so a silent flip — e.g.
// someone "simplifying" both defaults to the same value — is caught here
// rather than as a behavior change inside Build. The presets are the entire
// public contract of this file; once Build lands in the next sub-task, every
// downstream rule depends on these returning what the doc comments promise.
func TestDefaultPolicies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  Policy
		want Policy
	}{
		{"top-level is strict", DefaultTopLevelPolicy(), Policy{StrictAdjacency: true}},
		{"block-body is passthrough", DefaultBlockBodyPolicy(), Policy{StrictAdjacency: false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.got)
		})
	}
}
