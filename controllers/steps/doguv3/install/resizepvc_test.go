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
		WithObjects(objects...).
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

func prepareResizeStep(client client.Client, t *testing.T, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured) *ResizePVCStep {
	t.Helper()
	chartService := NewMockChartService(t)
	chartService.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return([]byte("meta"), true, nil).Maybe()
	chartService.EXPECT().Render(mock.Anything, doguResource, assembledValues).Return(rendered, nil).Maybe()
	assembler := NewMockValueAssembler(t)
	assembler.EXPECT().Assemble(mock.Anything, doguResource, []byte("meta")).Return(assembledValues, nil).Maybe()

	return NewResizePVCStep(chartService, assembler, pvc.NewResizeChecker(client), workloads.NewDiscovery(client), client, client)
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

func helmRelease(suspended bool) *flux.HelmRelease {
	return &flux.HelmRelease{Name: testDoguName, Namespace: testNamespace, Spec: flux.HelmReleaseSpec{Suspend: suspended}}
}

func TestResizePVCStep_Run(t *testing.T) {
	t.Run("grows the PVC and deletes the workloads that mount it", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		other := resizeDeployment("other", "other-data")
		mountingClaim := volumePVC("data", "1Gi", "1Gi")
		otherClaim := volumePVC("other-data", "1Gi", "1Gi")
		delRecClient := newDeletionRecClient(&mountingClaim, &otherClaim, mounting, other, helmRelease(true))
		doguResource := testDoguResource.DeepCopy()

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for workloads that mount resizing volumes to be deleted: [Deployment/nexus]"), result)
		claim := &corev1.PersistentVolumeClaim{Name: "data", Namespace: testNamespace}
		require.True(t, delRecClient.exists(t, claim))
		assert.Equal(t, resource.MustParse("2Gi"), claim.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.False(t, delRecClient.exists(t, mounting))
		assert.True(t, delRecClient.exists(t, other))
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
		doguResource := testDoguResource.DeepCopy()

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
		doguResource := testDoguResource.DeepCopy()

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
		doguResource := testDoguResource.DeepCopy()

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
		doguResource := testDoguResource.DeepCopy()

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.True(t, delRecClient.exists(t, succeeded))
		assert.True(t, delRecClient.exists(t, failed))
	})

	t.Run("continues once no workload or pod mounts the resizing PVC", func(t *testing.T) {
		claim := volumePVC("data", "2Gi", "1Gi")
		delRecClient := newDeletionRecClient(&claim, helmRelease(true))
		doguResource := testDoguResource.DeepCopy()

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
		doguResource := testDoguResource.DeepCopy()
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
		doguResource := testDoguResource.DeepCopy()

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		assert.True(t, delRecClient.exists(t, recreated))
		assert.True(t, delRecClient.exists(t, pod))
	})

	t.Run("continues without changes when no volume grows", func(t *testing.T) {
		mounting := resizeDeployment("nexus", "data")
		claim := volumePVC("data", "2Gi", "2Gi")
		// suspended on purpose to ensure that the release state doesn't matter without a resizing volume
		delRecClient := newDeletionRecClient(&claim, mounting, helmRelease(true))
		doguResource := testDoguResource.DeepCopy()

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
			doguResource := testDoguResource.DeepCopy()
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
		doguResource := testDoguResource.DeepCopy()

		step := prepareResizeStep(delRecClient.client, t, doguResource, renderedPVC(t, "data", "2Gi"))
		result := step.Run(testCtx, doguResource)

		require.ErrorContains(t, result.Err, "is not bound")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})
}
