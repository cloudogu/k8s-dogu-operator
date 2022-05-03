package dependency_test

import (
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewOperatorDependencyValidator(t *testing.T) {
	// given
	version, err := core.ParseVersion("0.0.1")
	require.NoError(t, err)

	// when
	validator := dependency.NewOperatorDependencyValidator(&version)

	// then
	assert.NotNil(t, validator)
}

func TestOperatorDependencyValidator_ValidateAllDependencies(t *testing.T) {
	t.Run("error on not parsable mandatory dependency operation", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("0.0.1")
		require.NoError(t, err)
		validator := dependency.NewOperatorDependencyValidator(&version)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: "-1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: "-1.0.0",
			}},
		}

		// when
		err = validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse dependency version")
	})

	t.Run("error on invalid mandatory dependency operator", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("0.0.1")
		require.NoError(t, err)
		validator := dependency.NewOperatorDependencyValidator(&version)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: ">>1.0.0",
			}},
			OptionalDependencies: []core.Dependency{},
		}

		// when
		err = validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compare dependency version with operator version")
	})

	t.Run("error on invalid mandatory dependency", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("0.0.1")
		require.NoError(t, err)
		validator := dependency.NewOperatorDependencyValidator(&version)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: ">=1.0.0",
			}},
			OptionalDependencies: []core.Dependency{},
		}

		// when
		err = validator.ValidateAllDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsed Version does not fulfill version requirement of 0.0.1 dogu k8s-dogu-operator")
	})

	t.Run("success on mandatory and optrional dependency", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("1.5.0")
		require.NoError(t, err)
		validator := dependency.NewOperatorDependencyValidator(&version)
		dogu := &core.Dogu{
			Name:    "dogu",
			Version: "1.0.0",
			Dependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: ">=1.0.0",
			}},
			OptionalDependencies: []core.Dependency{{
				Type:    "client",
				Name:    "k8s-dogu-operator",
				Version: ">=1.0.0",
			}},
		}

		// when
		err = validator.ValidateAllDependencies(dogu)

		// then
		require.NoError(t, err)
	})
}
