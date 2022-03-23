package controllers

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte
var ldapCr = &k8sv1.Dogu{}

//go:embed testdata/image-config.json
var imageConfigBytes []byte
var image = &mocks.Image{}
var imageConfig = &imagev1.ConfigFile{}

func init() {
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(imageConfigBytes, imageConfig)
	if err != nil {
		panic(err)
	}
}

func TestDoguManager_Install(t *testing.T) {
	testError := errors.New("test error")
	scheme := runtime.NewScheme()
	resourceGenerator := *NewResourceGenerator(scheme)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &v1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &corev1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &corev1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &corev1.PersistentVolumeClaim{})

	// fake k8sClient
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	t.Run("success", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImage", mock.Anything, mock.Anything).Return(image, nil)
		image.Mock.On("ConfigFile").Return(imageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)
		_ = client.Create(context.TODO(), ldapCr)

		err := doguManager.Install(context.TODO(), ldapCr)
		require.NoError(t, err)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(testError)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)
		_ = client.Create(context.TODO(), ldapCr)

		err := doguManager.Install(context.TODO(), ldapCr)
		require.Error(t, err)

		assert.True(t, errors.Is(err, testError))
		mock.AssertExpectationsForObjects(t, doguRegistrator, imageRegistry, doguRegsitry)
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImage", mock.Anything, mock.Anything).Return(image, nil)
		image.Mock.On("ConfigFile").Return(imageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)

		err := doguManager.Install(context.TODO(), ldapCr)

		assert.Contains(t, err.Error(), "not found")
		doguRegsitry.AssertExpectations(t)
		imageRegistry.AssertExpectations(t)
	})

	t.Run("error get dogu", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(nil, testError)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := doguManager.Install(context.TODO(), ldapCr)

		assert.True(t, errors.Is(err, testError))
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error create deployment", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImage", mock.Anything, mock.Anything).Return(image, nil)
		image.Mock.On("ConfigFile").Return(imageConfig, nil)
		doguManager := NewDoguManager(client, runtime.NewScheme(), &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := doguManager.Install(context.TODO(), ldapCr)

		assert.Error(t, err)
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error pull image config", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImage", mock.Anything, mock.Anything).Return(nil, testError)
		image.Mock.On("ConfigFile").Return(imageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)

		err := doguManager.Install(context.TODO(), ldapCr)

		assert.True(t, errors.Is(err, testError))
		doguRegsitry.AssertExpectations(t)
	})

	t.Run("error create service resource", func(t *testing.T) {
		doguRegsitry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegsitry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		exposedPorts := make(map[string]struct{})
		// wrong port
		exposedPorts["tcp/80"] = struct{}{}
		brokenImageConfig := &imagev1.ConfigFile{Config: imagev1.Config{ExposedPorts: exposedPorts}}
		imageRegistry.Mock.On("PullImage", mock.Anything, mock.Anything).Return(image, nil)
		image.Mock.On("ConfigFile").Return(brokenImageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		doguManager := NewDoguManager(client, scheme, &resourceGenerator, doguRegsitry, imageRegistry, doguRegistrator)

		err := doguManager.Install(context.TODO(), ldapCr)

		assert.Error(t, err)
		doguRegsitry.AssertExpectations(t)
	})
}
