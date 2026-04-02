package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestRunnerWithResults_Parallel(t *testing.T) {
	ctx := context.Background()

	engine1 := &mockEngine{name: "engine1", findings: []sdk.Finding{{Rule: "rule1"}}}
	engine2 := &mockEngine{name: "engine2", findings: []sdk.Finding{{Rule: "rule2"}}}

	runner := New().AddEngine(engine1).AddEngine(engine2).SetParallel(true)

	results := runner.RunWithResults(ctx, []string{"test.tf"})
	require.Len(t, results, 2)

	// Results may be in any order for parallel execution
	engines := map[string]bool{}
	for _, r := range results {
		engines[r.Engine] = true
		assert.NoError(t, r.Error)
	}
	assert.True(t, engines["engine1"])
	assert.True(t, engines["engine2"])
}

func TestRunnerZeroEngines(t *testing.T) {
	ctx := context.Background()
	runner := New()

	findings, err := runner.Run(ctx, []string{"test.tf"})
	require.NoError(t, err)
	assert.Empty(t, findings)

	results := runner.RunWithResults(ctx, []string{"test.tf"})
	assert.Empty(t, results)
}

func TestRunnerZeroEngines_Parallel(t *testing.T) {
	ctx := context.Background()
	runner := New().SetParallel(true)

	findings, err := runner.Run(ctx, []string{"test.tf"})
	require.NoError(t, err)
	assert.Empty(t, findings)
}
