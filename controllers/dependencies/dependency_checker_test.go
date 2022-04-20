package dependencies_test

import (
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependencies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func getDoguWithDependencies(dependencies []core.Dependency, optionalDependencies []core.Dependency) *core.Dogu {
	return &core.Dogu{
		Name:                 "dogu",
		Version:              "1.0.0",
		Dependencies:         dependencies,
		OptionalDependencies: optionalDependencies,
	}
}

func TestDependencyChecker_ValidateDependencies(t *testing.T) {
	t.Run("successfully validate dogu dependencies for a dependency depth of one", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "1.0.0",
		}
		cockpitDogu := &core.Dogu{
			Name:    "cockpit",
			Version: "1.0.0",
		}

		version, _ := core.ParseVersion("0.5.0")
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		cesRegistryMock.Mock.On("Get", "cockpit").Return(cockpitDogu, nil)
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)
		dep := []core.Dependency{{
			Type:    "dogu",
			Name:    "redmine",
			Version: ">=1.0.0",
		}, {
			Type:    "client",
			Name:    "k8s-dogu-operator",
			Version: "=0.5.0",
		}}
		optionalDependencies := []core.Dependency{{
			Type:    "dogu",
			Name:    "cockpit",
			Version: ">=1.0.0",
		}}
		dogu := getDoguWithDependencies(dep, optionalDependencies)

		// when
		err := dependencyChecker.ValidateDependencies(dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("error on invalid dogu dependency version", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:    "redmine",
			Version: "1.0.0",
		}
		version, _ := core.ParseVersion("0.0.0")
		cesRegistryMock := &cesmocks.DoguRegistry{}
		cesRegistryMock.Mock.On("Get", "redmine").Return(redmineDogu, nil)
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)
		dep := []core.Dependency{{
			Type:    "dogu",
			Name:    "redmine",
			Version: "-->1.0.0",
		}}
		dogu := getDoguWithDependencies(dep, []core.Dependency{})

		// when
		err := dependencyChecker.ValidateDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse version")
	})

	t.Run("error on invalid client dependency version", func(t *testing.T) {
		// given
		version, _ := core.ParseVersion("0.0.0")
		cesRegistryMock := &cesmocks.DoguRegistry{}
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)
		dep := []core.Dependency{{
			Type:    "client",
			Name:    "k8s-dogu-operator",
			Version: "-->1.0.0",
		}}
		dogu := getDoguWithDependencies(dep, []core.Dependency{})

		// when
		err := dependencyChecker.ValidateDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse version")
	})

	t.Run("error on invalid optional client dependency", func(t *testing.T) {
		// given
		version, _ := core.ParseVersion("1.0.0")
		cesRegistryMock := &cesmocks.DoguRegistry{}
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)
		optionalDeps := []core.Dependency{{
			Type:    "client",
			Name:    "k8s-dogu-operator",
			Version: ">1.0.1",
		}}
		dogu := getDoguWithDependencies([]core.Dependency{}, optionalDeps)

		// when
		err := dependencyChecker.ValidateDependencies(dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("error on invalid mandatory client dependency", func(t *testing.T) {
		// given
		version, _ := core.ParseVersion("1.0.0")
		cesRegistryMock := &cesmocks.DoguRegistry{}
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)
		deps := []core.Dependency{{
			Type:    "client",
			Name:    "k8s-dogu-operator",
			Version: ">1.0.1",
		}}
		dogu := getDoguWithDependencies(deps, []core.Dependency{})

		// when
		err := dependencyChecker.ValidateDependencies(dogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "1.0.0 parsed Version does not fulfill version requirement of >1.0.1 for client k8s-dogu-operator")
	})
}

func TestNewDependencyChecker(t *testing.T) {
	t.Run("successfully create new checker", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("0.0.0")
		require.NoError(t, err)

		cesRegistryMock := &cesmocks.DoguRegistry{}

		// when
		dependencyChecker := dependencies.NewDependencyChecker(&version, cesRegistryMock)

		// then
		assert.NotNil(t, dependencyChecker)
		assert.NotNil(t, dependencyChecker.OperatorDependencyValidator)
		assert.Equal(t, &version, dependencyChecker.OperatorDependencyValidator.Version)
		assert.Equal(t, cesRegistryMock, dependencyChecker.DoguRegistry)
	})
}
