package upgrade

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
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

var detectedUpgrade = upgrade.Change{Kind: upgrade.Upgrade, Installed: "1.5.0", Target: "2.0.0"}

func testDogu() *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name: "nexus", Namespace: "ecosystem", Generation: 2,
		Spec: v3beta1.DoguSpec{Name: "nexus", DoguNamespace: "official", Version: "2.0.0"},
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

func assertValidCondition(t *testing.T, condition *metav1.Condition, status metav1.ConditionStatus, reason, message string) {
	t.Helper()
	require.NotNil(t, condition)
	assert.Equal(t, status, condition.Status)
	assert.Equal(t, reason, condition.Reason)
	assert.Equal(t, message, condition.Message)
	assert.Equal(t, int64(2), condition.ObservedGeneration)
}

func validatorReturning(t *testing.T, dogu *v3beta1.Dogu, devChart bool, result upgrade.Result, err error) *MockUpgradeValidator {
	validator := NewMockUpgradeValidator(t)
	validator.EXPECT().Validate(testCtx, dogu, devChart).Return(result, err)
	return validator
}

// upgradeDetectedRecorder expects the event of the detected upgrade.
func upgradeDetectedRecorder(t *testing.T, dogu *v3beta1.Dogu) *MockEventRecorder {
	recorder := NewMockEventRecorder(t)
	recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, ReasonUpgradeDetected, "Upgrade from version 1.5.0 to 2.0.0 detected")
	return recorder
}

