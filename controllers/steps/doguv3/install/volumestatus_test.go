package install

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var testWorkloads = workloads.Workloads{StatefulSets: []appsv1.StatefulSet{{Name: "nexus"}}}

func volumePVC(name, request, capacity string, conditions ...corev1.PersistentVolumeClaimConditionType) corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{
		Name: name, Namespace: testNamespace,
		Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(request)},
		}},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase:    corev1.ClaimBound,
			Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(capacity)},
		},
	}
	for _, condition := range conditions {
		pvc.Status.Conditions = append(pvc.Status.Conditions, corev1.PersistentVolumeClaimCondition{Type: condition, Status: corev1.ConditionTrue})
	}
	return pvc
}

func mountedBy(kind, name string) []corev1.TypedLocalObjectReference {
	return []corev1.TypedLocalObjectReference{{APIGroup: ptr.To(appsv1.GroupName), Kind: kind, Name: name}}
}

// toUnstructured rendered charts are of type Unstructured and this helper can be used to create those
func toUnstructured(t *testing.T, objects ...client.Object) []*unstructured.Unstructured {
	t.Helper()
	result := make([]*unstructured.Unstructured, 0, len(objects))
	for _, object := range objects {
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		require.NoError(t, err)
		result = append(result, &unstructured.Unstructured{Object: content})
	}
	return result
}

func renderedChart(t *testing.T) []*unstructured.Unstructured {
	t.Helper()
	pvc := volumePVC("data", "2Gi", "2Gi")
	pvc.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
	template := volumePVC("db", "5Gi", "5Gi")

	return toUnstructured(t,
		&pvc,
		&appsv1.StatefulSet{
			APIVersion: "apps/v1", Kind: "StatefulSet",
			Name: "nexus", Namespace: testNamespace,
			Spec: appsv1.StatefulSetSpec{VolumeClaimTemplates: []corev1.PersistentVolumeClaim{template}},
		},
	)
}

func newVolumeStatusFakeClient(doguResource *v3beta1.Dogu, funcs interceptor.Funcs) K8sClient {
	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(doguResource).
		WithStatusSubresource(&v3beta1.Dogu{}).
		WithInterceptorFuncs(funcs).
		Build()
}

func mockChartRendering(t *testing.T, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, renderErr error) (*MockChartService, *MockValueAssembler) {
	cs := NewMockChartService(t)
	cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
	cs.EXPECT().Render(mock.Anything, doguResource, assembledValues).Return(rendered, renderErr)
	as := NewMockValueAssembler(t)
	as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil)
	return cs, as
}

func TestVolumeStatusStep_Run(t *testing.T) {
	volumes := []workloads.Volume{
		{PVC: volumePVC("existing", "1Gi", "1Gi"), MountedBy: mountedBy("Deployment", "nexus-proxy")},
		{PVC: volumePVC("db-nexus-0", "5Gi", "3Gi"), MountedBy: mountedBy("StatefulSet", "nexus")},
		{PVC: volumePVC("data", "2Gi", "2Gi"), MountedBy: mountedBy("StatefulSet", "nexus")},
		{PVC: volumePVC("db-nexus-1", "5Gi", "5Gi", corev1.PersistentVolumeClaimFileSystemResizePending), MountedBy: mountedBy("StatefulSet", "nexus")},
		{PVC: volumePVC("db-nexus-2", "5Gi", "3Gi", corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimControllerResizeError), MountedBy: mountedBy("StatefulSet", "nexus")},
		{PVC: volumePVC("db-nexus-3", "5Gi", "3Gi", corev1.PersistentVolumeClaimNodeResizeError), MountedBy: mountedBy("StatefulSet", "nexus")},
	}
	happyCaseStatus := []v3beta1.VolumeStatus{
		{ClaimName: "data", Size: resource.MustParse("2Gi"), DesiredSize: resource.MustParse("2Gi"), State: v3beta1.VolumeStatusStateReady, MountedBy: mountedBy("StatefulSet", "nexus")},
		{ClaimName: "db-nexus-0", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizing, MountedBy: mountedBy("StatefulSet", "nexus")},
		{ClaimName: "db-nexus-1", Size: resource.MustParse("5Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizing, MountedBy: mountedBy("StatefulSet", "nexus")},
		{ClaimName: "db-nexus-2", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizeFailed, MountedBy: mountedBy("StatefulSet", "nexus")},
		{ClaimName: "db-nexus-3", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizeFailed, MountedBy: mountedBy("StatefulSet", "nexus")},
		{ClaimName: "existing", Size: resource.MustParse("1Gi"), State: v3beta1.VolumeStatusStateReady, MountedBy: mountedBy("Deployment", "nexus-proxy")},
	}
	happyCaseResult := doguv3.RequeueWithError(errors.New("volumes failed to resize: [db-nexus-2 db-nexus-3]"), v3beta1.ReasonResizingPVC)

	tests := []struct {
		name       string
		setup      func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu)
		want       doguv3.StepResult
		wantErr    string
		wantStatus []v3beta1.VolumeStatus
	}{
		{
			name: "write size, desired size, state and mounting workloads of each volume sorted by claim name",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(volumes, nil)
				return NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			want:       happyCaseResult,
			wantStatus: happyCaseStatus,
		},
		{
			name: "remove volumes that are no longer mounted",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Status.VolumeStatus = happyCaseStatus
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(nil, nil)
				return NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			want:       doguv3.Continue(),
			wantStatus: nil,
		},
		{
			name: "skip the status update when the volume status is unchanged",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Status.VolumeStatus = happyCaseStatus
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(volumes, nil)

				// throw error on update function, so that we notice if the update is not skipped
				k8sClient := newVolumeStatusFakeClient(doguResource, interceptor.Funcs{
					SubResourceUpdate: func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
						return assert.AnError
					},
				})
				return NewVolumeStatusStep(cs, as, d, k8sClient), doguResource
			},
			want:       happyCaseResult,
			wantStatus: happyCaseStatus,
		},
		{
			name: "requeue when the dogu meta values cannot be read",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, assert.AnError)
				return NewVolumeStatusStep(cs, NewMockValueAssembler(t), NewMockVolumeDiscovery(t), newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to get dogu meta values",
		},
		{
			name: "requeue when the values cannot be assembled",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs := NewMockChartService(t)
				cs.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil)
				as := NewMockValueAssembler(t)
				as.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(nil, assert.AnError)
				return NewVolumeStatusStep(cs, as, NewMockVolumeDiscovery(t), newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to assemble values",
		},
		{
			name: "requeue when the chart cannot be rendered",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs, as := mockChartRendering(t, doguResource, nil, assert.AnError)
				return NewVolumeStatusStep(cs, as, NewMockVolumeDiscovery(t), newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to render chart",
		},
		{
			name: "requeue when the workloads cannot be discovered",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, assert.AnError)
				return NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to discover workloads",
		},
		{
			name: "requeue when the volumes cannot be discovered",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(nil, assert.AnError)
				return NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to discover volumes",
		},
		{
			name: "requeue when the desired sizes cannot be read from the rendered chart",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				pvcWithoutRequest := volumePVC("data", "2Gi", "2Gi")
				pvcWithoutRequest.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
				pvcWithoutRequest.Spec.Resources.Requests = nil
				cs, as := mockChartRendering(t, doguResource, toUnstructured(t, &pvcWithoutRequest), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(volumes, nil)
				return NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})), doguResource
			},
			wantErr: "failed to read desired volume sizes",
		},
		{
			name: "requeue when the status cannot be updated",
			setup: func(t *testing.T) (*VolumeStatusStep, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
				d := NewMockVolumeDiscovery(t)
				d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
				d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(volumes, nil)
				k8sClient := newVolumeStatusFakeClient(doguResource, interceptor.Funcs{
					SubResourceUpdate: func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
						return assert.AnError
					},
				})
				return NewVolumeStatusStep(cs, as, d, k8sClient), doguResource
			},
			wantErr: "failed to update volume status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, doguResource := tt.setup(t)

			got := step.Run(testCtx, doguResource)

			if tt.wantErr != "" {
				assert.ErrorContains(t, got.Err, tt.wantErr)
				assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
				assert.False(t, got.Continue)
				return
			}
			assert.Equal(t, tt.want, got)
			persisted := &v3beta1.Dogu{}
			require.NoError(t, step.k8sClient.Get(testCtx, testNamespacedName, persisted))
			assert.Equal(t, tt.wantStatus, persisted.Status.VolumeStatus)
		})
	}
}

