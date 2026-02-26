package install

import (
	"context"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type authRegistrationManagerStub struct {
	ensureErr error
}

func (arms *authRegistrationManagerStub) EnsureAuthRegistration(_ context.Context, _ *cesappcore.Dogu) error {
	return arms.ensureErr
}

func (arms *authRegistrationManagerStub) RemoveAuthRegistration(_ context.Context, _ cescommons.SimpleName) error {
	return nil
}

func TestNewAuthRegistrationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewAuthRegistrationStep(&authRegistrationManagerStub{}, newMockLocalDoguFetcher(t))

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
			localDoguFetcher:        fetcher,
			authRegistrationManager: &authRegistrationManagerStub{},
		}

		assert.Equal(t, steps.RequeueWithError(assert.AnError), step.Run(testCtx, doguResource))
	})

	t.Run("should requeue if manager returns an error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchInstalled(testCtx, doguName).Return(&cesappcore.Dogu{Name: "test"}, nil)

		step := &AuthRegistrationStep{
			localDoguFetcher:        fetcher,
			authRegistrationManager: &authRegistrationManagerStub{ensureErr: assert.AnError},
		}

		assert.Equal(t, steps.RequeueWithError(assert.AnError), step.Run(testCtx, doguResource))
	})

	t.Run("should continue on success", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		fetcher.EXPECT().FetchInstalled(testCtx, doguName).Return(&cesappcore.Dogu{Name: "test"}, nil)

		step := &AuthRegistrationStep{
			localDoguFetcher:        fetcher,
			authRegistrationManager: &authRegistrationManagerStub{},
		}

		assert.Equal(t, steps.Continue(), step.Run(testCtx, doguResource))
	})
}
