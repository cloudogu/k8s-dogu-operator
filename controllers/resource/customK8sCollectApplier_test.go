package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCollectApplier(t *testing.T) {
	t.Run("should return a valid applier", func(t *testing.T) {
		// when
		actual := NewCollectApplier(nil)

		// then
		require.NotNil(t, actual)
		require.IsType(t, &collectApplier{}, actual)
	})
}

func Test_deploymentCollector_Predicate(t *testing.T) {
	t.Run("should return true for a deployment", func(t *testing.T) {
		input := []byte(`apiVersion: apps/v1
kind: Deployment`)

		// when
		actual, err := (&deploymentCollector{}).Predicate(input)

		// then
		require.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("should return false for other resources", func(t *testing.T) {
		input := []byte(`apiVersion: apps/v1
kind: DeploymentStrategy`)

		// when
		actual, err := (&deploymentCollector{}).Predicate(input)

		// then
		require.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("should fail when YAML parsing fails", func(t *testing.T) {
		input := []byte(`hello world`)

		// when
		_, err := (&deploymentCollector{}).Predicate(input)

		// then
		require.Error(t, err)
	})
}

func Test_deploymentCollector_Collect(t *testing.T) {
	t.Run("should collect a deployment", func(t *testing.T) {
		input := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test`)
		sut := &deploymentCollector{}

		// when
		sut.Collect(input)

		// then
		require.NotEmpty(t, sut.collected)
		expected := &appsv1.Deployment{
			TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
		}
		require.Len(t, sut.collected, 1)
		assert.Contains(t, sut.collected, expected)
	})
	// 	t.Run("should not collect anything", func(t *testing.T) {
	// 		input := []byte(`apiVersion: apps/v1
	// kind: DeploymentStrategy`)
	// 		sut := &deploymentCollector{}
	// 		// when
	// 		sut.Collect(input)
	//
	// 		// then
	// 		require.Empty(t, sut.collected)
	// 	})
}

func Test_deploymentAntiFilter_Predicate(t *testing.T) {
	t.Run("should return true for all non-deployment resources", func(t *testing.T) {
		input := []byte(`apiVersion: apps/v1
kind: DeploymentStrategy`)

		// when
		actual, err := (&deploymentAntiFilter{}).Predicate(input)

		// then
		require.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("should return true for deployments", func(t *testing.T) {
		input := []byte(`apiVersion: apps/v1
kind: Deployment`)

		// when
		actual, err := (&deploymentAntiFilter{}).Predicate(input)

		// then
		require.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("should fail when YAML parsing fails", func(t *testing.T) {
		input := []byte(`hello world`)

		// when
		_, err := (&deploymentAntiFilter{}).Predicate(input)

		// then
		require.Error(t, err)
	})
}