func TestVolumeStatusStep_RunWaitsForResize(t *testing.T) {
	statefulSet := mountedBy("StatefulSet", "nexus")

	t.Run("requeue with ResizingPVC while a rendered volume has not reached its desired capacity", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		volumes := []workloads.Volume{{PVC: volumePVC("data", "2Gi", "2Gi"), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-0", "5Gi", "3Gi"), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-1", "5Gi", "5Gi", corev1.PersistentVolumeClaimFileSystemResizePending), MountedBy: statefulSet}}

		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
		d := mockDiscovery(t, volumes)

		result := NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for volumes to reach their desired capacity: [db-nexus-0 db-nexus-1]"), result)
	})

	t.Run("requeue with error when a rendered volume failed to resize", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		volumes := []workloads.Volume{
			{PVC: volumePVC("db-nexus-0", "5Gi", "3Gi"), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-1", "5Gi", "3Gi", corev1.PersistentVolumeClaimNodeResizeError), MountedBy: statefulSet},
		}
		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
		d := mockDiscovery(t, volumes)

		result := NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})).Run(testCtx, doguResource)

		require.EqualError(t, result.Err, "volumes failed to resize: [db-nexus-1]")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("continue while a volume the chart does not render is resizing", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		// nexus-proxy is not part of the rendered chart
		volumes := []workloads.Volume{
			{PVC: volumePVC("existing", "5Gi", "3Gi"), MountedBy: mountedBy("Deployment", "nexus-proxy")},
		}

		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
		d := mockDiscovery(t, volumes)

		result := NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
	})

	for name, modify := range map[string]func(*v3beta1.Dogu){
		"stopped": func(d *v3beta1.Dogu) { d.Spec.Stopped = true },
		"paused":  func(d *v3beta1.Dogu) { d.Spec.PauseReconciliation = true },
	} {
		t.Run("continue without waiting for a "+name+" dogu", func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			modify(doguResource)
			volumes := []workloads.Volume{
				{PVC: volumePVC("db-nexus-0", "5Gi", "3Gi"), MountedBy: statefulSet},
			}

			cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
			d := mockDiscovery(t, volumes)

			result := NewVolumeStatusStep(cs, as, d, newVolumeStatusFakeClient(doguResource, interceptor.Funcs{})).Run(testCtx, doguResource)

			assert.Equal(t, doguv3.Continue(), result)
		})
	}
}

func mockDiscovery(t *testing.T, volumes []workloads.Volume) *MockVolumeDiscovery {
	d := NewMockVolumeDiscovery(t)
	d.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(testWorkloads, nil)
	d.EXPECT().Volumes(mock.Anything, testNamespace, testWorkloads).Return(volumes, nil)
	return d
}
