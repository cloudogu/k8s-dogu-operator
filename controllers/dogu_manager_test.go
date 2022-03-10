package controllers

import (
	_ "embed"
	"errors"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte
var ldapCr = &k8sv1.Dogu{}

func init() {
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		panic(err)
	}
}

func TestDoguManager_Install(t *testing.T) {
	resourceGenerator := ResourceGenerator{}
	testError := errors.New("test error")
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)

	// fake k8sClient
	client := fake.NewClientBuilder().Build()

	t.Run("success", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)
		require.NoError(t, err)
	})

	t.Run("error get dogu", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(nil, testError)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.True(t, errors.Is(err, testError))
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error create deployment", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		doguManager := NewDoguManager(client, runtime.NewScheme(), &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error pull image config", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, testError)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.True(t, errors.Is(err, testError))
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error create service resource", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		exposedPorts := make(map[string]struct{})
		// wrong port
		exposedPorts["tcp/80"] = struct{}{}
		brokenImageConfig := &imagev1.ConfigFile{Config: imagev1.Config{ExposedPorts: exposedPorts}}
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(brokenImageConfig, nil)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
		doguRegsitry.AssertExpectations(t)
	})
}
