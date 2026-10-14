package restart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace = "ecosystem"
	testDoguName  = "nexus"
)

var (
	testCtx       = context.Background()
	testRestartAt = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	olderRestart  = "2026-10-13T11:00:00Z"
	sameRestart   = "2026-10-13T12:00:00Z"
	newerRestart  = "2026-10-13T13:00:00Z"
)

func testScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(objects...).Build()
}

func labels(doguName string) map[string]string {
	return map[string]string{v3beta1.DoguLabelName: doguName}
}

func templateWithAnnotations(annotations map[string]string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Annotations: annotations,
		Spec:        corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:1.0.0"}}},
	}
}

func deployment(name string, annotations map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		Name: name, Namespace: testNamespace, Labels: labels(testDoguName),
		Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(1)), Template: templateWithAnnotations(annotations)},
	}
}

func statefulSet(name string, annotations map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Name: name, Namespace: testNamespace, Labels: labels(testDoguName),
		Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1)), Template: templateWithAnnotations(annotations)},
	}
}

func daemonSet(name string, annotations map[string]string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		Name: name, Namespace: testNamespace, Labels: labels(testDoguName),
		Spec: appsv1.DaemonSetSpec{Template: templateWithAnnotations(annotations)},
	}
}

func restartAnnotation(value string) map[string]string {
	return map[string]string{AnnotationKey: value}
}

func templateAnnotations(t *testing.T, c client.Client, obj client.Object) map[string]string {
	t.Helper()
	require.NoError(t, c.Get(testCtx, client.ObjectKeyFromObject(obj), obj))
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return o.Spec.Template.Annotations
	case *appsv1.StatefulSet:
		return o.Spec.Template.Annotations
	case *appsv1.DaemonSet:
		return o.Spec.Template.Annotations
	}
	t.Fatalf("unsupported workload %T", obj)
	return nil
}

