package dependency_test

import (
	"context"
	regLibErr "github.com/cloudogu/k8s-registry-lib/errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/dependency"
	cloudoguMocks "github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu/mocks"
)

var testCtx = context.Background()

func TestNewDoguDependencyValidator(t *testing.T) {
	// given
	localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)

	// when
	validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)

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
		localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)
		localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)
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
		err := validator.ValidateAllDependencies(testCtx, dogu)

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
		localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)
		localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)
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
		err := validator.ValidateAllDependencies(testCtx, dogu)

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
		localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)
		localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)
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
		err := validator.ValidateAllDependencies(testCtx, dogu)

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
		localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)
		localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "redmine").Return(redmineDogu, nil)
		validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)
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
		err := validator.ValidateAllDependencies(testCtx, dogu)

		// then
		require.NoError(t, err)
	})
}

func Test_doguDependencyValidator_checkDoguDependency(t *testing.T) {
	t.Run("should return nil if optional and a k8s not found error", func(t *testing.T) {
		// given
		redmineDogu := &core.Dogu{
			Name:                 "redmine",
			Version:              "1.1.0",
			OptionalDependencies: []core.Dependency{{Type: "dogu", Name: "test"}},
		}
		localDoguFetcherMock := cloudoguMocks.NewMockLocalDoguFetcher(t)
		localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "test").Return(redmineDogu, regLibErr.NewNotFoundError(assert.AnError))
		validator := dependency.NewDoguDependencyValidator(localDoguFetcherMock)

		// when
		err := validator.ValidateAllDependencies(testCtx, redmineDogu)

		// then
		require.Nil(t, err)
	})
}
