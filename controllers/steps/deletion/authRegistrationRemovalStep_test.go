package deletion

import (
	"context"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewAuthRegistrationRemoverStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		authRegistrationManagerMock := newMockAuthRegistrationManager(t)
		step := NewAuthRegistrationRemoverStep(authRegistrationManagerMock)

		assert.NotNil(t, step)
	})
}

func TestAuthRegistrationRemoverStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue on manager error", func(t *testing.T) {
		authRegistrationManagerMock := newMockAuthRegistrationManager(t)
		authRegistrationManagerMock.EXPECT().RemoveAuthRegistration(testCtx, cescommons.SimpleName("test")).Return(assert.AnError)

		step := &AuthRegistrationRemoverStep{
			authRegistrationManager: authRegistrationManagerMock,
		}

		assert.Equal(t, steps.RequeueWithError(assert.AnError), step.Run(testCtx, doguResource))
	})

	t.Run("should continue on success", func(t *testing.T) {
		authRegistrationManagerMock := newMockAuthRegistrationManager(t)
		authRegistrationManagerMock.EXPECT().RemoveAuthRegistration(testCtx, cescommons.SimpleName("test")).Return(nil)

		step := &AuthRegistrationRemoverStep{
			authRegistrationManager: authRegistrationManagerMock,
		}

		assert.Equal(t, steps.Continue(), step.Run(testCtx, doguResource))
	})
}
