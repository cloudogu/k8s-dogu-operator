package resource

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test_collectApplier_CollectApply(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
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
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlOther), doguResource.Namespace, doguResource).Once().Return(nil)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlDeployment), doguResource.Namespace, doguResource).Once().Return(nil)
		sut := NewCollectApplier(applier)

		// when
		err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
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
		err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "resource application failed for file aResourceYamlFile")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should succeed with more than 1 deployments", func(t *testing.T) {
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
		const yamlFile1 = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
`
		const yamlFile2 = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test2
`

		inputResource["aResourceYamlFile"] = yamlFile
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlFile1), doguResource.Namespace, doguResource).Once().Return(nil)
		applier.On("ApplyWithOwner", apply.YamlDocument(yamlFile2), doguResource.Namespace, doguResource).Once().Return(nil)
		sut := NewCollectApplier(applier)

		// when
		err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed without given resources being applied", func(t *testing.T) {
		inputResource := make(map[string]string, 0)
		doguResource := readLdapDoguResource(t)

		applier := mocks.NewApplier(t)
		sut := NewCollectApplier(applier)

		// when
		err := sut.CollectApply(testCtx, inputResource, doguResource)

		// then
		require.NoError(t, err)
	})
}
