package health

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubDiscovery struct {
	workloads workloads.Workloads
	err       error
}

func (s stubDiscovery) Workloads(_ context.Context, _, _ string) (workloads.Workloads, error) {
	return s.workloads, s.err
}

var testDogu = &v3beta1.Dogu{
	ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: "ecosystem"},
	Spec:       v3beta1.DoguSpec{Name: "nexus"},
}

func replicas(n int32) *int32 {
	return &n
}

func readyStatefulSet(name string, desired int32) appsv1.StatefulSet {
	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Generation: 1},
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas(desired)},
		Status:     appsv1.StatefulSetStatus{ObservedGeneration: 1, ReadyReplicas: desired, UpdatedReplicas: desired},
	}
}

func readyDeployment(name string, desired int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Generation: 1},
		Spec:       appsv1.DeploymentSpec{Replicas: replicas(desired)},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1, UpdatedReplicas: desired, AvailableReplicas: desired,
		},
	}
}

func readyDaemonSet(name string, desired int32) appsv1.DaemonSet {
	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Generation: 1},
		Status: appsv1.DaemonSetStatus{
			ObservedGeneration: 1, DesiredNumberScheduled: desired, NumberReady: desired, UpdatedNumberScheduled: desired,
		},
	}
}

func TestChecker_Check(t *testing.T) {
	notReadyStatefulSet := readyStatefulSet("nexus-nexus", 1)
	notReadyStatefulSet.Status.ReadyReplicas = 0

	rollingDeployment := readyDeployment("nexus-proxy", 2)
	rollingDeployment.Status.UpdatedReplicas = 1

	outdatedStatefulSet := readyStatefulSet("nexus-nexus", 1)
	outdatedStatefulSet.Generation = 2

	notReadyDaemonSet := readyDaemonSet("nexus-agent", 3)
	notReadyDaemonSet.Status.NumberReady = 2

	defaultReplicasDeployment := readyDeployment("nexus-proxy", 1)
	defaultReplicasDeployment.Spec.Replicas = nil

	tests := []struct {
		name        string
		dogu        *v3beta1.Dogu
		workloads   workloads.Workloads
		wantHealthy bool
		wantReason  string
		wantMessage string
	}{
		{
			name: "should be healthy if all workloads are ready",
			dogu: testDogu,
			workloads: workloads.Workloads{
				StatefulSets: []appsv1.StatefulSet{readyStatefulSet("nexus-nexus", 1), readyStatefulSet("nexus-nexus-postgresql", 1)},
				Deployments:  []appsv1.Deployment{readyDeployment("nexus-proxy", 2)},
				DaemonSets:   []appsv1.DaemonSet{readyDaemonSet("nexus-agent", 3)},
			},
			wantHealthy: true,
			wantReason:  v3beta1.ReasonSucceeded,
			wantMessage: "all workloads are ready",
		},
		{
			name:        "should be healthy if the deployment uses the default of one replica",
			dogu:        testDogu,
			workloads:   workloads.Workloads{Deployments: []appsv1.Deployment{defaultReplicasDeployment}},
			wantHealthy: true,
			wantReason:  v3beta1.ReasonSucceeded,
		},
		{
			name:        "should not be healthy if the dogu is stopped",
			dogu:        &v3beta1.Dogu{Spec: v3beta1.DoguSpec{Name: "nexus", Stopped: true}},
			workloads:   workloads.Workloads{StatefulSets: []appsv1.StatefulSet{readyStatefulSet("nexus-nexus", 0)}},
			wantReason:  v3beta1.ReasonStopped,
			wantMessage: "the dogu is stopped",
		},
		{
			name:        "should not be healthy if the dogu has no workloads",
			dogu:        testDogu,
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "the dogu has no running workloads",
		},
		{
			name:        "should not be healthy if all workloads are scaled to zero",
			dogu:        testDogu,
			workloads:   workloads.Workloads{StatefulSets: []appsv1.StatefulSet{readyStatefulSet("nexus-nexus", 0)}},
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "the dogu has no running workloads",
		},
		{
			name: "should not be healthy if a statefulset is not ready",
			dogu: testDogu,
			workloads: workloads.Workloads{
				StatefulSets: []appsv1.StatefulSet{notReadyStatefulSet, readyStatefulSet("nexus-nexus-postgresql", 1)},
			},
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "workloads not ready: StatefulSet/nexus-nexus (0/1 ready, 1/1 updated)",
		},
		{
			name:        "should not be healthy while a deployment is rolled out",
			dogu:        testDogu,
			workloads:   workloads.Workloads{Deployments: []appsv1.Deployment{rollingDeployment}},
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "workloads not ready: Deployment/nexus-proxy (2/2 available, 1/2 updated)",
		},
		{
			name:        "should not be healthy if the controller has not observed the latest generation",
			dogu:        testDogu,
			workloads:   workloads.Workloads{StatefulSets: []appsv1.StatefulSet{outdatedStatefulSet}},
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "workloads not ready: StatefulSet/nexus-nexus (1/1 ready, 1/1 updated)",
		},
		{
			name:        "should not be healthy if a daemonset is not ready",
			dogu:        testDogu,
			workloads:   workloads.Workloads{DaemonSets: []appsv1.DaemonSet{notReadyDaemonSet}},
			wantReason:  v3beta1.ReasonWorkloadsNotReady,
			wantMessage: "workloads not ready: DaemonSet/nexus-agent (2/3 ready, 3/3 updated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(stubDiscovery{workloads: tt.workloads})

			got, err := checker.Check(context.Background(), tt.dogu)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHealthy, got.Healthy)
			assert.Equal(t, tt.wantReason, got.Reason)
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}

	t.Run("should return error if the workloads cannot be discovered", func(t *testing.T) {
		checker := NewChecker(stubDiscovery{err: assert.AnError})

		_, err := checker.Check(context.Background(), testDogu)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to get workloads of dogu "nexus"`)
	})
}
