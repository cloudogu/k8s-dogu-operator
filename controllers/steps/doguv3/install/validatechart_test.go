package install

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/charts"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	validationValuesMetaContent   = "meta"
	validationRuntimePatchContent = "runtime"
)

// The consumer-side interfaces must stay satisfied by the real production types.
var (
	_ chartService   = (*charts.Service)(nil)
	_ valueAssembler = values.Assembler{}
	_ ResizeChecker  = pvc.NewResizeChecker(nil)
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
		name      string
		skip      bool
		wantCheck bool
		setup     func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu)
		want      doguv3.StepResult
		assertFn  func(t *testing.T, doguResource *v3beta1.Dogu)
	}{
		{
			name:      "continue when schema is valid and the chart renders",
			wantCheck: true,
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return([]byte(validationRuntimePatchContent), true, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(validationRuntimePatchContent)).Return(assembledValues, nil)
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
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(assert.AnError)
				// Render must not be called once the schema check fails.
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(nil), []byte(nil)).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Abort(v3beta1.ReasonSchemaInvalid, "values do not satisfy the chart schema: "+assert.AnError.Error()),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionFalse, v3beta1.ReasonSchemaInvalid)
				assertCondition(t, doguResource, v3beta1.ConditionSchemaValidationSkipped, metav1.ConditionFalse, v3beta1.ReasonSchemaValidationEnabled)
			},
		},
		{
			name:      "skip schema validation and continue when the chart renders",
			skip:      true,
			wantCheck: true,
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.SkipSchemaValidation = true
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
				// ValidateValues must not be called when schema validation is skipped.
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(nil)).Return(assembledValues, nil)
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
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.SkipSchemaValidation = true
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, assert.AnError)
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(nil)).Return(assembledValues, nil)
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
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
				cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
				cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, assert.AnError)
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(nil)).Return(assembledValues, nil)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.Abort(ReasonRenderFailed, "chart could not be rendered: "+assert.AnError.Error()),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assertCondition(t, doguResource, v3beta1.ConditionValid, metav1.ConditionFalse, ReasonRenderFailed)
			},
		},
		{
			name: "requeue when the dogu meta values cannot be read",
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, assert.AnError)
				// Neither assembly nor validation nor render must run.
				as := newMockValueAssembler(t)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to get dogu meta values for dogu %s/%s: %w", testDoguResource.Namespace, testDoguResource.Name, assert.AnError), v3beta1.ReasonInstalling),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
			},
		},
		{
			name: "requeue when the values cannot be assembled",
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := newMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
				cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
				as := newMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(nil)).Return(nil, assert.AnError)
				return cs, as, newValidationFakeClient(doguResource), doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to assemble values for dogu %s/%s: %w", testDoguResource.Namespace, testDoguResource.Name, assert.AnError), v3beta1.ReasonInstalling),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
			},
		},
		{
			name: "continue without checking if PauseReconciliation is set",
			setup: func(t *testing.T) (*mockChartService, *mockValueAssembler, K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.PauseReconciliation = true
				return nil, nil, nil, doguResource
			},
			want: doguv3.Continue(),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, as, k8sClient, doguResource := tt.setup(t)
			rc := NewMockResizeChecker(t)
			if tt.wantCheck {
				rc.EXPECT().Check(mock.Anything, testNamespace, []*unstructured.Unstructured(nil)).Return(pvc.CheckResult{}, nil)
			}
			step := NewValidateChartStep(cs, as, rc, k8sClient)

			got := step.Run(testCtx, doguResource)

			assert.Equal(t, tt.want, got)
			tt.assertFn(t, doguResource)
		})
	}
}

func TestValidateChartStep_Run_rejectsEmptyRuntimePatch(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source []byte
	}{
		{name: "nil"},
		{name: "empty", source: []byte{}},
		{name: "whitespace", source: []byte("\t\n    ")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dogu := testDoguResource.DeepCopy()
			service := newMockChartService(t)
			service.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil)
			service.EXPECT().RuntimePatchTemplate(mock.Anything, dogu).Return(tt.source, true, nil)
			assembler := newMockValueAssembler(t)
			step := NewValidateChartStep(service, assembler, newValidationFakeClient(dogu))

			result := step.Run(t.Context(), dogu)

			require.ErrorContains(t, result.Err, "runtime-patch-tpl.yaml must not be empty")
			assert.ErrorContains(t, result.Err, dogu.Namespace+"/"+dogu.Name)
			assert.False(t, result.Continue)
			assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		})
	}
}

