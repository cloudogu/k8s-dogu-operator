package health

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "test-namespace"

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

func Test_doguChecker_checkDoguHealth(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig
	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		testDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{ServiceAccountName: "nothingToSeeHere"},
				},
			},
		}
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataPostgresqlCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.NoError(t, err)
		doguRegistry.AssertExpectations(t)
	})
	t.Run("should fail because of unready replicas", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		testDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "testServiceAccount",
					},
				},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 0}, // trigger failure
		}
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataPostgresqlCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogu appears unhealthy (expected: 1, ready: 0)", err.Error())
		doguRegistry.AssertExpectations(t)
	})
}

func Test_doguChecker_checkDependencyDogusHealthy(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed when all dogu dependencies are in a healthy state", func(t *testing.T) {
		// when

		// then

	})
	t.Run("should fail when at least one dependency dogus is unhealthy", func(t *testing.T) {
		redmineDogu := readTestDataRedmineDogu(t)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.On("Get", "postgresql").Return(redmineDogu, nil)
		dependentDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redmine",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 1},
		}
		dependencyDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "postgresql",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmpty"}},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 0},
		}

		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(dependentDeployment, dependencyDeployment).Build()

		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogu appears unhealthy (expected: 1, ready: 0)", err.Error())
		doguRegistry.AssertExpectations(t)
	})
}

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
}
