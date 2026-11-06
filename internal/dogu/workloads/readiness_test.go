package workloads

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestReplicasOrDefault(t *testing.T) {
	t.Run("should default to 1 if replicas are not set", func(t *testing.T) {
		assert.Equal(t, int32(1), ReplicasOrDefault(nil))
	})
	t.Run("should return the set replicas", func(t *testing.T) {
		assert.Equal(t, int32(3), ReplicasOrDefault(new(int32(3))))
	})
	t.Run("should return zero replicas of a stopped workload", func(t *testing.T) {
		assert.Equal(t, int32(0), ReplicasOrDefault(new(int32(0))))
	})
}

func readyDeployment(replicas *int32) appsv1.Deployment {
	desired := ReplicasOrDefault(replicas)
	return appsv1.Deployment{
		Generation: 2,
		Spec:       appsv1.DeploymentSpec{Replicas: replicas},
		Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: desired, AvailableReplicas: desired},
	}
}

func TestIsDeploymentReady(t *testing.T) {
	tests := []struct {
		name   string
		modify func(d *appsv1.Deployment)
		want   bool
	}{
		{name: "should be ready if all replicas are updated and available", modify: func(d *appsv1.Deployment) {}, want: true},
		{name: "should be ready with zero replicas", modify: func(d *appsv1.Deployment) {
			d.Spec.Replicas = new(int32(0))
			d.Status.UpdatedReplicas, d.Status.AvailableReplicas = 0, 0
		}, want: true},
		{name: "should default to 1 replica", modify: func(d *appsv1.Deployment) {
			d.Spec.Replicas = nil
			d.Status.UpdatedReplicas, d.Status.AvailableReplicas = 1, 1
		}, want: true},
		{name: "should be ready if a newer generation is observed", modify: func(d *appsv1.Deployment) { d.Status.ObservedGeneration = 3 }, want: true},
		{name: "should not be ready if the latest generation is not observed", modify: func(d *appsv1.Deployment) { d.Status.ObservedGeneration = 1 }, want: false},
		{name: "should not be ready if replicas are not updated", modify: func(d *appsv1.Deployment) { d.Status.UpdatedReplicas = 1 }, want: false},
		{name: "should not be ready if replicas are not available", modify: func(d *appsv1.Deployment) { d.Status.AvailableReplicas = 1 }, want: false},
		{name: "should not be ready while old replicas are scaled down", modify: func(d *appsv1.Deployment) { d.Status.AvailableReplicas = 3 }, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := readyDeployment(new(int32(2)))
			tt.modify(&deployment)

			assert.Equal(t, tt.want, IsDeploymentReady(deployment))
		})
	}
}

func readyStatefulSet(replicas *int32) appsv1.StatefulSet {
	desired := ReplicasOrDefault(replicas)
	return appsv1.StatefulSet{
		Generation: 2,
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas},
		Status:     appsv1.StatefulSetStatus{ObservedGeneration: 2, UpdatedReplicas: desired, ReadyReplicas: desired},
	}
}

func TestIsStatefulSetReady(t *testing.T) {
	tests := []struct {
		name   string
		modify func(s *appsv1.StatefulSet)
		want   bool
	}{
		{name: "should be ready if all replicas are updated and ready", modify: func(s *appsv1.StatefulSet) {}, want: true},
		{name: "should be ready with zero replicas", modify: func(s *appsv1.StatefulSet) {
			s.Spec.Replicas = new(int32(0))
			s.Status.UpdatedReplicas, s.Status.ReadyReplicas = 0, 0
		}, want: true},
		{name: "should default to 1 replica", modify: func(s *appsv1.StatefulSet) {
			s.Spec.Replicas = nil
			s.Status.UpdatedReplicas, s.Status.ReadyReplicas = 1, 1
		}, want: true},
		{name: "should be ready if a newer generation is observed", modify: func(s *appsv1.StatefulSet) { s.Status.ObservedGeneration = 3 }, want: true},
		{name: "should not be ready if the latest generation is not observed", modify: func(s *appsv1.StatefulSet) { s.Status.ObservedGeneration = 1 }, want: false},
		{name: "should not be ready if replicas are not updated", modify: func(s *appsv1.StatefulSet) { s.Status.UpdatedReplicas = 1 }, want: false},
		{name: "should not be ready if replicas are not ready", modify: func(s *appsv1.StatefulSet) { s.Status.ReadyReplicas = 1 }, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statefulSet := readyStatefulSet(new(int32(2)))
			tt.modify(&statefulSet)

			assert.Equal(t, tt.want, IsStatefulSetReady(statefulSet))
		})
	}
}

func readyDaemonSet(desired int32) appsv1.DaemonSet {
	return appsv1.DaemonSet{
		Generation: 2,
		Status: appsv1.DaemonSetStatus{
			ObservedGeneration: 2, DesiredNumberScheduled: desired, UpdatedNumberScheduled: desired, NumberReady: desired,
		},
	}
}

func TestIsDaemonSetReady(t *testing.T) {
	tests := []struct {
		name   string
		modify func(ds *appsv1.DaemonSet)
		want   bool
	}{
		{name: "should be ready if all scheduled pods are updated and ready", modify: func(ds *appsv1.DaemonSet) {}, want: true},
		{name: "should be ready if no pod is scheduled", modify: func(ds *appsv1.DaemonSet) {
			ds.Status.DesiredNumberScheduled, ds.Status.UpdatedNumberScheduled, ds.Status.NumberReady = 0, 0, 0
		}, want: true},
		{name: "should be ready if a newer generation is observed", modify: func(ds *appsv1.DaemonSet) { ds.Status.ObservedGeneration = 3 }, want: true},
		{name: "should not be ready if the latest generation is not observed", modify: func(ds *appsv1.DaemonSet) { ds.Status.ObservedGeneration = 1 }, want: false},
		{name: "should not be ready if pods are not updated", modify: func(ds *appsv1.DaemonSet) { ds.Status.UpdatedNumberScheduled = 1 }, want: false},
		{name: "should not be ready if pods are not ready", modify: func(ds *appsv1.DaemonSet) { ds.Status.NumberReady = 1 }, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemonSet := readyDaemonSet(2)
			tt.modify(&daemonSet)

			assert.Equal(t, tt.want, IsDaemonSetReady(daemonSet))
		})
	}
}
