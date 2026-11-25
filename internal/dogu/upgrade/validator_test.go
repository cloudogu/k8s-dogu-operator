package upgrade

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/dogu-lib/doguv3"
	doguv3reg "github.com/cloudogu/dogu-lib/doguv3/doguregistry"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var targetIdentifier = doguv3.Identifier{DoguNamespace: "official", Name: "nexus", Version: "2.0.0"}

// validator returns a Validator whose detector reads the given HelmRelease.
func validator(t *testing.T, registry DoguRegistryReader, release *fluxhelm.HelmRelease) *Validator {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, fluxhelm.AddToScheme(scheme))
	builder := fake.NewClientBuilder().WithScheme(scheme)
	if release != nil {
		builder = builder.WithObjects(release)
	}
	return NewValidator(NewDetector(builder.Build()), registry)
}

func deployed(version string) *fluxhelm.HelmRelease {
	return helmRelease(snapshot(1, version, "deployed"))
}

func TestValidator_Validate(t *testing.T) {
	t.Run("upgrade allowed by the descriptor of the target version is valid", func(t *testing.T) {
		registry := NewMockDoguRegistryReader(t)
		registry.EXPECT().Get(context.Background(), targetIdentifier).Return(&doguv3.Dogu{Upgrades: adrDescriptor}, nil)

		result, err := validator(t, registry, deployed("1.8.0")).Validate(context.Background(), testDogu("2.0.0"), false)

		require.NoError(t, err)
		assert.Equal(t, Result{
			Change:  Change{Kind: Upgrade, Installed: "1.8.0", Target: "2.0.0"},
			Verdict: Valid,
			Message: "Upgrade from version 1.8.0 to 2.0.0 is valid",
		}, result)
	})

	t.Run("dogu without version change is unchanged", func(t *testing.T) {
		result, err := validator(t, NewMockDoguRegistryReader(t), deployed("2.0.0")).Validate(context.Background(), testDogu("2.0.0"), false)

		require.NoError(t, err)
		assert.Equal(t, Unchanged, result.Verdict)
		assert.Equal(t, NoCause, result.Cause)
	})

	t.Run("invalid version", func(t *testing.T) {
		refusedWithoutRegistryTests := []struct {
			name        string
			installed   string
			target      string
			wantKind    Kind
			wantCause   Cause
			wantMessage string
		}{
			{
				name: "downgrade", installed: "2.0.0", target: "1.5.0", wantKind: Downgrade, wantCause: CauseDowngrade,
				wantMessage: "Downgrade from version 2.0.0 to 1.5.0 is not allowed",
			},
			{
				name: "invalid version", installed: "1.5.0", target: "two", wantKind: InvalidVersion, wantCause: CauseInvalidVersion,
				wantMessage: "Version change from 1.5.0 to two is not allowed because a version is not a valid semantic version",
			},
		}

		for _, tt := range refusedWithoutRegistryTests {
			for _, devChart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s is refused without asking the dogu registry (dev chart: %t)", tt.name, devChart), func(t *testing.T) {
					result, err := validator(t, NewMockDoguRegistryReader(t), deployed(tt.installed)).Validate(context.Background(), testDogu(tt.target), devChart)

					require.NoError(t, err)
					assert.Equal(t, Result{
						Change:  Change{Kind: tt.wantKind, Installed: tt.installed, Target: tt.target},
						Verdict: Refused,
						Cause:   tt.wantCause,
						Message: tt.wantMessage,
					}, result)
				})
			}
		}
	})

	t.Run("upgrade of a dev chart is skipped without asking the dogu registry", func(t *testing.T) {
		result, err := validator(t, NewMockDoguRegistryReader(t), deployed("1.5.0")).Validate(context.Background(), testDogu("2.0.0"), true)

		require.NoError(t, err)
		assert.Equal(t, Result{
			Change:  Change{Kind: Upgrade, Installed: "1.5.0", Target: "2.0.0"},
			Verdict: Skipped,
			Message: "Upgrade from version 1.5.0 to 2.0.0 is not validated because the dogu uses a dev chart",
		}, result)
	})

	t.Run("upgrade of a dogu without descriptor is valid", func(t *testing.T) {
		registry := NewMockDoguRegistryReader(t)
		registry.EXPECT().Get(context.Background(), targetIdentifier).Return(&doguv3.Dogu{}, nil)

		result, err := validator(t, registry, deployed("0.1.0")).Validate(context.Background(), testDogu("2.0.0"), false)

		require.NoError(t, err)
		assert.Equal(t, Valid, result.Verdict)
	})

	t.Run("refused by descriptor", func(t *testing.T) {
		refusedByDescriptorTests := []struct {
			name      string
			installed string
			upgrades  []doguv3.Upgrade
			wantCause Cause
		}{
			{name: "without upgrade path", installed: "0.9.0", upgrades: adrDescriptor, wantCause: CauseInvalidUpgradePath},
			{name: "across several upgrade steps", installed: "1.5.0", upgrades: adrDescriptor, wantCause: CauseInvalidUpgradePath},
			{name: "with an invalid descriptor", installed: "1.5.0", upgrades: []doguv3.Upgrade{{From: ">=a.b", To: "2.0.0"}}, wantCause: CauseInvalidDescriptor},
		}

		for _, tt := range refusedByDescriptorTests {
			t.Run("upgrade "+tt.name+" is refused", func(t *testing.T) {
				registry := NewMockDoguRegistryReader(t)
				registry.EXPECT().Get(context.Background(), targetIdentifier).Return(&doguv3.Dogu{Upgrades: tt.upgrades}, nil)

				result, err := validator(t, registry, deployed(tt.installed)).Validate(context.Background(), testDogu("2.0.0"), false)

				require.NoError(t, err)
				assert.Equal(t, Refused, result.Verdict)
				assert.Equal(t, tt.wantCause, result.Cause)
				assert.Contains(t, result.Message, "Upgrade from version "+tt.installed+" to 2.0.0 is not allowed: ")
			})
		}
	})

	t.Run("refused by registry", func(t *testing.T) {
		permanentRegistryErrors := map[string]error{
			"not found":    doguv3reg.NewNotFoundError(errors.New("not found")),
			"unauthorized": doguv3reg.NewUnauthorizedError(errors.New("unauthorized")),
			"forbidden":    doguv3reg.NewForbiddenError(errors.New("forbidden")),
		}

		for name, registryErr := range permanentRegistryErrors {
			t.Run("upgrade is refused if the dogu registry answers "+name, func(t *testing.T) {
				registry := NewMockDoguRegistryReader(t)
				registry.EXPECT().Get(context.Background(), targetIdentifier).Return(nil, registryErr)

				result, err := validator(t, registry, deployed("1.5.0")).Validate(context.Background(), testDogu("2.0.0"), false)

				require.NoError(t, err)
				assert.Equal(t, Refused, result.Verdict)
				assert.Equal(t, CauseDescriptorUnavailable, result.Cause)
				assert.Contains(t, result.Message, "failed to get upgrade descriptor for identifier official/nexus:2.0.0")
			})
		}
	})

	t.Run("reconcile due to registry errors", func(t *testing.T) {
		temporaryRegistryErrors := map[string]error{
			"connection": doguv3reg.NewConnectionError(errors.New("connection refused")),
			"generic":    doguv3reg.NewGenericError(errors.New("failed to read response body")),
		}

		for name, registryErr := range temporaryRegistryErrors {
			t.Run("temporary "+name+" error of the dogu registry is returned as unavailable descriptor", func(t *testing.T) {
				registry := NewMockDoguRegistryReader(t)
				registry.EXPECT().Get(context.Background(), targetIdentifier).Return(nil, registryErr)

				_, err := validator(t, registry, deployed("1.5.0")).Validate(context.Background(), testDogu("2.0.0"), false)

				assert.ErrorIs(t, err, ErrDescriptorUnavailable)
				assert.ErrorIs(t, err, registryErr)
			})
		}
	})

	t.Run("failing HelmRelease read is returned without unavailable descriptor", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, fluxhelm.AddToScheme(scheme))
		failingClient := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return assert.AnError
			},
		}).Build()

		_, err := NewValidator(NewDetector(failingClient), NewMockDoguRegistryReader(t)).Validate(context.Background(), testDogu("2.0.0"), false)

		assert.ErrorIs(t, err, assert.AnError)
		assert.NotErrorIs(t, err, ErrDescriptorUnavailable)
	})
}
