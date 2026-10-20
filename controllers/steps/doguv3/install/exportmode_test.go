package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExportModeStep_Run(t *testing.T) {
	testDoguWithExportMode := &v3beta1.Dogu{Name: testDoguName, Namespace: testNamespace, Spec: v3beta1.DoguSpec{ExportMode: true}}
	testDoguWithoutExportMode := &v3beta1.Dogu{Name: testDoguName, Namespace: testNamespace, Spec: v3beta1.DoguSpec{}}
	testDoguWithExportModeAndCondition := &v3beta1.Dogu{
		Name:       testDoguName,
		Namespace:  testNamespace,
		Generation: 23,
		Spec: v3beta1.DoguSpec{
			ExportMode: true,
		},
		Status: v3beta1.DoguStatus{
			Conditions: []metav1.Condition{
				{
					Type:               v3beta1.ConditionExportModeActive,
					Status:             metav1.ConditionFalse,
					Reason:             conditionExportModeReason,
					Message:            conditionOrEventExportModeMessage,
					ObservedGeneration: 23,
				},
			},
		},
	}
	testDoguWithExportModeAndInvalidCondition := &v3beta1.Dogu{Name: testDoguName,
		Namespace: testNamespace,
		Spec: v3beta1.DoguSpec{
			ExportMode: true,
		},
		Status: v3beta1.DoguStatus{
			Conditions: []metav1.Condition{
				{
					Type:   v3beta1.ConditionExportModeActive,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	type fields struct {
		k8sClient     func(t *testing.T) K8sClient
		eventRecorder func(t *testing.T, doguResource *v3beta1.Dogu) EventRecorder
	}
	type args struct {
		ctx          context.Context
		doguResource *v3beta1.Dogu
	}
	assertValidWarningConditionFn := func(t *testing.T, k8sClient K8sClient) {
		dogu := &v3beta1.Dogu{}
		err := k8sClient.Get(testCtx, testNamespacedName, dogu)
		require.NoError(t, err)
		condition := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionExportModeActive)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditionExportModeReason, condition.Reason)
		assert.Equal(t, conditionOrEventExportModeMessage, condition.Message)
		assert.Equal(t, dogu.Generation, condition.ObservedGeneration)
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantFn   func(t *testing.T, step doguv3.StepResult)
		assertFn func(t *testing.T, k8sClient K8sClient)
	}{
		{
			name: "should set a informative ExportModeActiveCondition if exportMode is true",
			fields: fields{
				k8sClient: func(t *testing.T) K8sClient {
					return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(testDoguWithExportMode).WithStatusSubresource(testDoguWithExportMode).Build()
				},
				eventRecorder: func(t *testing.T, doguResource *v3beta1.Dogu) EventRecorder {
					eventRecorderMock := NewMockEventRecorder(t)
					eventRecorderMock.EXPECT().Event(doguResource, v1.EventTypeWarning, "ExportModeActive", "Export mode was activated but is currently not supported for V3 Dogus")
					return eventRecorderMock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: testDoguWithExportMode,
			},
			wantFn: func(t *testing.T, step doguv3.StepResult) {
				assert.Equal(t, testStepResultContinue, step)
			},
			assertFn: assertValidWarningConditionFn,
		},
		{
			name: "should continue and do nothing if export mode is false",
			args: args{
				ctx:          testCtx,
				doguResource: testDoguWithoutExportMode,
			},
			wantFn: func(t *testing.T, step doguv3.StepResult) {
				assert.Equal(t, testStepResultContinue, step)
			},
		},
		{
			name: "should continue if export mode is active but condition is already present",
			fields: fields{
				k8sClient: func(t *testing.T) K8sClient {
					return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(testDoguWithExportModeAndCondition).WithStatusSubresource(testDoguWithExportModeAndCondition).Build()
				},
				eventRecorder: func(t *testing.T, doguResource *v3beta1.Dogu) EventRecorder {
					eventRecorderMock := NewMockEventRecorder(t)
					return eventRecorderMock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: testDoguWithExportModeAndCondition,
			},
			wantFn: func(t *testing.T, step doguv3.StepResult) {
				assert.Equal(t, testStepResultContinue, step)
			},
		},
		{
			name: "should update present condition if is not equal like the NotImplemented condition",
			fields: fields{
				k8sClient: func(t *testing.T) K8sClient {
					return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(testDoguWithExportModeAndInvalidCondition).WithStatusSubresource(testDoguWithExportModeAndInvalidCondition).Build()
				},
				eventRecorder: func(t *testing.T, doguResource *v3beta1.Dogu) EventRecorder {
					eventRecorderMock := NewMockEventRecorder(t)
					return eventRecorderMock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: testDoguWithExportModeAndInvalidCondition,
			},
			wantFn: func(t *testing.T, step doguv3.StepResult) {
				assert.Equal(t, testStepResultContinue, step)
			},
			assertFn: assertValidWarningConditionFn,
		},
		{
			name: "should return error on error updating conditions",
			fields: fields{
				k8sClient: func(t *testing.T) K8sClient {
					return fake.NewClientBuilder().Build()
				},
				eventRecorder: func(t *testing.T, doguResource *v3beta1.Dogu) EventRecorder {
					eventRecorderMock := NewMockEventRecorder(t)
					eventRecorderMock.EXPECT().Event(doguResource, v1.EventTypeWarning, "ExportModeActive", "Export mode was activated but is currently not supported for V3 Dogus")
					return eventRecorderMock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: &v3beta1.Dogu{Spec: v3beta1.DoguSpec{ExportMode: true}},
			},
			wantFn: func(t *testing.T, step doguv3.StepResult) {
				assert.ErrorContains(t, step.Err, "failed to apply condition to status")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ems := &ExportModeStep{}
			if tt.fields.k8sClient != nil {
				ems.k8sClient = tt.fields.k8sClient(t)
			}
			if tt.fields.eventRecorder != nil {
				ems.eventRecorder = tt.fields.eventRecorder(t, tt.args.doguResource)
			}

			result := ems.Run(tt.args.ctx, tt.args.doguResource)
			if tt.wantFn != nil {
				tt.wantFn(t, result)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, ems.k8sClient)
			}
		})
	}
}

func TestNewExportModeStep(t *testing.T) {
	// given
	k8sClientMock := fake.NewClientBuilder().Build()
	eventRecorderMock := NewMockEventRecorder(t)

	// when
	step := NewExportModeStep(k8sClientMock, eventRecorderMock)

	// then
	require.NotNil(t, step)
	assert.Equal(t, k8sClientMock, step.k8sClient)
	assert.Equal(t, eventRecorderMock, step.eventRecorder)
}
