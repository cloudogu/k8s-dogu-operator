package install

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewUpdatePendingStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)

	sut := NewUpdatePendingStep(clientMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
}

const (
	currentGeneration int64 = 2
	lastGeneration    int64 = 1
)

func TestUpdatePendingStep_Run(t *testing.T) {
	for _, test := range []struct {
		name           string
		doguResource   *v3beta1.Dogu
		includesStatus bool
		want           doguv3.StepResult
		assertFn       func(t *testing.T, dogu *v3beta1.Dogu)
	}{
		{
			name: "Dogu with paused reconciliation simple continues",
			doguResource: &v3beta1.Dogu{
				Name:      "dogu",
				Namespace: "namespace",
				Spec:      v3beta1.DoguSpec{PauseReconciliation: true},
			},
			want:     doguv3.Continue(),
			assertFn: checkNoUpdatePendingCondition,
		},
		{
			name: "Dogu without paused reconciliation and no UpdatePending continues",
			doguResource: &v3beta1.Dogu{
				Name:       "dogu",
				Namespace:  "namespace",
				Generation: currentGeneration,
				Status:     v3beta1.DoguStatus{},
			},
			includesStatus: true,
			want:           doguv3.Continue(),
			assertFn: func(t *testing.T, dogu *v3beta1.Dogu) {
				checkUpdatePendingCondition(t, dogu, metav1.ConditionFalse, v3beta1.ReasonReconciliationEnabled, "Reconciliation is enabled", currentGeneration)
			},
		},
		{
			name: "Dogu without paused reconciliation and UpdatePending resets it",
			doguResource: &v3beta1.Dogu{
				Name:       "dogu",
				Namespace:  "namespace",
				Generation: currentGeneration,
				Status: v3beta1.DoguStatus{
					Conditions: []metav1.Condition{
						{
							Type:   v3beta1.ConditionUpdatePending,
							Reason: ReasonReconciliationPaused,
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			includesStatus: true,
			want:           doguv3.Continue(),
			assertFn: func(t *testing.T, dogu *v3beta1.Dogu) {
				checkUpdatePendingCondition(t, dogu, metav1.ConditionFalse, v3beta1.ReasonReconciliationEnabled, "Reconciliation is enabled", currentGeneration)
			},
		},
		{
			name: "Dogu without paused reconciliation and UpdatePending: False updates condition",
			doguResource: &v3beta1.Dogu{
				Name:       "dogu",
				Namespace:  "namespace",
				Generation: currentGeneration,
				Status: v3beta1.DoguStatus{
					Conditions: []metav1.Condition{
						{
							Type:               v3beta1.ConditionUpdatePending,
							Reason:             ReasonReconciliationPaused,
							Status:             metav1.ConditionFalse,
							ObservedGeneration: lastGeneration,
						},
					},
				},
			},
			includesStatus: true,
			want:           doguv3.Continue(),
			assertFn: func(t *testing.T, dogu *v3beta1.Dogu) {
				checkUpdatePendingCondition(t, dogu, metav1.ConditionFalse, v3beta1.ReasonReconciliationEnabled, "Reconciliation is enabled", currentGeneration)
			},
		},
		{
			name: "Dogu with paused reconciliation and UpdatePending: True for an old gen only - should reset condition",
			doguResource: &v3beta1.Dogu{
				Name:       "dogu",
				Namespace:  "namespace",
				Generation: currentGeneration,
				Spec:       v3beta1.DoguSpec{PauseReconciliation: true},
				Status: v3beta1.DoguStatus{
					Conditions: []metav1.Condition{
						{
							Type:               v3beta1.ConditionUpdatePending,
							Reason:             ReasonReconciliationPaused,
							Status:             metav1.ConditionTrue,
							ObservedGeneration: lastGeneration,
						},
					},
				},
			},
			includesStatus: true,
			want:           doguv3.Continue(),
			assertFn: func(t *testing.T, dogu *v3beta1.Dogu) {
				checkUpdatePendingCondition(t, dogu, metav1.ConditionFalse, ReasonReconciliationPaused, "Reconciliation is paused", 2)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clientBuilder := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(test.doguResource)
			if test.includesStatus {
				clientBuilder = clientBuilder.WithStatusSubresource(test.doguResource)
			}
			fakeClient := clientBuilder.Build()
			sut := NewUpdatePendingStep(fakeClient)

			stepResult := sut.Run(testCtx, test.doguResource)

			assert.Equal(t, test.want, stepResult)
			if test.assertFn != nil {
				test.assertFn(t, test.doguResource)
			}
		})
	}
}

func checkNoUpdatePendingCondition(t *testing.T, dogu *v3beta1.Dogu) {
	assert.Len(t, dogu.Status.Conditions, 0)
}

func checkUpdatePendingCondition(t *testing.T, dogu *v3beta1.Dogu, status metav1.ConditionStatus, reason string, message string, generation int64) {
	assert.Len(t, dogu.Status.Conditions, 1)
	assert.Equal(t, v3beta1.ConditionUpdatePending, dogu.Status.Conditions[0].Type)
	assert.Equal(t, status, dogu.Status.Conditions[0].Status)
	assert.Equal(t, reason, dogu.Status.Conditions[0].Reason)
	assert.Equal(t, message, dogu.Status.Conditions[0].Message)
	assert.Equal(t, generation, dogu.Status.Conditions[0].ObservedGeneration)
}
