package resource

import (
	_ "embed"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/cloudogu/k8s-host-change/pkg/alias"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"

	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewResourceGenerator(t *testing.T) {
	// given
	registry := cesmocks.NewRegistry(t)
	globalConfig := cesmocks.NewConfigurationContext(t)
	registry.On("GlobalConfig").Return(globalConfig)
	mockImageGetter := mocks.NewAdditionalImageNameGetter(t)

	// when
	generator := NewResourceGenerator(getTestScheme(), NewRequirementsGenerator(registry), alias.NewHostAliasGenerator(registry.GlobalConfig()), mockImageGetter)

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
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.2.3", nil)
		generator := resourceGenerator{
			scheme:                getTestScheme(),
			additionalImageGetter: imageNameGetter,
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		client, clientExists := ldapDogu.Volumes[2].GetClient(doguOperatorClient)
		require.True(t, clientExists)
		client.Params = "invalid"

		// when
		_, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume configmap-test")
	})

	t.Run("Return simple deployment", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generator := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return simple deployment with service account", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		ldapDogu.ServiceAccounts = []core.ServiceAccount{
			{Type: "k8s-dogu-operator", Kind: "k8s"},
		}

		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generator := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.Spec.Template.Spec.ServiceAccountName = "ldap"
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return simple deployment with resource requirements", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		requirements := v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceMemory:           resource.MustParse("250Mi"),
				v1.ResourceCPU:              resource.MustParse("0.5"),
				v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory:           resource.MustParse("150Mi"),
				v1.ResourceCPU:              resource.MustParse("42m"),
				v1.ResourceEphemeralStorage: resource.MustParse("3Gi"),
			},
		}

		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(requirements, nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generator := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.Spec.Template.Spec.Containers[0].Resources = requirements
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return simple deployment with development stage", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generator := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		oldStage := config.Stage
		defer func() {
			config.Stage = oldStage
		}()
		config.Stage = config.StageDevelopment

		// when
		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedCustomDeployment := readLdapDoguExpectedDevelopDeployment(t)
		assert.Equal(t, expectedCustomDeployment, actualDeployment)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generator := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})

	t.Run("Error on generating resource requirements", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		requirementsGenerator := mocks.NewResourceRequirementsGenerator(t)
		requirementsGenerator.EXPECT().Generate(ldapDogu).Return(v1.ResourceRequirements{}, assert.AnError)
		hostAliasGenerator := extMocks.NewHostAliasGenerator(t)
		hostAliasGenerator.EXPECT().Generate().Return(nil, nil)
		imageNameGetter := mocks.NewAdditionalImageNameGetter(t)
		imageNameGetter.EXPECT().ImageForKey(testCtx, "chownInitImage").Return("busybox:1.36", nil)

		generatorFail := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGenerator,
			hostAliasGenerator:    hostAliasGenerator,
			additionalImageGetter: imageNameGetter,
		}

		// when
		_, err := generatorFail.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
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

const testChownInitContainerImage = "busybox:1.36"

func Test_getChownInitContainer(t *testing.T) {
	t.Run("success with whitespace in volume path", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "whitespace", Path: "/etc/ldap config/test", Owner: "100", Group: "100"}}}
		doguResource := &corev1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}}
		expectedCommand := []string{"sh", "-c", "mkdir -p \"/etc/ldap config/test\" && chown -R 100:100 \"/etc/ldap config/test\""}

		// when
		container, err := getChownInitContainer(dogu, doguResource, testChownInitContainerImage)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedCommand, container.Command)
	})

	t.Run("should return nil if volumes are only of type dogu-operator", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Clients: []core.VolumeClient{{Name: "k8s-dogu-operator"}}}}}

		// when
		container, err := getChownInitContainer(dogu, nil, testChownInitContainerImage)

		// then
		require.NoError(t, err)
		require.Nil(t, container)
	})

	t.Run("should return error if owner cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "3sdf"}}}

		// when
		_, err := getChownInitContainer(dogu, nil, testChownInitContainerImage)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse owner id 3sdf from volume test")
	})

	t.Run("should return error if group cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "1", Group: "3sdf"}}}

		// when
		_, err := getChownInitContainer(dogu, nil, testChownInitContainerImage)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse group id 3sdf from volume test")
	})

	t.Run("should return error if ids are not greater than 0", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "0", Group: "-1"}}}

		// when
		_, err := getChownInitContainer(dogu, nil, testChownInitContainerImage)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "owner 0 or group -1 are not greater than 0")
	})
	t.Run("should return no initContainer if the desired image is empty", func(t *testing.T) {
		// when
		actual, err := getChownInitContainer(nil, nil, "")

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
	})
}
