package dependency_test

import (
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
		err := validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
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
		err := validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "An error occurred when comparing the versions")
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
		err := validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsed Version does not fulfill version requirement of")
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
		err := validator.ValidateAllDependencies(dogu)

		// then
		require.NoError(t, err)
	})
}
