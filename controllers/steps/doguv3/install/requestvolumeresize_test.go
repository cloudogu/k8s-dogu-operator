package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var resizeScheme = func() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v3beta1.AddToScheme(scheme)
	_ = flux.AddToScheme(scheme)
	return scheme
}()

type deletion struct {
	key         client.ObjectKey
	uid         types.UID
	propagation metav1.DeletionPropagation
}

type deletionRecClient struct {
	client    client.Client
	deletions *[]deletion
}

func newDeletionRecClient(objects ...client.Object) deletionRecClient {
	var deletions []deletion
	k8sClient := fake.NewClientBuilder().
		WithScheme(resizeScheme).
		WithObjects(append(objects, testDoguResource.DeepCopy())...).
		WithStatusSubresource(&v3beta1.Dogu{}).
		WithInterceptorFuncs(interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			options := &client.DeleteOptions{}
			options.ApplyOptions(opts)
			recorded := deletion{key: client.ObjectKeyFromObject(obj)}
			if options.Preconditions != nil && options.Preconditions.UID != nil {
				recorded.uid = *options.Preconditions.UID
			}
			if options.PropagationPolicy != nil {
				recorded.propagation = *options.PropagationPolicy
			}
			deletions = append(deletions, recorded)
			return c.Delete(ctx, obj, opts...)
		}}).
		Build()
	return deletionRecClient{client: k8sClient, deletions: &deletions}
}

func (rc deletionRecClient) exists(t *testing.T, obj client.Object) bool {
	err := rc.client.Get(testCtx, client.ObjectKeyFromObject(obj), obj)
	if apierrors.IsNotFound(err) {
		return false
	}
	require.NoError(t, err)
	return true
}

func prepareResizeStep(client client.Client, t *testing.T, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured) *RequestVolumeResizeStep {
	t.Helper()
	chartService := newMockChartService(t)
	chartService.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil).Maybe()
	chartService.EXPECT().RuntimePatchTemplate(mock.Anything, doguResource).Return(nil, false, nil).Maybe()
	chartService.EXPECT().Render(mock.Anything, doguResource, assembledValues).Return(rendered, nil).Maybe()
	assembler := newMockValueAssembler(t)
	assembler.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta"), []byte(nil)).Return(assembledValues, nil).Maybe()

	planner := NewVolumeResizePlanner(chartService, assembler, pvc.NewResizeChecker(client), client, true)
	return NewRequestVolumeResizeStep(planner, client, client, record.NewFakeRecorder(10))
}

func recordedEvents(step *RequestVolumeResizeStep) []string {
	recorder := step.eventRecorder.(*record.FakeRecorder)
	var events []string
	for {
		select {
		case event := <-recorder.Events:
			events = append(events, event)
		default:
			return events
		}
	}
}

func resizeDeployment(name, claimName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		Name:      name,
		Namespace: testNamespace,
		UID:       types.UID(name + "-uid"),
		Labels:    map[string]string{v3beta1.DoguLabelName: testDoguName},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
			Name:                  "data",
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		}}}}},
	}
}

func resizePod(name, claimName string) *corev1.Pod {
	return &corev1.Pod{Name: name, Namespace: testNamespace, Spec: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
		Name:                  "data",
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
	}}}}.Spec}
}

func helmRelease(suspended bool) *flux.HelmRelease {
	return &flux.HelmRelease{Name: testDoguName, Namespace: testNamespace, Spec: flux.HelmReleaseSpec{Suspend: suspended}}
}

// claimTemplateStatefulSet has the claim template "data", so its pods mount "data-<name>-<ordinal>".
func claimTemplateStatefulSet(name, size string) *appsv1.StatefulSet {
	template := volumePVC("data", size, size)
	template.Namespace = ""
	return &appsv1.StatefulSet{
		APIVersion: "apps/v1", Kind: "StatefulSet",
		Name:      name,
		Namespace: testNamespace,
		UID:       types.UID(name + "-uid"),
		Labels:    map[string]string{v3beta1.DoguLabelName: testDoguName},
		Spec:      appsv1.StatefulSetSpec{VolumeClaimTemplates: []corev1.PersistentVolumeClaim{template}},
	}
}

