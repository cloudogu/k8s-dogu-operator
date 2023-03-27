package resource

import (
	_ "embed"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/cloudogu/k8s-host-change/pkg/alias"
	"testing"

	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

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
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewResourceGenerator(t *testing.T) {
	// given
	registry := cesmocks.NewRegistry(t)
	globalConfig := cesmocks.NewConfigurationContext(t)
	registry.On("GlobalConfig").Return(globalConfig)

	// when
	generator := NewResourceGenerator(getTestScheme(), limit.NewDoguDeploymentLimitPatcher(registry), alias.NewHostAliasGenerator(registry.GlobalConfig()))

	// then
	require.NotNil(t, generator)
}

func TestResourceGenerator_GetDoguDeployment(t *testing.T) {
	oldStage := config.Stage
	defer func() {
		config.Stage = oldStage
	}()
	config.Stage = config.StageProduction

	t.Run("should fail to create pod template", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		client, clientExists := ldapDogu.Volumes[2].GetClient(doguOperatorClient)
		require.True(t, clientExists)
		client.Params = "invalid"

		// when
		_, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume configmap-test")
	})

	t.Run("Return simple deployment", func(t *testing.T) {
		// when
		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", readLdapDoguResource(t)).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generator := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		actualDeployment, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		assert.Equal(t, expectedDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return simple deployment with service account", func(t *testing.T) {
		// when
		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", readLdapDoguResource(t)).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generator := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		ldapDogu.ServiceAccounts = []core.ServiceAccount{
			{Type: "k8s-dogu-operator", Kind: "k8s"},
		}
		actualDeployment, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.Spec.Template.Spec.ServiceAccountName = "ldap"
		assert.Equal(t, expectedDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return simple deployment with development stage", func(t *testing.T) {
		// given
		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", readLdapDoguResource(t)).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generator := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		oldStage := config.Stage
		defer func() {
			config.Stage = oldStage
		}()
		config.Stage = config.StageDevelopment
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		// when
		actualDeployment, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedCustomDeployment := readLdapDoguExpectedDevelopDeployment(t)
		assert.Equal(t, expectedCustomDeployment, actualDeployment)
		mock.AssertExpectationsForObjects(t, generator.doguLimitPatcher)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", readLdapDoguResource(t)).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generator := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		// when
		_, err := generator.CreateDoguDeployment(ldapDoguResource, ldapDogu)

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
		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(mocks.NewDoguLimits(t), assert.AnError)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generatorFail := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		// when
		_, err := generatorFail.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})

	t.Run("Error on patching deployment", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		patcher := mocks.NewLimitPatcher(t)
		patcher.On("RetrievePodLimits", ldapDoguResource).Return(mocks.NewDoguLimits(t), nil)
		patcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(assert.AnError)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)

		generatorFail := resourceGenerator{
			scheme:             getTestScheme(),
			doguLimitPatcher:   patcher,
			hostAliasGenerator: hostAliasGenerator,
		}

		// when
		_, err := generatorFail.CreateDoguDeployment(ldapDoguResource, ldapDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generatorFail.doguLimitPatcher)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {

	t.Run("Return simple service", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		imageConf := readLdapDoguImageConfig(t)
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		// when
		actualService, err := generator.CreateDoguService(ldapDoguResource, imageConf)

		assert.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedService(t), actualService)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

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

	t.Run("Return no exposed services when given dogu json does not contain any exposed ports", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

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
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		actualExposedServices, err := generator.CreateDoguExposedServices(ldapDoguResource, ldapDogu)

		// then
		assert.NoError(t, err)
		assert.Len(t, actualExposedServices, 2)
		assert.Equal(t, readLdapDoguExpectedExposedServices(t), actualExposedServices)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

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

func TestResourceGenerator_GetDoguSecret(t *testing.T) {

	t.Run("Return secret", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualSecret, err := generator.CreateDoguSecret(ldapDoguResource, map[string]string{"key": "value"})

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedSecret(t), actualSecret)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

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

func Test_getChownInitContainer(t *testing.T) {
	t.Run("success with whitespace in volume path", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "whitespace", Path: "/etc/ldap config/test", Owner: "100", Group: "100"}}}
		doguResource := &corev1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}}
		expectedCommand := []string{"sh", "-c", "mkdir -p \"/etc/ldap config/test\" && chown -R 100:100 \"/etc/ldap config/test\""}

		// when
		container, err := getChownInitContainer(dogu, doguResource)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedCommand, container.Command)
	})

	t.Run("should return nil if volumes are only of type dogu-operator", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Clients: []core.VolumeClient{{Name: "k8s-dogu-operator"}}}}}

		// when
		container, err := getChownInitContainer(dogu, nil)

		// then
		require.NoError(t, err)
		require.Nil(t, container)
	})

	t.Run("should return error if owner cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "3sdf"}}}

		// when
		_, err := getChownInitContainer(dogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse owner id 3sdf from volume test")
	})

	t.Run("should return error if group cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "1", Group: "3sdf"}}}

		// when
		_, err := getChownInitContainer(dogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse group id 3sdf from volume test")
	})

	t.Run("should return error if ids are not greater than 0", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "0", Group: "-1"}}}

		// when
		_, err := getChownInitContainer(dogu, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "owner 0 or group -1 are not greater than 0")
	})
}