func TestValidateChartStep_Run_requeuesWhenStatusUpdateFails(t *testing.T) {
	doguResource := testDoguResource.DeepCopy()
	cs := newMockChartService(t)
	cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte(validationValuesMetaContent), true, nil)
	cs.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil)
	cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
	cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
	as := newMockValueAssembler(t)
	as.EXPECT().Assemble(mock.Anything, doguResource, []byte(validationValuesMetaContent), []byte(nil)).Return(assembledValues, nil)

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
	rc := NewMockResizeChecker(t)
	rc.EXPECT().Check(mock.Anything, testNamespace, []*unstructured.Unstructured(nil)).Return(pvc.CheckResult{}, nil)

	step := NewValidateChartStep(cs, as, rc, k8sClient)

	got := step.Run(testCtx, doguResource)

	require.Error(t, got.Err)
	assert.ErrorContains(t, got.Err, "failed to update dogu validation conditions")
	assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
	assert.False(t, got.Continue)
}

func TestValidateChartStep_Run_volumeCheck(t *testing.T) {
	shrinkErr := &pvc.VolumeShrinkError{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}, Current: resource.MustParse("2Gi"), Desired: resource.MustParse("1Gi")}
	storageClassErr := &pvc.StorageClassImmutableError{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}, Current: ptr.To("standard"), Desired: ptr.To("fast")}

	tests := []struct {
		name      string
		checkErr  error
		want      doguv3.StepResult
		wantValid metav1.ConditionStatus
		reason    string
	}{
		{
			name:      "continue when no volume is shrunk or changes its storage class",
			want:      doguv3.Continue(),
			wantValid: metav1.ConditionTrue,
			reason:    ReasonValid,
		},
		{
			name:      "abort when a volume would shrink",
			checkErr:  shrinkErr,
			want:      doguv3.Abort(v3beta1.ReasonVolumeShrink, `volumes cannot be changed as requested: cannot shrink PVC "ecosystem/data" from 2Gi to 1Gi`),
			wantValid: metav1.ConditionFalse,
			reason:    v3beta1.ReasonVolumeShrink,
		},
		{
			name:      "abort when a volume would change its storage class",
			checkErr:  storageClassErr,
			want:      doguv3.Abort(v3beta1.ReasonStorageClassImmutable, `volumes cannot be changed as requested: cannot change storage class of PVC "ecosystem/data" from "standard" to "fast"`),
			wantValid: metav1.ConditionFalse,
			reason:    v3beta1.ReasonStorageClassImmutable,
		},
		{
			name:      "abort with shrink when additionally another volume is not ready",
			checkErr:  errors.Join(assert.AnError, shrinkErr),
			want:      doguv3.Abort(v3beta1.ReasonVolumeShrink, "volumes cannot be changed as requested: "+errors.Join(assert.AnError, shrinkErr).Error()),
			wantValid: metav1.ConditionFalse,
			reason:    v3beta1.ReasonVolumeShrink,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			rendered := renderedChart(t)
			cs := NewMockChartService(t)
			cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
			cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
			cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(rendered, nil)
			as := NewMockValueAssembler(t)
			as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
			rc := NewMockResizeChecker(t)
			rc.EXPECT().Check(mock.Anything, testNamespace, rendered).Return(pvc.CheckResult{}, tt.checkErr)
			step := NewValidateChartStep(cs, as, rc, newValidationFakeClient(doguResource))

			got := step.Run(testCtx, doguResource)

			assert.Equal(t, tt.want, got)
			assertCondition(t, doguResource, v3beta1.ConditionValid, tt.wantValid, tt.reason)
		})
	}
}

func TestValidateChartStep_Run_requeuesWhenVolumeCheckFails(t *testing.T) {
	doguResource := testDoguResource.DeepCopy()
	cs := NewMockChartService(t)
	cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
	cs.EXPECT().ValidateValues(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil)
	cs.EXPECT().Render(mock.Anything, doguResource, map[string]any(assembledValues)).Return(nil, nil)
	as := NewMockValueAssembler(t)
	as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
	rc := NewMockResizeChecker(t)
	rc.EXPECT().Check(mock.Anything, testNamespace, []*unstructured.Unstructured(nil)).Return(pvc.CheckResult{}, assert.AnError)
	step := NewValidateChartStep(cs, as, rc, newValidationFakeClient(doguResource))

	got := step.Run(testCtx, doguResource)

	assert.Equal(t, doguv3.RequeueWithError(fmt.Errorf("failed to check volumes: %w", assert.AnError), v3beta1.ReasonInstalling), got)
	assert.Nil(t, apimeta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid))
}

func assertCondition(t *testing.T, doguResource *v3beta1.Dogu, condType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	cond := apimeta.FindStatusCondition(doguResource.Status.Conditions, condType)
	require.NotNil(t, cond, "condition %q must be set", condType)
	assert.Equal(t, status, cond.Status, "condition %q status", condType)
	assert.Equal(t, reason, cond.Reason, "condition %q reason", condType)
}
