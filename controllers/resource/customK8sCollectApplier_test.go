package resource

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.TODO()

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

func Test_collectApplier_CollectApply(t *testing.T) {
	t.Run("should succeed and not return a deployment", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		const yamlFile = `apiVersion: apps/v1
kind: DeploymentStrategy`
		inputResource["aResourceYamlFile"] = yamlFile
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlFile), doguResource.Namespace, doguResource).Return(nil)
		sut := NewCollectApplier(applier)

		// when
		actual, err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
	})
	t.Run("should succeed and return a deployment", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		const yamlDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test`
		const yamlOther = `apiVersion: apps/v1
kind: DeploymentStrategy
metadata:
  name: something-else`
		inputResource["myYamlDeployment"] = yamlDeployment
		inputResource["myOtherResource"] = yamlOther
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlOther), doguResource.Namespace, doguResource).Return(nil)
		sut := NewCollectApplier(applier)

		// when
		actual, err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Equal(t, "test", actual.Name)
	})
	t.Run("should fail", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		const yamlFile = `apiVersion: apps/v1
kind: DeploymentStrategy`
		inputResource["aResourceYamlFile"] = yamlFile
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlFile), doguResource.Namespace, doguResource).Return(assert.AnError)
		sut := NewCollectApplier(applier)

		// when
		_, err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource application failed for file aResourceYamlFile")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail with more than 1 deployments", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		const yamlFile = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test2
`
		inputResource["aResourceYamlFile"] = yamlFile
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		sut := NewCollectApplier(applier)

		// when
		_, err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected exactly one Deployment but found 2 - not sure how to continue")
	})
	t.Run("should succeed without given resources being applied", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		sut := NewCollectApplier(applier)

		// when
		actual, err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
	})
}
