package migration

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestMigrationCleanupStep_Run(t *testing.T) {
	t.Run("removes the migration resources and requeues for the regular installation", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := newClient(t, migrationSecret(), migrationPVC())
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration resources removed`)

		result := NewMigrationCleanupStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "Migration completed"), result)
		err := k8sClient.Get(testCtx, client.ObjectKey{Namespace: "ecosystem", Name: "nexus-migration"}, &corev1.Secret{})
		assert.True(t, apierrors.IsNotFound(err))
		err = k8sClient.Get(testCtx, client.ObjectKey{Namespace: "ecosystem", Name: "nexus-migration"}, &corev1.PersistentVolumeClaim{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("cleanup error requeues with a warning", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return assert.AnError
			},
		}).Build()

		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, mock.MatchedBy(func(msg string) bool {
			return assert.Contains(t, msg, "failed to cleanup migration resources")
		}))

		result := NewMigrationCleanupStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.ErrorContains(t, result.Err, assert.AnError.Error())
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})
}
