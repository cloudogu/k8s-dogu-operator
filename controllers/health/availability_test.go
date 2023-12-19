package health

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestAvailabilityChecker_IsAvailable(t *testing.T) {
	var one int32 = 1
	tests := []struct {
		name       string
		deployment *appsv1.Deployment
		want       bool
	}{
		{
			name:       "should return false if spec replicas is nil and updated replicas is less than its default (1)",
			deployment: &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: nil}, Status: appsv1.DeploymentStatus{UpdatedReplicas: 0}},
			want:       false,
		},
		{
			name:       "should return false if updated replicas is less than spec replicas",
			deployment: &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &one}, Status: appsv1.DeploymentStatus{UpdatedReplicas: 0}},
			want:       false,
		},
		{
			name:       "should return false if replicas is more than updated replicas",
			deployment: &appsv1.Deployment{Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 0}},
			want:       false,
		},
		{
			name:       "should return false if available replicas is less than updated replicas",
			deployment: &appsv1.Deployment{Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 0}},
			want:       false,
		},
		{
			name:       "should return false if available replicas is less than updated replicas",
			deployment: &appsv1.Deployment{Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 0}},
			want:       false,
		},
		{
			name:       "should return true otherwise",
			deployment: &appsv1.Deployment{Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}},
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AvailabilityChecker{}
			if got := ac.IsAvailable(tt.deployment); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
