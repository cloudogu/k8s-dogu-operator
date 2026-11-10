package migration

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func jobValuesAssembler(t *testing.T, assembled values.Values, assembleErr error, hasMigrationJob bool, chartErr error) *JobValuesAssembler {
	t.Helper()
	assembler := newMockValueAssembler(t)
	assembler.EXPECT().Assemble(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assembled, assembleErr)
	charts := newMockChartInspector(t)
	if assembleErr == nil {
		charts.EXPECT().HasMigrationJob(mock.Anything, mock.Anything).Return(hasMigrationJob, chartErr)
	}

	return NewJobValuesAssembler(assembler, charts)
}

func TestJobValuesAssembler_Assemble(t *testing.T) {
	ctx := context.Background()

	t.Run("enables the migration jobs of a chart that ships them", func(t *testing.T) {
		assembler := jobValuesAssembler(t, values.Values{
			"replicas":     1,
			"migrationJob": values.Values{"enabled": false, "image": "migrate"},
		}, nil, true, nil)

		result, err := assembler.Assemble(ctx, v3Dogu("nexus"), nil, nil)

		require.NoError(t, err)
		assert.Equal(t, values.Values{
			"replicas":     1,
			"migrationJob": values.Values{"enabled": true, "image": "migrate"},
		}, result)
	})

	t.Run("leaves the values of a chart without migration jobs untouched", func(t *testing.T) {
		assembler := jobValuesAssembler(t, values.Values{"replicas": 1}, nil, false, nil)

		result, err := assembler.Assemble(ctx, v3Dogu("nexus"), nil, nil)

		require.NoError(t, err)
		assert.Equal(t, values.Values{"replicas": 1}, result)
		assert.Len(t, result, 1)
	})

	t.Run("propagates assembler error", func(t *testing.T) {
		assembler := jobValuesAssembler(t, nil, assert.AnError, false, nil)

		_, err := assembler.Assemble(ctx, v3Dogu("nexus"), nil, nil)

		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("propagates chart error", func(t *testing.T) {
		assembler := jobValuesAssembler(t, values.Values{}, nil, false, assert.AnError)

		_, err := assembler.Assemble(ctx, v3Dogu("nexus"), nil, nil)

		assert.ErrorIs(t, err, assert.AnError)
	})
}
