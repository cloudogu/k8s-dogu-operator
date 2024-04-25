package util

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	regMock "github.com/cloudogu/cesapp-lib/registry/mocks"
	config "github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewManagerSet(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		restConfig := &rest.Config{}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		clientSet := fake2.NewSimpleClientset()
		opConfig := &config.OperatorConfig{
			Namespace: "myNamespace",
		}
		doguReg := regMock.NewDoguRegistry(t)
		globalReg := regMock.NewConfigurationContext(t)
		reg := regMock.NewRegistry(t)
		reg.On("DoguRegistry").Return(doguReg, nil)
		reg.On("GlobalConfig").Return(globalReg, nil)
		doguClient := mocks.NewDoguInterface(t)
		ecosystemMock := mocks.NewEcosystemInterface(t)
		ecosystemMock.EXPECT().Dogus("myNamespace").Return(doguClient)
		applier := mocks.NewApplier(t)
		var addImages map[string]string

		// when
		actual, err := NewManagerSet(restConfig, client, clientSet, ecosystemMock, opConfig, reg, applier, addImages)

		// then
		require.NoError(t, err)
		assert.NotNil(t, actual)
	})
}
