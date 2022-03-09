package controllers

import (
	_ "embed"
	"errors"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	mockErr := errors.New("error")
	scheme := runtime.NewScheme()
	// fake k8sClient
	client := fake.NewClientBuilder().Build()

	t.Run("error get dogu", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(nil, mockErr)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
	})

	t.Run("error create deployment", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
	})

	t.Run("error pull image config", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, mockErr)
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "dogu.cloudogu.com",
			Version: "v1",
			Kind:    "dogu",
		}, ldapCr)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
	})

	t.Run("error create service resource", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		exposedPorts := make(map[string]struct{})
		//wrong port
		exposedPorts["tcp/80"] = struct{}{}
		errImageConfig := &imagev1.ConfigFile{Config: imagev1.Config{ExposedPorts: exposedPorts}}
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(errImageConfig, nil)
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "dogu.cloudogu.com",
			Version: "v1",
			Kind:    "dogu",
		}, ldapCr)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "dogu.cloudogu.com",
			Version: "v1",
			Kind:    "dogu",
		}, ldapCr)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry)

		err := doguManager.Install(ctx, ldapCr)

		assert.NoError(t, err)
	})
}
