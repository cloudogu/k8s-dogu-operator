package dependency_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/dependency"
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
	ctx := context.Background()

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
		err = validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse dependency version")
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
		err = validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to compare dependency version with operator version")
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
		err = validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "parsed version does not fulfill version requirement of 0.0.1 dogu k8s-dogu-operator")
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
		err = validator.ValidateAllDependencies(ctx, dogu)

		// then
		require.NoError(t, err)
	})
}
