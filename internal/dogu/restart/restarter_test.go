package restart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
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

// applyRecorder records the server-side applies of the restarter instead of executing them.
type applyRecorder struct {
	payloads [][]byte
	options  []*client.ApplyOptions
	// restarts maps "<kind>/<name>" to the applied restart annotation
	restarts map[string]string
}

// newRecordingClient returns a client that records every apply and fails it with applyErr.
func newRecordingClient(t *testing.T, applyErr error) (client.Client, *applyRecorder) {
	t.Helper()
	recorder := &applyRecorder{restarts: map[string]string{}}
	c := fake.NewClientBuilder().
		WithInterceptorFuncs(interceptor.Funcs{
			Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
				payload, err := json.Marshal(obj)
				require.NoError(t, err)
				recorder.payloads = append(recorder.payloads, payload)

				applyOpts := &client.ApplyOptions{}
				applyOpts.ApplyOptions(opts)
				recorder.options = append(recorder.options, applyOpts)

				var applied struct {
					Kind     string `json:"kind"`
					Metadata struct {
						Name string `json:"name"`
					} `json:"metadata"`
					Spec struct {
						Template struct {
							Metadata struct {
								Annotations map[string]string `json:"annotations"`
							} `json:"metadata"`
						} `json:"template"`
					} `json:"spec"`
				}
				require.NoError(t, json.Unmarshal(payload, &applied))
				recorder.restarts[applied.Kind+"/"+applied.Metadata.Name] = applied.Spec.Template.Metadata.Annotations[AnnotationKey]

				return applyErr
			},
		}).Build()
	return c, recorder
}

func discoveryReturning(found workloads.Workloads, err error) *mockWorkloadDiscovery {
	return &mockWorkloadDiscovery{
		WorkloadsFunc: func(ctx context.Context, namespace string, doguName string) (workloads.Workloads, error) {
			return found, err
		},
	}
}

func templateWithAnnotations(annotations map[string]string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{Annotations: annotations}
}

func deployment(name string, annotations map[string]string) appsv1.Deployment {
	d := appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(1)), Template: templateWithAnnotations(annotations)}}
	d.Name, d.Namespace = name, testNamespace
	return d
}

func statefulSet(name string, annotations map[string]string) appsv1.StatefulSet {
	s := appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1)), Template: templateWithAnnotations(annotations)}}
	s.Name, s.Namespace = name, testNamespace
	return s
}

func daemonSet(name string, annotations map[string]string) appsv1.DaemonSet {
	ds := appsv1.DaemonSet{Spec: appsv1.DaemonSetSpec{Template: templateWithAnnotations(annotations)}}
	ds.Name, ds.Namespace = name, testNamespace
	return ds
}

func readyDeployment(name string, annotations map[string]string) appsv1.Deployment {
	d := deployment(name, annotations)
	d.Generation = 2
	d.Status = appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: 1, AvailableReplicas: 1}
	return d
}

func readyStatefulSet(name string, annotations map[string]string) appsv1.StatefulSet {
	s := statefulSet(name, annotations)
	s.Status = appsv1.StatefulSetStatus{UpdatedReplicas: 1, ReadyReplicas: 1}
	return s
}

func readyDaemonSet(name string, annotations map[string]string) appsv1.DaemonSet {
	ds := daemonSet(name, annotations)
	ds.Status = appsv1.DaemonSetStatus{DesiredNumberScheduled: 2, UpdatedNumberScheduled: 2, NumberReady: 2}
	return ds
}

func restartAnnotation(value string) map[string]string {
	return map[string]string{AnnotationKey: value}
}

func TestNewRestarter(t *testing.T) {
	// given
	c := fake.NewClientBuilder().Build()

	// when
	sut := NewRestarter(c)

	// then
	assert.Same(t, c, sut.k8s)
	assert.NotNil(t, sut.discovery)
}

