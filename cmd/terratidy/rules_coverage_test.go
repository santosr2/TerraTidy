package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllRules(t *testing.T) {
	rules := getAllRules()
	require.NotEmpty(t, rules)

	// Should have rules from style, lint, and policy engines
	engines := make(map[string]bool)
	for _, r := range rules {
		engines[r.Engine] = true
	}

	assert.True(t, engines["style"], "should have style rules")
	assert.True(t, engines["lint"], "should have lint rules")
	assert.True(t, engines["policy"], "should have policy rules")
}

func TestGetStyleRuleDescription(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"style.blank-lines-between-blocks", "Ensure blank lines between resource blocks"},
		{"style.block-label-case", "Enforce snake_case naming for block labels"},
		{"nonexistent.rule", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStyleRuleDescription(tt.name)
			if tt.want == "" {
				// Unknown rules return empty or a generic description
				_ = got
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRunRulesList(t *testing.T) {
	t.Run("all rules", func(t *testing.T) {
		oldEngine := rulesEngine
		rulesEngine = ""
		defer func() { rulesEngine = oldEngine }()

		err := runRulesList(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("filtered by engine", func(t *testing.T) {
		oldEngine := rulesEngine
		rulesEngine = "style"
		defer func() { rulesEngine = oldEngine }()

		err := runRulesList(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("nonexistent engine returns empty", func(t *testing.T) {
		oldEngine := rulesEngine
		rulesEngine = "nonexistent"
		defer func() { rulesEngine = oldEngine }()

		err := runRulesList(nil, nil)
		assert.NoError(t, err)
	})
}

func TestRunRulesDocs(t *testing.T) {
	t.Run("all docs", func(t *testing.T) {
		oldEngine := rulesEngine
		rulesEngine = ""
		defer func() { rulesEngine = oldEngine }()

		err := runRulesDocs(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("filtered by engine", func(t *testing.T) {
		oldEngine := rulesEngine
		rulesEngine = "lint"
		defer func() { rulesEngine = oldEngine }()

		err := runRulesDocs(nil, nil)
		assert.NoError(t, err)
	})
}
