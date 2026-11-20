package upgrade

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/upgrade"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testCtx = context.Background()

func testDogu(version string) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name: "nexus", Namespace: "ecosystem", Generation: 2,
		Spec: v3beta1.DoguSpec{Name: "nexus", DoguNamespace: "official", Version: version},
	}
}

func fakeClient(t *testing.T, dogu *v3beta1.Dogu) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v3beta1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogu).WithStatusSubresource(dogu).Build()
}

// storedValidCondition returns ConditionValid of the dogu as persisted in the cluster.
func storedValidCondition(t *testing.T, k8sClient client.Client, dogu *v3beta1.Dogu) *metav1.Condition {
	t.Helper()
	stored := &v3beta1.Dogu{}
	require.NoError(t, k8sClient.Get(testCtx, client.ObjectKeyFromObject(dogu), stored))
	return meta.FindStatusCondition(stored.Status.Conditions, v3beta1.ConditionValid)
}

func TestUpgradeCheckStep_Run(t *testing.T) {
	t.Run("continues and records an event if an upgrade is detected", func(t *testing.T) {
		dogu := testDogu("2.0.0")
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{Kind: upgrade.Upgrade, Installed: "1.5.0", Target: "2.0.0"}, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, ReasonUpgradeDetected, "Upgrade from version 1.5.0 to 2.0.0 detected")

		result := NewUpgradeCheckStep(detector, fakeClient(t, dogu), recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("aborts and invalidates the dogu if a downgrade is requested", func(t *testing.T) {
		dogu := testDogu("1.5.0")
		k8sClient := fakeClient(t, dogu)
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{Kind: upgrade.Downgrade, Installed: "2.0.0", Target: "1.5.0"}, nil)

		result := NewUpgradeCheckStep(detector, k8sClient, NewMockEventRecorder(t)).Run(testCtx, dogu)

		const msg = "Downgrade from version 2.0.0 to 1.5.0 is not allowed"
		assert.Equal(t, stepsv3.Abort(ReasonDowngradeNotAllowed, msg), result)

		condition := storedValidCondition(t, k8sClient, dogu)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, ReasonDowngradeNotAllowed, condition.Reason)
		assert.Equal(t, msg, condition.Message)
		assert.Equal(t, int64(2), condition.ObservedGeneration)
	})

	t.Run("continues without event if the version does not change", func(t *testing.T) {
		dogu := testDogu("2.0.0")
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{Kind: upgrade.NoChange, Installed: "2.0.0", Target: "2.0.0"}, nil)

		result := NewUpgradeCheckStep(detector, fakeClient(t, dogu), NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("requeues if the version change cannot be detected", func(t *testing.T) {
		dogu := testDogu("2.0.0")
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{}, errors.New("api unavailable"))

		result := NewUpgradeCheckStep(detector, fakeClient(t, dogu), NewMockEventRecorder(t)).Run(testCtx, dogu)

		require.Error(t, result.Err)
		assert.ErrorContains(t, result.Err, "api unavailable")
		assert.False(t, result.Continue)
		assert.Equal(t, v3beta1.ReasonUpgrading, result.ReadyReason)
	})

	t.Run("aborts and invalidates the dogu if a version cannot be parsed", func(t *testing.T) {
		dogu := testDogu("two")
		k8sClient := fakeClient(t, dogu)
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{Kind: upgrade.InvalidVersion, Installed: "1.5.0", Target: "two"}, nil)

		result := NewUpgradeCheckStep(detector, k8sClient, NewMockEventRecorder(t)).Run(testCtx, dogu)

		const msg = "Version change from 1.5.0 to two is not allowed because a version is not a valid semantic version"
		assert.Equal(t, stepsv3.Abort(ReasonInvalidVersion, msg), result)
		condition := storedValidCondition(t, k8sClient, dogu)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, ReasonInvalidVersion, condition.Reason)
	})

	t.Run("does not repeat the event for an upgrade that was already validated in this generation", func(t *testing.T) {
		dogu := testDogu("2.0.0")
		dogu.Status.Conditions = []metav1.Condition{{Type: v3beta1.ConditionValid, Status: metav1.ConditionFalse, Reason: ReasonUpgradeDescriptorUnavailable, ObservedGeneration: 2}}
		detector := NewMockUpgradeDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(upgrade.Change{Kind: upgrade.Upgrade, Installed: "1.5.0", Target: "2.0.0"}, nil)
		recoder := NewMockEventRecorder(t)

		result := NewUpgradeCheckStep(detector, fakeClient(t, dogu), recoder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Len(t, recoder.Calls, 0)
	})

	t.Run("skips the check of a paused dogu", func(t *testing.T) {
		dogu := testDogu("2.0.0")
		dogu.Spec.PauseReconciliation = true

		result := NewUpgradeCheckStep(NewMockUpgradeDetector(t), fakeClient(t, dogu), NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})
}