func TestRestarter_Apply(t *testing.T) {
	t.Run("should set the annotation in RFC-3339 on all workloads of the dogu", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		discovery := discoveryReturning(workloads.Workloads{
			Deployments:  []appsv1.Deployment{deployment("nexus-proxy", nil)},
			StatefulSets: []appsv1.StatefulSet{statefulSet("nexus", map[string]string{"checksum/config": "abc"})},
			DaemonSets:   []appsv1.DaemonSet{daemonSet("nexus-agent", nil)},
		}, nil)
		sut := &Restarter{k8s: c, discovery: discovery}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt.In(time.FixedZone("CEST", 2*60*60)))

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, map[string]string{
			"Deployment/nexus-proxy": sameRestart,
			"StatefulSet/nexus":      sameRestart,
			"DaemonSet/nexus-agent":  sameRestart,
		}, recorder.restarts)
		require.Len(t, discovery.WorkloadsCalls(), 1)
		assert.Equal(t, testNamespace, discovery.WorkloadsCalls()[0].Namespace)
		assert.Equal(t, testDoguName, discovery.WorkloadsCalls()[0].DoguName)
	})
	t.Run("should apply only the restart annotation with the restart field manager", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		found := workloads.Workloads{Deployments: []appsv1.Deployment{deployment("nexus-proxy", map[string]string{"checksum/config": "abc"})}}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		_, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		require.Len(t, recorder.payloads, 1)
		// the payload must not contain any field owned by Helm, otherwise the restart field manager would share them
		assert.JSONEq(t, `{"kind":"Deployment","apiVersion":"apps/v1","metadata":{"name":"nexus-proxy","namespace":"ecosystem"},"spec":{"template":{"metadata":{"annotations":{"k8s.cloudogu.com/restartAt":"2026-10-13T12:00:00Z"}}}}}`, string(recorder.payloads[0]))
		assert.Equal(t, FieldManager, recorder.options[0].FieldManager)
		assert.Equal(t, ptr.To(true), recorder.options[0].Force)
	})
	t.Run("should overwrite older and unparsable annotations", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		found := workloads.Workloads{Deployments: []appsv1.Deployment{
			deployment("nexus-proxy", restartAnnotation(olderRestart)),
			deployment("nexus-web", restartAnnotation("yesterday")),
		}}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, map[string]string{
			"Deployment/nexus-proxy": sameRestart,
			"Deployment/nexus-web":   sameRestart,
		}, recorder.restarts)
	})
	t.Run("should not change workloads that already carry this restart", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		found := workloads.Workloads{Deployments: []appsv1.Deployment{deployment("nexus-proxy", restartAnnotation(sameRestart))}}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Empty(t, recorder.payloads)
	})
	t.Run("should never overwrite a newer restart", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		found := workloads.Workloads{
			Deployments:  []appsv1.Deployment{deployment("nexus-proxy", restartAnnotation(newerRestart))},
			StatefulSets: []appsv1.StatefulSet{statefulSet("nexus", restartAnnotation(olderRestart))},
		}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Equal(t, map[string]string{"StatefulSet/nexus": sameRestart}, recorder.restarts)
	})
	t.Run("should be superseded if all workloads carry a newer restart", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		found := workloads.Workloads{
			Deployments:  []appsv1.Deployment{deployment("nexus-proxy", restartAnnotation(newerRestart))},
			StatefulSets: []appsv1.StatefulSet{statefulSet("nexus", restartAnnotation(newerRestart))},
		}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.True(t, superseded)
		assert.Empty(t, recorder.payloads)
	})
	t.Run("should not be superseded if the dogu has no workloads", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		sut := &Restarter{k8s: c, discovery: discoveryReturning(workloads.Workloads{}, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, superseded)
		assert.Empty(t, recorder.payloads)
	})
	t.Run("should fail if the workloads cannot be discovered", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, nil)
		sut := &Restarter{k8s: c, discovery: discoveryReturning(workloads.Workloads{}, assert.AnError)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get workloads for dogu nexus in namespace ecosystem")
		assert.False(t, superseded)
		assert.Empty(t, recorder.payloads)
	})
	t.Run("should try all workloads and join the errors if applying fails", func(t *testing.T) {
		// given
		c, recorder := newRecordingClient(t, assert.AnError)
		found := workloads.Workloads{
			Deployments:  []appsv1.Deployment{deployment("nexus-proxy", nil)},
			StatefulSets: []appsv1.StatefulSet{statefulSet("nexus", nil)},
		}
		sut := &Restarter{k8s: c, discovery: discoveryReturning(found, nil)}

		// when
		superseded, err := sut.Apply(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to set restart annotation on Deployment/nexus-proxy of dogu "nexus"`)
		assert.ErrorContains(t, err, `failed to set restart annotation on StatefulSet/nexus of dogu "nexus"`)
		assert.False(t, superseded)
		assert.Len(t, recorder.payloads, 2)
	})
}

func TestRestarter_RolledOut(t *testing.T) {
	t.Run("should be rolled out if all workloads carry this or a newer restart and are ready", func(t *testing.T) {
		// given
		discovery := discoveryReturning(workloads.Workloads{
			Deployments:  []appsv1.Deployment{readyDeployment("nexus-proxy", restartAnnotation(sameRestart))},
			StatefulSets: []appsv1.StatefulSet{readyStatefulSet("nexus", restartAnnotation(newerRestart))},
			DaemonSets:   []appsv1.DaemonSet{readyDaemonSet("nexus-agent", restartAnnotation(sameRestart))},
		}, nil)
		sut := &Restarter{discovery: discovery}

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.True(t, rolledOut)
		assert.Empty(t, message)
		require.Len(t, discovery.WorkloadsCalls(), 1)
		assert.Equal(t, testNamespace, discovery.WorkloadsCalls()[0].Namespace)
		assert.Equal(t, testDoguName, discovery.WorkloadsCalls()[0].DoguName)
	})
	t.Run("should not be rolled out if a workload is not ready yet", func(t *testing.T) {
		// given
		d := readyDeployment("nexus-proxy", restartAnnotation(sameRestart))
		d.Status.UpdatedReplicas = 0
		sut := &Restarter{discovery: discoveryReturning(workloads.Workloads{Deployments: []appsv1.Deployment{d}}, nil)}

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
		found := workloads.Workloads{Deployments: []appsv1.Deployment{readyDeployment("nexus-proxy", restartAnnotation(olderRestart))}}
		sut := &Restarter{discovery: discoveryReturning(found, nil)}

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "workloads not rolled out: Deployment/nexus-proxy", message)
	})
	t.Run("should list all pending workloads", func(t *testing.T) {
		// given
		notReady := readyDaemonSet("nexus-agent", restartAnnotation(sameRestart))
		notReady.Status.NumberReady = 1
		found := workloads.Workloads{
			Deployments:  []appsv1.Deployment{readyDeployment("nexus-proxy", restartAnnotation(sameRestart))},
			StatefulSets: []appsv1.StatefulSet{readyStatefulSet("nexus", restartAnnotation("yesterday"))},
			DaemonSets:   []appsv1.DaemonSet{notReady},
		}
		sut := &Restarter{discovery: discoveryReturning(found, nil)}

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "workloads not rolled out: StatefulSet/nexus, DaemonSet/nexus-agent", message)
	})
	t.Run("should not be rolled out if the dogu has no workloads", func(t *testing.T) {
		// given
		sut := &Restarter{discovery: discoveryReturning(workloads.Workloads{}, nil)}

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.NoError(t, err)
		assert.False(t, rolledOut)
		assert.Equal(t, "the dogu has no workloads", message)
	})
	t.Run("should fail if the workloads cannot be discovered", func(t *testing.T) {
		// given
		sut := &Restarter{discovery: discoveryReturning(workloads.Workloads{}, assert.AnError)}

		// when
		rolledOut, message, err := sut.RolledOut(testCtx, testNamespace, testDoguName, testRestartAt)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to get workloads for dogu "nexus"`)
		assert.False(t, rolledOut)
		assert.Empty(t, message)
	})
}