func TestValidateUpgradeStep_Run(t *testing.T) {
	t.Run("valid upgrade continues", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fakeClient(t, dogu)
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Valid, Message: "Upgrade from version 1.5.0 to 2.0.0 is valid"}

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, result, nil), k8sClient, upgradeDetectedRecorder(t, dogu), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
		assertValidCondition(t, storedValidCondition(t, k8sClient, dogu), metav1.ConditionTrue, ReasonUpgradeValid, result.Message)
	})

	t.Run("skipped upgrade of a dev chart continues", func(t *testing.T) {
		dogu := testDogu()
		dogu.Annotations = map[string]string{install.AnnotationDevChartURL: "oci://localhost/nexus"}
		k8sClient := fakeClient(t, dogu)
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Skipped, Message: "Upgrade from version 1.5.0 to 2.0.0 is not validated because the dogu uses a dev chart"}
		recorder := upgradeDetectedRecorder(t, dogu)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, ReasonUpgradeValidationSkipped, result.Message)

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, true, result, nil), k8sClient, recorder, true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
		assertValidCondition(t, storedValidCondition(t, k8sClient, dogu), metav1.ConditionTrue, ReasonUpgradeValidationSkipped, result.Message)
	})

	t.Run("dev chart annotation is ignored if dev charts are disabled", func(t *testing.T) {
		dogu := testDogu()
		dogu.Annotations = map[string]string{install.AnnotationDevChartURL: "oci://localhost/nexus"}
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Valid, Message: "valid"}

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, result, nil), fakeClient(t, dogu), upgradeDetectedRecorder(t, dogu), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})

	t.Run("refused Upgrades", func(t *testing.T) {

	})

	refusedTests := []struct {
		name       string
		change     upgrade.Change
		cause      upgrade.Cause
		wantReason string
	}{
		{name: "downgrade", change: upgrade.Change{Kind: upgrade.Downgrade, Installed: "2.0.0", Target: "1.5.0"}, cause: upgrade.CauseDowngrade, wantReason: ReasonDowngradeNotAllowed},
		{name: "invalid version", change: upgrade.Change{Kind: upgrade.InvalidVersion, Installed: "1.5.0", Target: "two"}, cause: upgrade.CauseInvalidVersion, wantReason: ReasonInvalidVersion},
		{name: "missing upgrade path", change: detectedUpgrade, cause: upgrade.CauseInvalidUpgradePath, wantReason: v3beta1.ReasonNoUpgradePath},
		{name: "invalid descriptor", change: detectedUpgrade, cause: upgrade.CauseInvalidDescriptor, wantReason: ReasonInvalidUpgradeDescriptor},
		{name: "unavailable descriptor", change: detectedUpgrade, cause: upgrade.CauseDescriptorUnavailable, wantReason: ReasonUpgradeDescriptorUnavailable},
	}
	for _, tt := range refusedTests {
		t.Run("refuses the version change because of "+tt.name, func(t *testing.T) {
			dogu := testDogu()
			k8sClient := fakeClient(t, dogu)
			result := upgrade.Result{Change: tt.change, Verdict: upgrade.Refused, Cause: tt.cause, Message: "refused because of " + tt.name}
			recorder := NewMockEventRecorder(t)
			if tt.change.Kind == upgrade.Upgrade {
				recorder = upgradeDetectedRecorder(t, dogu)
			}

			got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, result, nil), k8sClient, recorder, false).Run(testCtx, dogu)

			assert.Equal(t, stepsv3.Abort(tt.wantReason, result.Message), got)
			assertValidCondition(t, storedValidCondition(t, k8sClient, dogu), metav1.ConditionFalse, tt.wantReason, result.Message)
		})
	}

	t.Run("records the event once the descriptor becomes available after a temporary failure", func(t *testing.T) {
		dogu := testDogu()
		dogu.Status.Conditions = []metav1.Condition{{Type: v3beta1.ConditionValid, Status: metav1.ConditionFalse, Reason: ReasonUpgradeDescriptorUnavailable, Message: "connection refused", ObservedGeneration: 2}}
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Valid, Message: "valid"}

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, result, nil), fakeClient(t, dogu), upgradeDetectedRecorder(t, dogu), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})

	t.Run("does not repeat the events of a skipped upgrade with unchanged condition", func(t *testing.T) {
		dogu := testDogu()
		dogu.Annotations = map[string]string{install.AnnotationDevChartURL: "oci://localhost/nexus"}
		dogu.Status.Conditions = []metav1.Condition{{Type: v3beta1.ConditionValid, Status: metav1.ConditionTrue, Reason: ReasonUpgradeValidationSkipped, Message: "skipped", ObservedGeneration: 2}}
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Skipped, Message: "skipped"}

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, true, result, nil), fakeClient(t, dogu), NewMockEventRecorder(t), true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})

	t.Run("does not repeat the event of an upgrade that was already validated in this generation", func(t *testing.T) {
		dogu := testDogu()
		dogu.Status.Conditions = []metav1.Condition{{Type: v3beta1.ConditionValid, Status: metav1.ConditionTrue, Reason: ReasonUpgradeValid, Message: "valid", ObservedGeneration: 2}}
		result := upgrade.Result{Change: detectedUpgrade, Verdict: upgrade.Valid, Message: "valid"}

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, result, nil), fakeClient(t, dogu), NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})

	t.Run("requeues and invalidates the dogu if the descriptor is temporarily unavailable", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fakeClient(t, dogu)
		err := errors.Join(upgrade.ErrDescriptorUnavailable, errors.New("connection refused"))

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, upgrade.Result{}, err), k8sClient, NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.ErrorIs(t, got.Err, err)
		assert.Equal(t, v3beta1.ReasonUpgrading, got.ReadyReason)
		assertValidCondition(t, storedValidCondition(t, k8sClient, dogu), metav1.ConditionFalse, ReasonUpgradeDescriptorUnavailable, err.Error())
	})

	t.Run("requeues without condition if the validation fails otherwise", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fakeClient(t, dogu)

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, upgrade.Result{}, assert.AnError), k8sClient, NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.ErrorIs(t, got.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonUpgrading, got.ReadyReason)
		assert.Nil(t, storedValidCondition(t, k8sClient, dogu))
	})

	t.Run("continues if the version is unchanged", func(t *testing.T) {
		dogu := testDogu()

		got := NewValidateUpgradeStep(validatorReturning(t, dogu, false, upgrade.Result{Verdict: upgrade.Unchanged}, nil), fakeClient(t, dogu), NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})

	t.Run("skips the validation of a paused dogu", func(t *testing.T) {
		dogu := testDogu()
		dogu.Spec.PauseReconciliation = true

		got := NewValidateUpgradeStep(NewMockUpgradeValidator(t), fakeClient(t, dogu), NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), got)
	})
}
