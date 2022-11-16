package resource

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks"
)

func getResourceGenerator(t *testing.T) *resourceGenerator {
	t.Helper()

	patcher := &mocks.LimitPatcher{}
	patcher.On("RetrievePodLimits", readLdapDoguResource(t)).Return(mocks.NewDoguLimits(t), nil)
	patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)

	return &resourceGenerator{
		scheme:           getTestScheme(),
		doguLimitPatcher: patcher,
	}
}

func TestNewResourceGenerator(t *testing.T) {
	// given
	registry := &cesmocks.Registry{}

	// when
	generator := NewResourceGenerator(getTestScheme(), limit.NewDoguDeploymentLimitPatcher(registry))

	// then
	require.NotNil(t, generator)
}

func TestResourceGenerator_GetDoguDeployment(t *testing.T) {
	oldStage := config.Stage
	defer func() {
		config.Stage = oldStage
	}()
	config.Stage = config.StageProduction
	generator := getResourceGenerator(t)

	t.Run("should fail to create pod template", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		client, clientExists := ldapDogu.Volumes[2].GetClient(operatorVolumeClientName)
		require.True(t, clientExists)
		client.Params = "invalid"

		// when
		_, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume configmap-test")
	})

	t.Run("Return simple deployment", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		actualDeployment, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		assert.Equal(t, expectedDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return simple deployment with development stage", func(t *testing.T) {
		// given
		oldStage := config.Stage
		defer func() {
			config.Stage = oldStage
		}()
		config.Stage = config.StageDevelopment
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		// when
		actualDeployment, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.NoError(t, err)
		expectedCustomDeployment := readLdapDoguExpectedDevelopDeployment(t)
		assert.Equal(t, expectedCustomDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		// when
		_, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Error on retrieving memory limits", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		generatorFail := getResourceGenerator(t)
		patcher := &mocks.LimitPatcher{}
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(mocks.NewDoguLimits(t), assert.AnError)
		generatorFail.doguLimitPatcher = patcher

		// when
		_, err := generatorFail.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})

	t.Run("Error on patching deployment", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		generatorFail := getResourceGenerator(t)
		patcher := &mocks.LimitPatcher{}
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(assert.AnError)
		generatorFail.doguLimitPatcher = patcher

		// when
		_, err := generatorFail.CreateDoguDeployment(ldapDoguResource, ldapDogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {
	generator := getResourceGenerator(t)

	t.Run("Return simple service", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		imageConf := readLdapDoguImageConfig(t)

		// when
		actualService, err := generator.CreateDoguService(ldapDoguResource, imageConf)

		assert.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedService(t), actualService)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		imageConf := readLdapDoguImageConfig(t)

		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguService(ldapDoguResource, imageConf)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func TestResourceGenerator_GetDoguExposedServices(t *testing.T) {
	generator := getResourceGenerator(t)

	t.Run("Return no exposed services when given dogu json does not contain any exposed ports", func(t *testing.T) {
		// given
		dogu := &core.Dogu{
			Name: "ldap",
		}
		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualExposedServices, err := generator.CreateDoguExposedServices(ldapDoguResource, dogu)

		assert.NoError(t, err)
		assert.Len(t, actualExposedServices, 0)
	})

	t.Run("Return all exposed services when given dogu json contains multiple exposed ports", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		actualExposedServices, err := generator.CreateDoguExposedServices(ldapDoguResource, ldapDogu)

		// then
		assert.NoError(t, err)
		assert.Len(t, actualExposedServices, 2)
		assert.Equal(t, readLdapDoguExpectedExposedServices(t), actualExposedServices)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		// when
		_, err := generator.CreateDoguExposedServices(ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func TestResourceGenerator_CreateDoguPVC(t *testing.T) {
	generator := getResourceGenerator(t)

	t.Run("Return simple pvc", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualPVC, err := generator.CreateDoguPVC(ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedDoguPVC(t), actualPVC)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguPVC(ldapDoguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func TestResourceGenerator_CreateReservedPVC(t *testing.T) {
	generator := getResourceGenerator(t)

	t.Run("Return simple pvc", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualPVC, err := generator.CreateReservedPVC(ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedReservedPVC(t), actualPVC)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateReservedPVC(ldapDoguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func TestResourceGenerator_GetDoguSecret(t *testing.T) {
	generator := getResourceGenerator(t)

	t.Run("Return secret", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualSecret, err := generator.CreateDoguSecret(ldapDoguResource, map[string]string{"key": "value"})

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedSecret(t), actualSecret)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguSecret(ldapDoguResource, map[string]string{"key": "value"})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func Test_createLivenessProbe(t *testing.T) {
	t.Run("should should return nil for a dogu without tcp probes", func(t *testing.T) {
		dogu := readLdapDogu(t)
		dogu.HealthChecks = []core.HealthCheck{{
			Type: "http",
		}}

		// when
		actual := createLivenessProbe(dogu)

		// then
		require.Nil(t, actual)
	})
}

func Test_createClientVolumeFromDoguVolume(t *testing.T) {
	t.Run("should fail due to missing client", func(t *testing.T) {
		// given
		doguVolume := core.Volume{
			Name:    "my-volume",
			Clients: []core.VolumeClient{},
		}

		// when
		_, err := createClientVolumeFromDoguVolume(doguVolume)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu volume my-volume has no client")
	})
	t.Run("should fail due to invalid config map type content", func(t *testing.T) {
		// given
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				{
					Name: "k8s-dogu-operator",
					Params: volumeClientParams{
						Type:    "configmap",
						Content: "invalid",
					},
				},
			},
		}

		// when
		_, err := createClientVolumeFromDoguVolume(doguVolume)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read configmap client type content of volume my-volume")
	})
	t.Run("should fail due to unsupported client param type", func(t *testing.T) {
		// given
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				{
					Name: "k8s-dogu-operator",
					Params: volumeClientParams{
						Type: "invalid",
					},
				},
			},
		}

		// when
		_, err := createClientVolumeFromDoguVolume(doguVolume)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported client param type invalid in volume my-volume")
	})
}
