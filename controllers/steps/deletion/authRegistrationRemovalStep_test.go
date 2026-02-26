package deletion

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
	removeErr error
}

func (arms *authRegistrationManagerStub) EnsureAuthRegistration(_ context.Context, _ *cesappcore.Dogu) error {
	return nil
}

func (arms *authRegistrationManagerStub) RemoveAuthRegistration(_ context.Context, _ cescommons.SimpleName) error {
	return arms.removeErr
}

func TestNewAuthRegistrationRemoverStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewAuthRegistrationRemoverStep(&authRegistrationManagerStub{})

		assert.NotNil(t, step)
	})
}

func TestAuthRegistrationRemoverStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue on manager error", func(t *testing.T) {
		step := &AuthRegistrationRemoverStep{
			authRegistrationManager: &authRegistrationManagerStub{removeErr: assert.AnError},
		}

		assert.Equal(t, steps.RequeueWithError(assert.AnError), step.Run(testCtx, doguResource))
	})

	t.Run("should continue on success", func(t *testing.T) {
		step := &AuthRegistrationRemoverStep{
			authRegistrationManager: &authRegistrationManagerStub{},
		}

		assert.Equal(t, steps.Continue(), step.Run(testCtx, doguResource))
	})
}
