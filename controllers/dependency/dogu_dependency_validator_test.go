package dependency_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
)

func TestNewDoguDependencyValidator(t *testing.T) {
	// given
	cesRegistryMock := &cesmocks.DoguRegistry{}

	// when
	validator := dependency.NewDoguDependencyValidator(cesRegistryMock)

	// then
	assert.NotNil(t, validator)
}

func TestDoguDependencyValidator_ValidateAllDependencies(t *testing.T) {
	ctx := context.Background()

	t.Run("error on not parsable mandatory dependency operation", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "1.0.0",
		}
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(cesRegistryMock)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: "-1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: "-1.0.0",
			}},
		}

		// when
		err := validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse")
	})

	t.Run("error on invalid mandatory dependency operator", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "1.0.0",
		}
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(cesRegistryMock)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">>1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">>1.0.0",
			}},
		}

		// when
		err := validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "an error occurred when comparing the versions")
	})

	t.Run("error on invalid mandatory dependency", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "0.9.0",
		}
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(cesRegistryMock)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">=1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">=1.0.0",
			}},
		}

		// when
		err := validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "parsed Version does not fulfill version requirement of")
	})

	t.Run("success on mandatory and optional dependency", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "1.1.0",
		}
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(cesRegistryMock)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">=1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "dogu",
				Name:    "redmine",
				Version: ">=1.0.0",
			}},
		}

		// when
		err := validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.NoError(t, err)
	})
}