func TestRestarter_Apply(t *testing.T) {
	t.Run("should set the annotation in RFC-3339 on all workloads of the dogu", func(t *testing.T) {
		// given
		d := deployment("nexus-proxy", nil)
		s := statefulSet("nexus", map[string]string{"checksum/config": "abc"})
		ds := daemonSet("nexus-agent", nil)
		other := deployment("redmine", nil)
		other.Labels = labels("redmine")
		c := newClient(d, s, ds, other)
		sut := NewRestarter(c)

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt.In(time.FixedZone("CEST", 2*60*60)))

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, restartAnnotation(sameRestart), templateAnnotations(t, c, &appsv1.Deployment{ObjectMeta: d.ObjectMeta}))
		assert.Equal(t, map[string]string{"checksum/config": "abc", AnnotationKey: sameRestart}, templateAnnotations(t, c, &appsv1.StatefulSet{ObjectMeta: s.ObjectMeta}))
		assert.Equal(t, restartAnnotation(sameRestart), templateAnnotations(t, c, &appsv1.DaemonSet{ObjectMeta: ds.ObjectMeta}))
		assert.Empty(t, templateAnnotations(t, c, &appsv1.Deployment{ObjectMeta: other.ObjectMeta}))
	})
	t.Run("should apply only the restart annotation with the restart field manager", func(t *testing.T) {
		// given
		var payload []byte
		applyOpts := &client.ApplyOptions{}
		c := fake.NewClientBuilder().WithScheme(testScheme()).
			WithObjects(deployment("nexus-proxy", map[string]string{"checksum/config": "abc"})).
			WithInterceptorFuncs(interceptor.Funcs{
				Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
					var err error
					payload, err = json.Marshal(obj)
					applyOpts.ApplyOptions(opts)
					return err
				},
			}).Build()
		sut := NewRestarter(c)

		// when
		_, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		// the payload must not contain any field owned by Helm, otherwise the restart field manager would share them
		assert.JSONEq(t, `{"kind":"Deployment","apiVersion":"apps/v1","metadata":{"name":"nexus-proxy","namespace":"ecosystem"},"spec":{"template":{"metadata":{"annotations":{"k8s.cloudogu.com/restartAt":"2026-10-13T12:00:00Z"}}}}}`, string(payload))
		assert.Equal(t, FieldManager, applyOpts.FieldManager)
		assert.Equal(t, ptr.To(true), applyOpts.Force)
	})
	t.Run("should overwrite older and unparsable annotations", func(t *testing.T) {
		// given
		older := deployment("nexus-proxy", restartAnnotation(olderRestart))
		invalid := deployment("nexus-web", restartAnnotation("yesterday"))
		c := newClient(older, invalid)
		sut := NewRestarter(c)

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, restartAnnotation(sameRestart), templateAnnotations(t, c, &appsv1.Deployment{ObjectMeta: older.ObjectMeta}))
		assert.Equal(t, restartAnnotation(sameRestart), templateAnnotations(t, c, &appsv1.Deployment{ObjectMeta: invalid.ObjectMeta}))
	})
	t.Run("should not change workloads that already carry this restart", func(t *testing.T) {
		// given
		d := deployment("nexus-proxy", restartAnnotation(sameRestart))
		c := newClient(d)
		before := &appsv1.Deployment{}
		require.NoError(t, c.Get(testCtx, client.ObjectKeyFromObject(d), before))
		sut := NewRestarter(c)

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		after := &appsv1.Deployment{}
		require.NoError(t, c.Get(testCtx, client.ObjectKeyFromObject(d), after))
		assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
	})
	t.Run("should never overwrite a newer restart", func(t *testing.T) {
		// given
		newer := deployment("nexus-proxy", restartAnnotation(newerRestart))
		older := statefulSet("nexus", restartAnnotation(olderRestart))
		c := newClient(newer, older)
		sut := NewRestarter(c)

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, restartAnnotation(newerRestart), templateAnnotations(t, c, &appsv1.Deployment{ObjectMeta: newer.ObjectMeta}))
		assert.Equal(t, restartAnnotation(sameRestart), templateAnnotations(t, c, &appsv1.StatefulSet{ObjectMeta: older.ObjectMeta}))
	})
	t.Run("should be superseded if all workloads carry a newer restart", func(t *testing.T) {
		// given
		c := newClient(deployment("nexus-proxy", restartAnnotation(newerRestart)), statefulSet("nexus", restartAnnotation(newerRestart)))
		sut := NewRestarter(c)

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.True(t, superseded)
	})
	t.Run("should not be superseded if the dogu has no workloads", func(t *testing.T) {
		// given
		sut := NewRestarter(newClient())

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
	})
}

func readyDeployment(annotations map[string]string) *appsv1.Deployment {
	d := deployment("nexus-proxy", annotations)
	d.Generation = 2
	d.Status = appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: 1, AvailableReplicas: 1}
	return d
}

func TestRestarter_RolledOut(t *testing.T) {
	t.Run("should be rolled out if all workloads carry this or a newer restart and are ready", func(t *testing.T) {
		// given
		s := statefulSet("nexus", restartAnnotation(newerRestart))
		s.Status = appsv1.StatefulSetStatus{UpdatedReplicas: 1, ReadyReplicas: 1}
		c := newClient(readyDeployment(restartAnnotation(sameRestart)), s)
		sut := NewRestarter(c)

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.True(t, rolledOut)
		assert.Empty(t, message)
	})
	t.Run("should not be rolled out if a workload is not ready yet", func(t *testing.T) {
		// given
		d := readyDeployment(restartAnnotation(sameRestart))
		d.Status.UpdatedReplicas = 0
		c := newClient(d)
		sut := NewRestarter(c)

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "workloads not rolled out: Deployment/nexus-proxy", message)
	})
	t.Run("should not be rolled out if a workload does not carry the restart yet", func(t *testing.T) {
		// given
		// e.g. a stale cache: the workload looks ready but still has the template from before the restart
		c := newClient(readyDeployment(restartAnnotation(olderRestart)))
		sut := NewRestarter(c)

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "workloads not rolled out: Deployment/nexus-proxy", message)
	})
	t.Run("should not be rolled out if the dogu has no workloads", func(t *testing.T) {
		// given
		sut := NewRestarter(newClient())

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "the dogu has no workloads", message)
	})
}
