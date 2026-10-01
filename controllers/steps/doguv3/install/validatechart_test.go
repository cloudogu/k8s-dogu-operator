package install

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/charts"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// The consumer-side interfaces must stay satisfied by the real production types.
var (
	_ ChartService   = (*charts.Service)(nil)
	_ ValueAssembler = values.Assembler{}
)

var assembledValues = values.Values{"replicaCount": 1}

func newValidationFakeClient(doguResource *v3beta1.Dogu) K8sClient {
	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(doguResource).
		WithStatusSubresource(&v3beta1.Dogu{}).
		Build()
}

func TestValidateChartStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		skip     bool
		setup    func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu)
		want     doguv3.StepResult
		assertFn func(t *testing.T, doguResource *v3beta1.Dogu)
	}{
		{
			name: "continue when schema is valid and the chart renders",
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Continue(),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionTrue, ReasonValid)
				assertCondition(t, doguResource, v3beta1.ConditionSchemaValidationSkipped, metav1.ConditionFalse, v3beta1.ReasonSchemaValidationEnabled)
			},
		},
		{
			name: "abort when the values violate the schema and do not render",
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(assert.AnError)
				// Render must not be called once the schema check fails.
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(nil)).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Abort(v3beta1.ReasonSchemaInvalid, "values do not satisfy the chart schema: "+assert.AnError.Error()),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionFalse, v3beta1.ReasonSchemaInvalid)
				assertCondition(t, doguResource, v3beta1.ConditionSchemaValidationSkipped, metav1.ConditionFalse, v3beta1.ReasonSchemaValidationEnabled)
			},
		},
		{
			name: "skip schema validation and continue when the chart renders",
			skip: true,
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.SkipSchemaValidation = true
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				// ValidateValues must not be called when schema validation is skipped.
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Continue(),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionTrue, ReasonValid)
				assertCondition(t, doguResource, v3beta1.ConditionSchemaValidationSkipped, metav1.ConditionTrue, v3beta1.ReasonSchemaValidationDisabled)
			},
		},
		{
			name: "abort with render failure even when schema validation is skipped",
			skip: true,
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.SkipSchemaValidation = true
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, assert.AnError)
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Abort(ReasonRenderFailed, "chart could not be rendered: "+assert.AnError.Error()),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionFalse, ReasonRenderFailed)
				assertCondition(t, doguResource, v3beta1.ConditionSchemaValidationSkipped, metav1.ConditionTrue, v3beta1.ReasonSchemaValidationDisabled)
			},
		},
		{
			name: "abort with render failure after a valid schema",
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, assert.AnError)
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Abort(ReasonRenderFailed, "chart could not be rendered: "+assert.AnError.Error()),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionFalse, ReasonRenderFailed)
			},
		},
		{
			name: "requeue when the dogu meta values cannot be read",
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, assert.AnError)
				// Neither assembly nor validation nor render must run.
				as := NewMockAssembler(t)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to get dogu meta values: %w", assert.AnError), v3beta1.ReasonInstalling),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
			},
		},
		{
			name: "requeue when the values cannot be assembled",
			setup: func(t *testing.T) (*MockChartService, *MockAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				as := NewMockAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(nil, assert.AnError)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to assemble values: %w", assert.AnError), v3beta1.ReasonInstalling),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, as, k8sClient, doguResource := tt.setup(t)
			step := NewValidateChartStep(cs, as, k8sClient)

			got := step.Run(testCtx, doguResource)

			assert.Equal(t, tt.want, got)
			tt.assertFn(t, doguResource)
		})
	}
}

func TestValidateChartStep_Run_requeuesWhenStatusUpdateFails(t *testing.T) {
	doguResource := testDoguResource.DeepCopy()
	cs := NewMockChartService(t)
	cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
	cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
	cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
	as := NewMockAssembler(t)
	as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)

	k8sClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(doguResource).
		WithStatusSubresource(&v3beta1.Dogu{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(_ context.Context, _ client.Client, _ string, _ client.Object, _ ...client.SubResourceUpdateOption) error {
				return conflictErr
			},
		}).
		Build()

	step := NewValidateChartStep(cs, as, k8sClient)

	got := step.Run(testCtx, doguResource)

	require.Error(t, got.Err)
	assert.ErrorContains(t, got.Err, "failed to update dogu validation conditions")
	assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
	assert.False(t, got.Continue)
}

func assertCondition(t *testing.T, doguResource *v3beta1.Dogu, condType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	cond := apimeta.FindStatusCondition(doguResource.Status.Conditions, condType)
	require.NotNil(t, cond, "condition %q must be set", condType)
	assert.Equal(t, status, cond.Status, "condition %q status", condType)
	assert.Equal(t, reason, cond.Reason, "condition %q reason", condType)
}