type mockedResizeStep struct {
	step      *RequestVolumeResizeStep
	discovery *MockVolumeDiscovery
	grower    *MockVolumeGrower
	remover   *MockWorkloadRemover
}

func newMockedResizeStep(t *testing.T, doguResource *v3beta1.Dogu, plan pvc.ResizePlan) mockedResizeStep {
	t.Helper()
	planner := NewMockResizePlanner(t)
	planner.EXPECT().Plan(mock.Anything, doguResource).Return(plan, nil)

	m := mockedResizeStep{
		discovery: NewMockVolumeDiscovery(t),
		grower:    NewMockVolumeGrower(t),
		remover:   NewMockWorkloadRemover(t),
	}
	m.step = &RequestVolumeResizeStep{
		resizePlanner: planner,
		discovery:     m.discovery,
		remover:       m.remover,
		grower:        m.grower,
		eventRecorder: record.NewFakeRecorder(10),
	}
	return m
}

var (
	dataKey      = client.ObjectKey{Namespace: testNamespace, Name: "data"}
	growData     = pvc.ResizeRequest{PVC: dataKey, Current: resource.MustParse("1Gi"), Desired: resource.MustParse("2Gi")}
	nexusMounted = workloads.Workloads{Deployments: []appsv1.Deployment{*resizeDeployment("nexus", "data")}}
	dataVolume   = []pvc.Volume{{
		PVC:       volumePVC("data", "1Gi", "1Gi"),
		MountedBy: []corev1.TypedLocalObjectReference{{Kind: "Deployment", Name: "nexus"}},
	}}
)

