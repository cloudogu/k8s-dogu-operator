package migration

import (
	"context"
	"fmt"
	"maps"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
)

type valueAssembler interface {
	Assemble(ctx context.Context, cr *v3beta1.Dogu, valuesMeta, runtimePatchTemplate []byte) (values.Values, error)
}

type chartInspector interface {
	HasMigrationJob(ctx context.Context, doguResource *v3beta1.Dogu) (bool, error)
}

// JobValuesAssembler enables the chart's migration jobs on top of the regular values.
// The operator owns the flag, so it overrides the dogu's own values.
type JobValuesAssembler struct {
	assembler valueAssembler
	charts    chartInspector
}

// NewJobValuesAssembler creates a JobValuesAssembler.
func NewJobValuesAssembler(assembler valueAssembler, charts chartInspector) *JobValuesAssembler {
	return &JobValuesAssembler{assembler: assembler, charts: charts}
}

// Assemble returns the regular values with migrationJob.enabled set to true, if the chart ships migration jobs.
func (a *JobValuesAssembler) Assemble(ctx context.Context, cr *v3beta1.Dogu, valuesMeta, runtimePatchTemplate []byte) (values.Values, error) {
	result, err := a.assembler.Assemble(ctx, cr, valuesMeta, runtimePatchTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble values: %w", err)
	}

	hasMigrationJob, err := a.charts.HasMigrationJob(ctx, cr)
	if err != nil {
		return nil, fmt.Errorf("failed to check chart for migration jobs: %w", err)
	}

	if !hasMigrationJob {
		return result, nil
	}

	migrationJob, _ := result[values.MigrationJobKey].(values.Values)
	migrationJob = maps.Clone(migrationJob)
	if migrationJob == nil {
		migrationJob = values.Values{}
	}

	migrationJob["enabled"] = true
	result[values.MigrationJobKey] = migrationJob

	return result, nil
}
