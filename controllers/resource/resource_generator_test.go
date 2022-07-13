package resource

import (
	_ "embed"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	mocks2 "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
	"testing"
)

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte
var ldapDogu = &core.Dogu{}

//go:embed testdata/ldap-cr.yaml
var ldapDoguResourceBytes []byte
var ldapDoguResource = &k8sv1.Dogu{}

//go:embed testdata/image-config.json
var imageConfBytes []byte
var imageConf = &imagev1.ConfigFile{}

//go:embed testdata/ldap_expectedDeployment.yaml
var expectedDeploymentBytes []byte
var expectedDeployment = &appsv1.Deployment{}

//go:embed testdata/ldap_expectedDeployment_withCustomValues.yaml
var expectedCustomDeploymentBytes []byte
var expectedCustomDeployment = &appsv1.Deployment{}

//go:embed testdata/ldap_expectedDeployment_Development.yaml
var expectedDeploymentDevelopBytes []byte
var expectedDeploymentDevelop = &appsv1.Deployment{}

//go:embed testdata/ldap_expectedPVC.yaml
var expectedPVCBytes []byte
var expectedPVC = &corev1.PersistentVolumeClaim{}

//go:embed testdata/ldap_expectedSecret.yaml
var expectedSecretBytes []byte
var expectedSecret = &corev1.Secret{}

//go:embed testdata/ldap_expectedService.yaml
var expectedServiceBytes []byte
var expectedService = &corev1.Service{}

//go:embed testdata/ldap_expectedExposedServices.yaml
var expectedExposedServicesBytes []byte
var expectedExposedServices = &[]corev1.Service{}

func init() {
	err := json.Unmarshal(ldapBytes, ldapDogu)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(imageConfBytes, imageConf)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(ldapDoguResourceBytes, ldapDoguResource)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedDeploymentBytes, expectedDeployment)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedCustomDeploymentBytes, expectedCustomDeployment)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedDeploymentDevelopBytes, expectedDeploymentDevelop)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedPVCBytes, expectedPVC)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedSecretBytes, expectedSecret)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedServiceBytes, expectedService)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(expectedExposedServicesBytes, expectedExposedServices)
	if err != nil {
		panic(err)
	}
}

func getResourceGenerator() *ResourceGenerator {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})
	patcher := &mocks2.LimitPatcher{}
	patcher.On("RetrievePodLimits", ldapDoguResource).Return(limit.DoguLimits{}, nil)
	patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
	return &ResourceGenerator{
		scheme:           scheme,
		doguLimitPatcher: patcher,
	}
}

func TestNewResourceGenerator(t *testing.T) {
	// given
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})
	registry := &mocks.Registry{}

	// when
	generator := NewResourceGenerator(scheme, limit.NewDoguDeploymentLimitPatcher(registry))

	// then
	require.NotNil(t, generator)
}

func TestResourceGenerator_GetDoguDeployment(t *testing.T) {
	oldStage := config.Stage
	defer func() {
		config.Stage = oldStage
	}()
	config.Stage = config.StageProduction
	generator := getResourceGenerator()

	t.Run("Return simple deployment", func(t *testing.T) {
		// when
		actualDeployment, err := generator.GetDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return simple deployment with given custom deployment", func(t *testing.T) {
		// when
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: ldapDoguResource.Name,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "mytestAccount",
						Containers: []corev1.Container{
							{Name: ldapDoguResource.Name, VolumeMounts: []corev1.VolumeMount{
								{Name: "myTestMount", MountPath: "/my/host/path/test.txt", SubPath: "test.txt"},
							}},
						},
						Volumes: []corev1.Volume{
							{Name: "myTestVolume", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/my/host/path",
									Type: nil,
								},
							}},
						},
					},
				},
			},
		}

		actualDeployment, err := generator.GetDoguDeployment(ldapDoguResource, ldapDogu, deployment)

		bytes, _ := yaml.Marshal(actualDeployment)
		fmt.Println(string(bytes))

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedCustomDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return simple deployment with development stage", func(t *testing.T) {
		// given
		oldStage := config.Stage
		defer func() {
			config.Stage = oldStage
		}()
		config.Stage = config.StageDevelopment

		// when
		actualDeployment, err := generator.GetDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDeploymentDevelop, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.GetDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set controller reference:")
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Error on retrieving memory limits", func(t *testing.T) {
		// when
		generatorFail := getResourceGenerator()
		patcher := &mocks2.LimitPatcher{}
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(limit.DoguLimits{}, assert.AnError)
		generatorFail.doguLimitPatcher = patcher
		_, err := generatorFail.GetDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})

	t.Run("Error on patching deployment", func(t *testing.T) {
		// when
		generatorFail := getResourceGenerator()
		patcher := &mocks2.LimitPatcher{}
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(limit.DoguLimits{}, nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(assert.AnError)
		generatorFail.doguLimitPatcher = patcher
		_, err := generatorFail.GetDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {
	generator := getResourceGenerator()

	t.Run("Return simple service", func(t *testing.T) {
		// when
		actualService, err := generator.GetDoguService(ldapDoguResource, imageConf)

		assert.NoError(t, err)
		assert.Equal(t, expectedService, actualService)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.GetDoguService(ldapDoguResource, imageConf)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set controller reference:")
	})
}

func TestResourceGenerator_GetDoguExposedServices(t *testing.T) {
	generator := getResourceGenerator()

	t.Run("Return no exposed services when given dogu json does not contain any exposed ports", func(t *testing.T) {
		// given
		dogu := &core.Dogu{
			Name: "ldap",
		}

		// when
		actualExposedServices, err := generator.GetDoguExposedServices(ldapDoguResource, dogu)

		assert.NoError(t, err)
		assert.Len(t, actualExposedServices, 0)
	})

	t.Run("Return all exposed services when given dogu json contains multiple exposed ports", func(t *testing.T) {
		// when
		actualExposedServices, err := generator.GetDoguExposedServices(ldapDoguResource, ldapDogu)

		// then
		assert.NoError(t, err)
		assert.Len(t, actualExposedServices, 2)
		assert.Equal(t, *expectedExposedServices, actualExposedServices)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.GetDoguExposedServices(ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set controller reference:")
	})
}

func TestResourceGenerator_GetDoguPVC(t *testing.T) {
	generator := getResourceGenerator()

	t.Run("Return simple pvc", func(t *testing.T) {
		// when
		actualPVC, err := generator.GetDoguPVC(ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedPVC, actualPVC)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.GetDoguPVC(ldapDoguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set controller reference:")
	})
}

func TestResourceGenerator_GetDoguSecret(t *testing.T) {
	generator := getResourceGenerator()

	t.Run("Return secret", func(t *testing.T) {
		// when
		actualSecret, err := generator.GetDoguSecret(ldapDoguResource, map[string]string{"key": "value"})

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedSecret, actualSecret)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.GetDoguSecret(ldapDoguResource, map[string]string{"key": "value"})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set controller reference:")
	})
}