func TestRequestVolumeResizeStep_Run(t *testing.T) {
	t.Run("grows the PVC and deletes the workloads that mount it", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(nexusMounted, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, nexusMounted).Return(dataVolume, nil)
		grownClaim := pvc.GrownClaim{PVC: dataKey, Previous: resource.MustParse("1Gi"), Desired: resource.MustParse("2Gi")}
		m.grower.EXPECT().Grow(mock.Anything, []pvc.ResizeRequest{growData}).Return([]pvc.GrownClaim{grownClaim}, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, nexusMounted, []types.UID(nil)).Return(nil, nil)
		m.remover.EXPECT().Unmount(mock.Anything, testNamespace, nexusMounted, dataVolume, []string{"data"}).
			Return(workloads.UnmountResult{Workloads: []string{"Deployment/nexus"}}, nil)

		result := m.step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads that mount resizing volumes to be deleted: [Deployment/nexus]"), result)
		assert.Equal(t, []string{
			"Normal ResizingPVC Growing volume data from 1Gi to 2Gi",
			"Normal ResizingPVC Deleted [Deployment/nexus] that mount resizing volumes",
		}, recordedEvents(m.step))
	})

	t.Run("waits for the deleted pods to terminate", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return(nil, nil)
		m.grower.EXPECT().Grow(mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().Unmount(mock.Anything, testNamespace, mock.Anything, mock.Anything, []string{"data"}).
			Return(workloads.UnmountResult{Pods: []string{"nexus-abc"}, NewlyDeletedPods: []string{"nexus-abc"}}, nil)

		result := m.step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for pods that mount resizing volumes to terminate: [nexus-abc]"), result)
		assert.Equal(t, []string{"Normal ResizingPVC Deleted pods [nexus-abc] that mount resizing volumes"}, recordedEvents(m.step))
	})

	t.Run("continues once nothing mounts the PVC anymore", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return(nil, nil)
		m.grower.EXPECT().Grow(mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().Unmount(mock.Anything, testNamespace, mock.Anything, mock.Anything, mock.Anything).Return(workloads.UnmountResult{}, nil)

		assert.Equal(t, doguv3.Continue(), m.step.Run(testCtx, doguResource))
	})

	t.Run("replaces outdated StatefulSets before it unmounts", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{OutdatedStatefulSets: []types.UID{"postgres-uid"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return(nil, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, workloads.Workloads{}, []types.UID{"postgres-uid"}).Return([]string{"StatefulSet/postgres"}, nil)

		result := m.step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for StatefulSets with outdated claim templates to be deleted: [StatefulSet/postgres]"), result)
		assert.Equal(t, []string{"Normal ResizingPVC Deleted [StatefulSet/postgres] with outdated claim templates, the release recreates them"}, recordedEvents(m.step))
	})

	t.Run("aborts without changes when a PVC is owned by another object than its StatefulSet", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		owned := volumePVC("data", "1Gi", "1Gi")
		owned.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Pod", Name: "nexus-abc", UID: "pod-uid"}}
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return([]pvc.Volume{{PVC: owned}}, nil)

		result := m.step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonResizingPVC, "cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: [data (Pod/nexus-abc)]"), result)
	})

	t.Run("aborts before deleting workloads when the cluster rejects growing the PVC", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(nexusMounted, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, nexusMounted).Return(dataVolume, nil)
		rejection := &pvc.GrowRejectedError{PVC: dataKey, Err: assert.AnError}
		m.grower.EXPECT().Grow(mock.Anything, mock.Anything).Return(nil, rejection)

		result := m.step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonResizingPVC, "growing PVC has been rejected: "+rejection.Error()), result)
	})

	t.Run("requeues with error when growing the PVC fails for another reason", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(nexusMounted, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, nexusMounted).Return(dataVolume, nil)
		m.grower.EXPECT().Grow(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		result := m.step.Run(testCtx, doguResource)

		require.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("requeues with error when replacing StatefulSets fails", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{OutdatedStatefulSets: []types.UID{"postgres-uid"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return(nil, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		result := m.step.Run(testCtx, doguResource)

		require.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("requeues with error when unmounting fails", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, nil)
		m.discovery.EXPECT().Volumes(mock.Anything, testNamespace, workloads.Workloads{}).Return(nil, nil)
		m.grower.EXPECT().Grow(mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().ReplaceStatefulSets(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		m.remover.EXPECT().Unmount(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(workloads.UnmountResult{}, assert.AnError)

		result := m.step.Run(testCtx, doguResource)

		require.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("requeues with error when the workloads cannot be discovered", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{Grow: []pvc.ResizeRequest{growData}, Unmount: []string{"data"}})
		m.discovery.EXPECT().Workloads(mock.Anything, testNamespace, testDoguName).Return(workloads.Workloads{}, assert.AnError)

		result := m.step.Run(testCtx, doguResource)

		require.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("continues without discovering when no volume grows", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		m := newMockedResizeStep(t, doguResource, pvc.ResizePlan{})

		assert.Equal(t, doguv3.Continue(), m.step.Run(testCtx, doguResource))
	})

	t.Run("requeues with error when the resize cannot be planned", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		planner := NewMockResizePlanner(t)
		planner.EXPECT().Plan(mock.Anything, doguResource).Return(pvc.ResizePlan{}, assert.AnError)
		step := &RequestVolumeResizeStep{resizePlanner: planner}

		result := step.Run(testCtx, doguResource)

		require.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	for name, modify := range map[string]func(*v3beta1.Dogu){
		"stopped": func(d *v3beta1.Dogu) { d.Spec.Stopped = true },
		"paused":  func(d *v3beta1.Dogu) { d.Spec.PauseReconciliation = true },
	} {
		t.Run("does not resize a "+name+" dogu", func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			modify(doguResource)
			step := &RequestVolumeResizeStep{resizePlanner: NewMockResizePlanner(t)}

			assert.Equal(t, doguv3.Continue(), step.Run(testCtx, doguResource))
		})
	}
}

func TestRequestVolumeResizeStep_RunIntegration(t *testing.T) {
	t.Run("walks from the resize request to an unmounted PVC over consecutive reconciles", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(
			&claim,
			resizeDeployment("nexus", "data"),
			resizePod("nexus-abc", "data"),
			helmRelease(true),
		)
		doguResource := &v3beta1.Dogu{}
		require.NoError(t, delRecClient.client.Get(testCtx, client.ObjectKeyFromObject(testDoguResource), doguResource))

		claim2Gi := volumePVC("data", "2Gi", "2Gi")
		claim2Gi.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
		rendered := toUnstructured(t, &claim2Gi)

		step := prepareResizeStep(delRecClient.client, t, doguResource, rendered)

		assert.Contains(t, step.Run(testCtx, doguResource).ReadyMessage, "Deployment/nexus")
		assert.Contains(t, step.Run(testCtx, doguResource).ReadyMessage, "nexus-abc")
		assert.Equal(t, doguv3.Continue(), step.Run(testCtx, doguResource))
	})

	t.Run("replaces an outdated StatefulSet before it unmounts its pods over consecutive reconciles", func(t *testing.T) {
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		pod := resizePod("postgres-0", "data-postgres-0")
		delRecClient := newDeletionRecClient(&claim, claimTemplateStatefulSet("postgres", "1Gi"), pod, helmRelease(true))
		doguResource := &v3beta1.Dogu{}
		require.NoError(t, delRecClient.client.Get(testCtx, client.ObjectKeyFromObject(testDoguResource), doguResource))

		step := prepareResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))

		// First Run - STS deleted, Pod still available
		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for StatefulSets with outdated claim templates to be deleted: [StatefulSet/postgres]"), step.Run(testCtx, doguResource))
		require.NoError(t, delRecClient.client.Get(testCtx, client.ObjectKeyFromObject(pod), pod))
		// Second Run - Pod deleted + Requeue
		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for pods that mount resizing volumes to terminate: [postgres-0]"), step.Run(testCtx, doguResource))
		// Third Run - Pod gone
		assert.Equal(t, doguv3.Continue(), step.Run(testCtx, doguResource))
		assert.True(t, apierrors.IsNotFound(delRecClient.client.Get(testCtx, client.ObjectKeyFromObject(pod), pod)))
		assert.Equal(t, []string{
			"Normal ResizingPVC Growing volume data-postgres-0 from 1Gi to 2Gi",
			"Normal ResizingPVC Deleted [StatefulSet/postgres] with outdated claim templates, the release recreates them",
			"Normal ResizingPVC Deleted pods [postgres-0] that mount resizing volumes",
		}, recordedEvents(step))
	})

	t.Run("leaves a recreated StatefulSet alone that a lagging cache still reports as outdated", func(t *testing.T) {
		claim := volumePVC("data-postgres-0", "2Gi", "2Gi")
		recreated := claimTemplateStatefulSet("postgres", "2Gi")
		recreated.UID = "recreated-uid"
		delRecClient := newDeletionRecClient(&claim, recreated, helmRelease(false))
		cache := fake.NewClientBuilder().WithScheme(resizeScheme).WithObjects(claimTemplateStatefulSet("postgres", "1Gi")).Build()
		doguResource := &v3beta1.Dogu{}
		require.NoError(t, delRecClient.client.Get(testCtx, client.ObjectKeyFromObject(testDoguResource), doguResource))
		rendered := toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi"))

		step := prepareResizeStep(delRecClient.client, t, doguResource, rendered)
		step.resizePlanner.(*VolumeResizePlanner).resizeChecker = pvc.NewResizeChecker(cache)
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.Empty(t, *delRecClient.deletions)
	})
}
