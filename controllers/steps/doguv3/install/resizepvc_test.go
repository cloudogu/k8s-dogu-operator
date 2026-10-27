package install

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
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
	t.Helper()
	err := rc.client.Get(testCtx, client.ObjectKeyFromObject(obj), obj)
	if apierrors.IsNotFound(err) {
		return false
	}
	require.NoError(t, err)
	return true
}

func (rc deletionRecClient) dogu(t *testing.T) *v3beta1.Dogu {
	t.Helper()
	doguResource := &v3beta1.Dogu{}
	require.NoError(t, rc.client.Get(testCtx, client.ObjectKeyFromObject(testDoguResource), doguResource))
	return doguResource
}

func prepareResizeStep(client client.Client, t *testing.T, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured) *ResizePVCStep {
	t.Helper()
	chartService := NewMockChartService(t)
	chartService.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil).Maybe()
	chartService.EXPECT().Render(mock.Anything, doguResource, assembledValues).Return(rendered, nil).Maybe()
	assembler := NewMockValueAssembler(t)
	assembler.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil).Maybe()

	planner := NewVolumeResizePlanner(chartService, assembler, pvc.NewResizeChecker(client), client)
	return NewResizePVCStep(planner, client, client, record.NewFakeRecorder(10), true)
}

func recordedEvents(step *ResizePVCStep) []string {
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

func prepareOnlineResizeStep(client client.Client, t *testing.T, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured) *ResizePVCStep {
	t.Helper()
	step := prepareResizeStep(client, t, doguResource, rendered)
	step.offlineExpansion = false
	return step
}

func renderedPVC(t *testing.T, name, request string) []*unstructured.Unstructured {
	t.Helper()
	claim := volumePVC(name, request, request)
	claim.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
	return toUnstructured(t, &claim)
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

func resizeDaemonSet(name, claimName string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		Name:      name,
		Namespace: testNamespace,
		UID:       types.UID(name + "-uid"),
		Labels:    map[string]string{v3beta1.DoguLabelName: testDoguName},
		Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
			Name:                  "data",
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		}}}}},
	}
}

