package install

import (
	"context"
	"errors"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewAuthRegistrationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewAuthRegistrationStep(
			newMockAuthRegistrationManager(t),
			newMockLocalDoguFetcher(t),
			&config.OperatorConfig{AuthRegistrationEnabled: true},
		)

		assert.NotNil(t, step)
	})
}

func TestAuthRegistrationStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguName := cescommons.SimpleName("test")
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue if dogu descriptor fetch fails", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchInstalled(testCtx, doguName).Return(nil, assert.AnError)

		step := &AuthRegistrationStep{
			doguFetcher:             fetcher,
			authRegistrationManager: newMockAuthRegistrationManager(t),
			authRegistrationEnabled: true,
		}

		result := step.Run(testCtx, doguResource)
		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.False(t, result.Continue)
	})

	t.Run("should requeue if manager returns an error", func(t *testing.T) {
		manager := newMockAuthRegistrationManager(t)
		fetcher := newMockLocalDoguFetcher(t)
		descriptor := &cesappcore.Dogu{Name: "test"}
		fetcher.EXPECT().FetchInstalled(testCtx, doguName).Return(descriptor, nil)
		managerErr := errors.New("auth registration credentials are not ready yet")
		manager.EXPECT().EnsureAuthRegistration(testCtx, descriptor).Return(managerErr)

		step := &AuthRegistrationStep{
			doguFetcher:             fetcher,
			authRegistrationManager: manager,
			authRegistrationEnabled: true,
		}

		result := step.Run(testCtx, doguResource)
		assert.ErrorIs(t, result.Err, managerErr)
		assert.ErrorContains(t, result.Err, "auth registration credentials are not ready yet")
		assert.False(t, result.Continue)
	})

	t.Run("should continue on success", func(t *testing.T) {
		manager := newMockAuthRegistrationManager(t)
		fetcher := newMockLocalDoguFetcher(t)
		descriptor := &cesappcore.Dogu{Name: "test"}
		fetcher.EXPECT().FetchInstalled(testCtx, doguName).Return(descriptor, nil)
		manager.EXPECT().EnsureAuthRegistration(testCtx, descriptor).Return(nil)

		step := &AuthRegistrationStep{
			doguFetcher:             fetcher,
			authRegistrationManager: manager,
			authRegistrationEnabled: true,
		}

		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})

	t.Run("should continue if auth registration is disabled", func(t *testing.T) {
		step := &AuthRegistrationStep{
			doguFetcher:             newMockLocalDoguFetcher(t),
			authRegistrationManager: newMockAuthRegistrationManager(t),
			authRegistrationEnabled: false,
		}

		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
}
