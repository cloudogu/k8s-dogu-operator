package controllers

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
var imageConfig = &imagev1.ConfigFile{}

//go:embed testdata/ldap-descriptor-cm.yaml
var ldapDescriptorCmBytes []byte
var ldapDescriptorCm = &corev1.ConfigMap{}

func init() {
	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(imageConfigBytes, imageConfig)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(ldapDescriptorCmBytes, ldapDescriptorCm)
	if err != nil {
		panic(err)
	}
}

func TestDoguManager_Install(t *testing.T) {
	testError := errors.New("test error")
	ctx := context.TODO()

	scheme := getInstallScheme()
	resourceGenerator := *NewResourceGenerator(scheme)
	version, err := core.ParseVersion("0.0.0")
	require.NoError(t, err)
	registry := &cesmocks.DoguRegistry{}

	t.Run("successfully install a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)
		_ = client.Create(ctx, ldapCr)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguRegistry, imageRegistry, doguRegistrator)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		//Reset resource version otherwise the resource can't be created
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapDescriptorCm)
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguRegistry, imageRegistry, doguRegistrator)
	})

	t.Run("failed to install dogu with invalid custom descriptor", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getDoguOnlyScheme()).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get custom dogu descriptor")
		mock.AssertExpectationsForObjects(t, doguRegistry, imageRegistry, doguRegistrator)
	})

	t.Run("failed to install dogu with error query descriptor configmap", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		//Reset resource version otherwise the resource can't be created
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, getCustomDoguDescriptorCm("invalid"))
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarschal custom dogu descriptor")
		mock.AssertExpectationsForObjects(t, doguRegistry, imageRegistry, doguRegistrator)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(testError)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, testError))
		mock.AssertExpectationsForObjects(t, doguRegistrator, imageRegistry, doguRegistry)
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		doguRegistry.AssertExpectations(t)
		imageRegistry.AssertExpectations(t)
	})

	t.Run("error get dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(nil, testError)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, testError))
		doguRegistry.AssertExpectations(t)
	})

	t.Run("error create deployment", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getDoguWithCmOnlyScheme()).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		doguManager := NewDoguManager(&version, client, runtime.NewScheme(), &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		doguRegistry.AssertExpectations(t)
	})

	t.Run("error pull image config", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, testError)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, testError))
		doguRegistry.AssertExpectations(t)
	})

	t.Run("error create service resource", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistry.Mock.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		exposedPorts := make(map[string]struct{})
		// wrong port
		exposedPorts["tcp/80"] = struct{}{}
		brokenImageConfig := &imagev1.ConfigFile{Config: imagev1.Config{ExposedPorts: exposedPorts}}
		imageRegistry.Mock.On("PullImageConfig", mock.Anything, mock.Anything).Return(brokenImageConfig, nil)
		doguRegistrator.Mock.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		doguRegistry.AssertExpectations(t)
	})
}

func TestDoguManager_Delete(t *testing.T) {
	scheme := getDoguOnlyScheme()
	ctx := context.TODO()
	resourceGenerator := *NewResourceGenerator(scheme)

	version, err := core.ParseVersion("0.0.0")
	require.NoError(t, err)

	testErr := errors.New("test")

	registry := &cesmocks.DoguRegistry{}

	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistrator.Mock.On("UnregisterDogu", mock.Anything).Return(nil)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)
		_ = client.Create(ctx, ldapCr)

		// when
		err := doguManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguRegistrator)
	})

	t.Run("failed to update dogu status", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update dogu status")
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		ldapCr.ResourceVersion = ""
		_ = client.Create(ctx, ldapCr)
		doguRegistry := &mocks.DoguRegistry{}
		imageRegistry := &mocks.ImageRegistry{}
		doguRegistrator := &mocks.DoguRegistrator{}
		doguRegistrator.Mock.On("UnregisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(testErr)
		doguManager := NewDoguManager(&version, client, scheme, &resourceGenerator, doguRegistry, imageRegistry, doguRegistrator, registry)

		// when
		err := doguManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unregister dogu")
		mock.AssertExpectationsForObjects(t, doguRegistrator)
	})
}

func getDoguOnlyScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)

	return scheme
}

func getDoguWithCmOnlyScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})
	return scheme
}

func getInstallScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

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
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})

	return scheme
}

func getCustomDoguDescriptorCm(value string) *corev1.ConfigMap {
	data := make(map[string]string)
	data["dogu.json"] = value
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ldapDogu.GetSimpleName() + "-descriptor", Namespace: ldapCr.Namespace},
		Data:       data,
	}
}