func resizeStatefulSet(name, claimName string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Name:      name,
		Namespace: testNamespace,
		UID:       types.UID(name + "-uid"),
		Labels:    map[string]string{v3beta1.DoguLabelName: testDoguName},
		Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
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

// rejectingPatches answers patches like the API server for a StorageClass without allowVolumeExpansion.
func rejectingPatches(c client.Client) client.Client {
	return interceptor.NewClient(c.(client.WithWatch), interceptor.Funcs{Patch: func(_ context.Context, _ client.WithWatch, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
		return apierrors.NewForbidden(corev1.Resource("persistentvolumeclaims"), obj.GetName(), errors.New("only dynamically provisioned pvc can be resized and the storageclass that provisions the pvc must support resize"))
	}})
}

func rejectedPatchMessage(claimName string) string {
	return `failed to grow PVC "ecosystem/` + claimName + `" to 2Gi: persistentvolumeclaims "` + claimName + `" is forbidden: only dynamically provisioned pvc can be resized and the storageclass that provisions the pvc must support resize`
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

func TestResizePVCStep_Run(t *testing.T) {
	t.Run("grows the PVC and deletes the workloads that mount it", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		other := resizeDeployment("other", "other-data")
		mountingClaim := volumePVC("data", "1Gi", "1Gi")
		otherClaim := volumePVC("other-data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&mountingClaim, &otherClaim, mounting, other, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads to be deleted and recreated by the release: [Deployment/nexus]"), result)
		claim := &corev1.PersistentVolumeClaim{Name: "data", Namespace: testNamespace}
		require.True(t, delRecClient.exists(t, claim))
		assert.Equal(t, resource.MustParse("2Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.False(t, delRecClient.exists(t, mounting))
		assert.True(t, delRecClient.exists(t, other))
	})

	t.Run("reports the grown PVC as event", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.Contains(t, recordedEvents(step), "Normal ResizingPVC Growing volume data from 1Gi to 2Gi")
	})

	t.Run("reports the deleted workloads as event", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, resizeDeployment("nexus", "data"), helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.Contains(t, recordedEvents(step), "Normal ResizingPVC Deleted [Deployment/nexus] to recreate them on the resized volumes")
	})

	t.Run("keeps the volume status while the workloads of a running resize are gone", func(t *testing.T) {
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, helmRelease(true))
		doguResource := delRecClient.dogu(t)
		resizing := []v3beta1.VolumeStatus{{ClaimName: "data", Size: resource.MustParse("1Gi"), DesiredSize: resource.MustParse("2Gi"), State: v3beta1.VolumeStatusStateResizing}}
		doguResource.Status.VolumeStatus = resizing
		require.NoError(t, delRecClient.client.Status().Update(testCtx, doguResource))

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.Equal(t, resizing, delRecClient.dogu(t).Status.VolumeStatus)
	})

	t.Run("reports only the newly deleted pods as event", func(t *testing.T) {
		terminating := resizePod("nexus-old", "data")
		terminating.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		terminating.Finalizers = []string{"test"}
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, terminating, resizePod("nexus-abc", "data"), helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.Equal(t, []string{"Normal ResizingPVC Deleted pods [nexus-abc] that mount resizing volumes"}, recordedEvents(step))
	})

	t.Run("deletes each workload kind with its propagation policy and a UID precondition", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(
			&claim,
			resizeDeployment("deployment", "data"),
			resizeDaemonSet("daemonset", "data"),
			resizeStatefulSet("statefulset", "data"),
			helmRelease(true),
		)
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.ElementsMatch(t, []deletion{
			{key: client.ObjectKey{Namespace: testNamespace, Name: "deployment"}, uid: "deployment-uid", propagation: metav1.DeletePropagationBackground},
			{key: client.ObjectKey{Namespace: testNamespace, Name: "daemonset"}, uid: "daemonset-uid", propagation: metav1.DeletePropagationBackground},
			{key: client.ObjectKey{Namespace: testNamespace, Name: "statefulset"}, uid: "statefulset-uid", propagation: metav1.DeletePropagationOrphan},
		}, *delRecClient.deletions)
	})

	t.Run("deletes the workload if it still mounts the inProgress resizing PVC", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
		assert.NotZero(t, result.RequeueAfter)
		assert.False(t, delRecClient.exists(t, mounting))
	})

	t.Run("deletes the pod if it still mounts the inProgress resizing PVC", func(t *testing.T) {
		mountingPod := resizePod("nexus-abc", "data")
		otherPod := resizePod("other-abc", "other-data")
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mountingPod, otherPod, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for pods that mount resizing volumes to terminate: [nexus-abc]"), result)
		assert.False(t, delRecClient.exists(t, mountingPod))
		assert.True(t, delRecClient.exists(t, otherPod))
	})

	t.Run("leaves finished pods alone because they no longer mount the PVC", func(t *testing.T) {
		succeeded := resizePod("job-succeeded", "data")
		succeeded.Status.Phase = corev1.PodSucceeded
		failed := resizePod("job-failed", "data")
		failed.Status.Phase = corev1.PodFailed
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, succeeded, failed, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.True(t, delRecClient.exists(t, succeeded))
		assert.True(t, delRecClient.exists(t, failed))
	})

	t.Run("continues once no workload or pod mounts the resizing PVC", func(t *testing.T) {
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
	})

	t.Run("walks from the resize request to an unmounted PVC over consecutive reconciles", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(
			&claim,
			resizeDeployment("nexus", "data"),
			resizePod("nexus-abc", "data"),
			helmRelease(true),
		)
		doguResource := delRecClient.dogu(t)
		rendered := renderedPVC(t, "data", "2Gi")

		step := prepareResizeStep(delRecClient.client, t, doguResource, rendered)

		assert.Contains(t, step.Run(testCtx, doguResource).ReadyMessage, "Deployment/nexus")
		assert.Contains(t, step.Run(testCtx, doguResource).ReadyMessage, "nexus-abc")
		assert.Equal(t, doguv3.Continue(), step.Run(testCtx, doguResource))
	})

	t.Run("leaves workloads of a resumed release alone while the PVC finishes resizing", func(t *testing.T) {
		recreated := resizeDeployment("nexus", "data")
		pod := resizePod("nexus-abc", "data")
		claim := volumePVC("data", "2Gi", "1Gi")
		// helm release not suspended anymore
		delRecClient := newDeletionRecClient(&claim, recreated, pod, helmRelease(false))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.True(t, delRecClient.exists(t, recreated))
		assert.True(t, delRecClient.exists(t, pod))
	})

	t.Run("orphans a StatefulSet with an outdated claim template although its volume already grew", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "2Gi", "2Gi")
		pod := resizePod("postgres-0", "data-postgres-0")
		delRecClient := newDeletionRecClient(&claim, live, pod, helmRelease(false))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads to be deleted and recreated by the release: [StatefulSet/postgres]"), result)
		assert.Equal(t, []deletion{
			{key: client.ObjectKey{Namespace: testNamespace, Name: "postgres"}, uid: "postgres-uid", propagation: metav1.DeletePropagationOrphan},
		}, *delRecClient.deletions)
		assert.True(t, delRecClient.exists(t, pod))
	})

	t.Run("reports the grown PVC as resizing while it waits for workloads and pods", func(t *testing.T) {
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, resizeDeployment("nexus", "data"), helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		step.Run(testCtx, doguResource)

		assert.Equal(t, []v3beta1.VolumeStatus{{
			ClaimName:   "data",
			Size:        resource.MustParse("1Gi"),
			DesiredSize: resource.MustParse("2Gi"),
			State:       v3beta1.VolumeStatusStateResizing,
			MountedBy:   mountedBy("Deployment", "nexus"),
		}}, delRecClient.dogu(t).Status.VolumeStatus)
	})

	t.Run("grows a PVC that is owned by its mounting StatefulSet", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		// persistentVolumeClaimRetentionPolicy.whenDeleted: Delete
		claim.OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "StatefulSet", Name: "postgres", UID: "postgres-uid"}}
		delRecClient := newDeletionRecClient(&claim, live, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
		require.True(t, delRecClient.exists(t, &claim))
		assert.Equal(t, resource.MustParse("2Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.False(t, delRecClient.exists(t, live))
	})

	t.Run("leaves a recreated StatefulSet alone that a lagging cache still reports as outdated", func(t *testing.T) {
		claim := volumePVC("data-postgres-0", "2Gi", "2Gi")
		recreated := claimTemplateStatefulSet("postgres", "2Gi")
		recreated.UID = "recreated-uid"
		delRecClient := newDeletionRecClient(&claim, recreated, helmRelease(false))
		cache := fake.NewClientBuilder().WithScheme(resizeScheme).WithObjects(claimTemplateStatefulSet("postgres", "1Gi")).Build()
		doguResource := delRecClient.dogu(t)
		rendered := toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi"))

		step := prepareResizeStep(delRecClient.client, t, doguResource, rendered)
		step.resizePlanner.resizeChecker = pvc.NewResizeChecker(cache)
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.Empty(t, *delRecClient.deletions)
	})

	t.Run("keeps the pods while the orphaned StatefulSet is still terminating", func(t *testing.T) {
		terminating := claimTemplateStatefulSet("postgres", "1Gi")
		// the garbage collector removes ownerReferences from pods and PVCs before it drops this finalizer
		terminating.Finalizers = []string{metav1.FinalizerOrphanDependents}
		terminating.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		claim := volumePVC("data-postgres-0", "2Gi", "1Gi")
		claim.OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "StatefulSet", Name: "postgres", UID: "postgres-uid"}}
		pod := resizePod("postgres-0", "data-postgres-0")
		delRecClient := newDeletionRecClient(&claim, terminating, pod, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads to be deleted and recreated by the release: [StatefulSet/postgres]"), result)
		assert.True(t, delRecClient.exists(t, pod))
	})

	t.Run("aborts without changes when a PVC is owned by another object than its StatefulSet", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		// persistentVolumeClaimRetentionPolicy.whenScaled: Delete
		claim.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Pod", Name: "postgres-0", UID: "pod-uid"}}
		pod := resizePod("postgres-0", "data-postgres-0")
		delRecClient := newDeletionRecClient(&claim, live, pod, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonResizingPVC, `cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: [data-postgres-0 (Pod/postgres-0)]`), result)
		require.True(t, delRecClient.exists(t, &claim))
		assert.Equal(t, resource.MustParse("1Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.Empty(t, *delRecClient.deletions)
		stored := delRecClient.dogu(t)
		require.Len(t, stored.Status.VolumeStatus, 1)
		assert.Equal(t, v3beta1.VolumeStatusStateResizeFailed, stored.Status.VolumeStatus[0].State)
	})

	t.Run("aborts before deleting workloads when the cluster rejects growing the PVC", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mounting, resizePod("nexus-abc", "data"), helmRelease(true))
		doguResource := delRecClient.dogu(t)

		// reject the patches
		step := prepareResizeStep(rejectingPatches(delRecClient.client), t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonResizingPVC, rejectedPatchMessage("data")), result)
		assert.Empty(t, *delRecClient.deletions)
		assert.Equal(t, []v3beta1.VolumeStatus{{
			ClaimName:   "data",
			Size:        resource.MustParse("1Gi"),
			DesiredSize: resource.MustParse("2Gi"),
			State:       v3beta1.VolumeStatusStateResizeFailed,
			MountedBy:   mountedBy("Deployment", "nexus"),
		}}, delRecClient.dogu(t).Status.VolumeStatus)
	})

	t.Run("continues without changes once the values of a rejected resize are reset", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		// reject the patches
		rejected := prepareResizeStep(rejectingPatches(delRecClient.client), t, doguResource, renderedPVC(t, "data", "2Gi"))
		require.Equal(t, v3beta1.ReasonResizingPVC, rejected.Run(testCtx, doguResource).ReadyReason)
		reset := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "1Gi"))
		result := reset.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.Empty(t, *delRecClient.deletions)
	})

	t.Run("requeues with error when growing the PVC fails for another reason", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
		failing := interceptor.NewClient(delRecClient.client.(client.WithWatch), interceptor.Funcs{Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
			return apierrors.NewTimeoutError("patch timed out", 1)
		}})
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(failing, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		require.ErrorContains(t, result.Err, "patch timed out")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
		assert.Empty(t, *delRecClient.deletions)
	})

	t.Run("continues without changes when no volume grows", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "2Gi", "2Gi")
		// suspended on purpose to ensure that the release state doesn't matter without a resizing volume
		delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.True(t, delRecClient.exists(t, mounting))
	})

	for name, modify := range map[string]func(*v3beta1.Dogu){
		"stopped": func(d *v3beta1.Dogu) { d.Spec.Stopped = true },
		"paused":  func(d *v3beta1.Dogu) { d.Spec.PauseReconciliation = true },
	} {
		t.Run("does not resize a "+name+" dogu", func(t *testing.T) {
			claim := volumePVC("data", "1Gi", "1Gi")
			mounting := resizeDeployment("nexus", "data")
			delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
			doguResource := delRecClient.dogu(t)
			modify(doguResource)

			step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
			result := step.Run(testCtx, doguResource)

			assert.Equal(t, doguv3.Continue(), result)
			require.True(t, delRecClient.exists(t, &claim))
			assert.Equal(t, resource.MustParse("1Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
			assert.True(t, delRecClient.exists(t, mounting))
		})
	}

	t.Run("requeues with error when the volumes cannot be checked", func(t *testing.T) {
		live := volumePVC("data", "1Gi", "1Gi")
		// Volume not bound
		live.Status.Phase = corev1.ClaimPending
		delRecClient := newDeletionRecClient(&live, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		require.ErrorContains(t, result.Err, "is not bound")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})
}

func TestResizePVCStep_RunOnline(t *testing.T) {
	t.Run("grows the PVC while its workload and pods keep running", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		pod := resizePod("nexus-abc", "data")
		claim := volumePVC("data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, mounting, pod, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareOnlineResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		require.True(t, delRecClient.exists(t, &claim))
		assert.Equal(t, resource.MustParse("2Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.Empty(t, *delRecClient.deletions)
	})

	t.Run("orphans the StatefulSet with the outdated claim template and keeps its pods for adoption", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		pod := resizePod("postgres-0", "data-postgres-0")
		delRecClient := newDeletionRecClient(&claim, live, pod, helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareOnlineResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads to be deleted and recreated by the release: [StatefulSet/postgres]"), step.Run(testCtx, doguResource))
		assert.Equal(t, doguv3.Continue(), step.Run(testCtx, doguResource))
		assert.Equal(t, []deletion{
			{key: client.ObjectKey{Namespace: testNamespace, Name: "postgres"}, uid: "postgres-uid", propagation: metav1.DeletePropagationOrphan},
		}, *delRecClient.deletions)
		assert.True(t, delRecClient.exists(t, pod))
	})

	t.Run("aborts before orphaning the outdated StatefulSet when the cluster rejects growing the PVC", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, live, helmRelease(false))
		doguResource := delRecClient.dogu(t)

		// reject the patch
		step := prepareOnlineResizeStep(rejectingPatches(delRecClient.client), t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonResizingPVC, rejectedPatchMessage("data-postgres-0")), result)
		assert.Empty(t, *delRecClient.deletions)
		stored := delRecClient.dogu(t)
		require.Len(t, stored.Status.VolumeStatus, 1)
		assert.Equal(t, v3beta1.VolumeStatusStateResizeFailed, stored.Status.VolumeStatus[0].State)
	})

	t.Run("grows a PVC owned by another object than its StatefulSet as no pod gets deleted", func(t *testing.T) {
		live := claimTemplateStatefulSet("postgres", "1Gi")
		claim := volumePVC("data-postgres-0", "1Gi", "1Gi")
		// persistentVolumeClaimRetentionPolicy.whenScaled: Delete
		claim.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Pod", Name: "postgres-0", UID: "pod-uid"}}
		delRecClient := newDeletionRecClient(&claim, live, resizePod("postgres-0", "data-postgres-0"), helmRelease(true))
		doguResource := delRecClient.dogu(t)

		step := prepareOnlineResizeStep(delRecClient.client, t, doguResource, toUnstructured(t, claimTemplateStatefulSet("postgres", "2Gi")))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
		assert.NoError(t, result.Err)
		require.True(t, delRecClient.exists(t, &claim))
		assert.Equal(t, resource.MustParse("2Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
	})
}
