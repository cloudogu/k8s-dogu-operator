package resource

import (
	_ "embed"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"slices"
	"testing"
)

var testAdditionalImages = map[string]string{"chownInitImage": "busybox:1.36", "exporterImage": "exporter:0.0.1", "dataSeederImage": "dataSeeder:0.0.1"}

const testInitContainerImage = "busybox:1.36"

func TestNewResourceGenerator(t *testing.T) {
	// given
	doguRepoMock := newMockDoguConfigGetter(t)
	hostAliasGenMock := newMockHostAliasGenerator(t)
	securityGenMock := newMockSecurityContextGenerator(t)

	// when
	generator := NewResourceGenerator(getTestScheme(), NewRequirementsGenerator(doguRepoMock), hostAliasGenMock, securityGenMock, testAdditionalImages)

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
			scheme:           getTestScheme(),
			additionalImages: testAdditionalImages,
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		client, clientExists := ldapDogu.Volumes[2].GetClient(doguOperatorClient)
		require.True(t, clientExists)
		client.Params = "invalid"

		// when
		_, err := generator.CreateDoguDeployment(nil, ldapDoguResource, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume configmap-test")
	})

	t.Run("Return simple deployment", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return deployment with security context", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		trueValue := true
		falseValue := false
		fsGroupValue := int64(101)
		fsGroupChangePolicyValue := v1.FSGroupChangeOnRootMismatch
		seLinuxOptions := &v1.SELinuxOptions{
			User:  "myUser",
			Role:  "myRole",
			Type:  "myType",
			Level: "myLevel",
		}
		profileValue := "myProfile"
		seccompProfile := &v1.SeccompProfile{
			Type:             v1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &profileValue,
		}
		appArmorProfile := &v1.AppArmorProfile{
			Type:             v1.AppArmorProfileTypeLocalhost,
			LocalhostProfile: &profileValue,
		}
		podSecurityContext := &v1.PodSecurityContext{
			RunAsNonRoot:        &trueValue,
			SELinuxOptions:      seLinuxOptions,
			SeccompProfile:      seccompProfile,
			AppArmorProfile:     appArmorProfile,
			FSGroup:             &fsGroupValue,
			FSGroupChangePolicy: &fsGroupChangePolicyValue,
		}
		containerSecurityContext := &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Drop: []v1.Capability{"ALL"},
				Add:  []v1.Capability{"DAC_OVERRIDE"},
			},
			RunAsNonRoot:           &trueValue,
			ReadOnlyRootFilesystem: &trueValue,
			Privileged:             &falseValue,
			SELinuxOptions:         seLinuxOptions,
			SeccompProfile:         seccompProfile,
			AppArmorProfile:        appArmorProfile,
		}
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(podSecurityContext, containerSecurityContext)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeploymentWithSecurityContext(t)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return simple deployment with service account", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		ldapDogu.ServiceAccounts = []core.ServiceAccount{
			{Type: "k8s-dogu-operator", Kind: "k8s"},
		}

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.Spec.Template.Spec.ServiceAccountName = "ldap"
		automountServiceAccountToken := true
		expectedDeployment.Spec.Template.Spec.AutomountServiceAccountToken = &automountServiceAccountToken
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

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(requirements, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.Spec.Template.Spec.Containers[0].Resources = requirements
		expectedDeployment.Spec.Template.Spec.InitContainers[0].Resources = requirements
		expectedDeployment.Spec.Template.Spec.InitContainers[1].Resources = requirements
		assert.Equal(t, expectedDeployment, actualDeployment)
	})

	t.Run("Return simple deployment with development stage", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
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

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme, opts ...controllerutil.OwnerReferenceOption) error {
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

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, assert.AnError)

		generatorFail := resourceGenerator{
			scheme:                getTestScheme(),
			requirementsGenerator: requirementsGen,
			additionalImages:      testAdditionalImages,
		}

		// when
		_, err := generatorFail.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Return deployment with exporter-sidecar", func(t *testing.T) {
		// when
		ldapDoguResource := readLdapDoguResource(t)
		ldapDoguResource.Spec.ExportMode = true
		ldapDogu := readLdapDogu(t)

		requirementsGen := newMockRequirementsGenerator(t)
		requirementsGen.EXPECT().Generate(testCtx, ldapDogu).Return(v1.ResourceRequirements{}, nil)
		hostAliasGeneratorMock := newMockHostAliasGenerator(t)
		hostAliasGeneratorMock.EXPECT().Generate(testCtx).Return(nil, nil)
		securityGenMock := newMockSecurityContextGenerator(t)
		securityGenMock.EXPECT().Generate(testCtx, ldapDogu, ldapDoguResource).Return(nil, nil)

		generator := resourceGenerator{
			scheme:                   getTestScheme(),
			requirementsGenerator:    requirementsGen,
			hostAliasGenerator:       hostAliasGeneratorMock,
			securityContextGenerator: securityGenMock,
			additionalImages:         testAdditionalImages,
		}

		actualDeployment, err := generator.CreateDoguDeployment(testCtx, ldapDoguResource, ldapDogu)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeploymentWithExporterSidecar(t)

		assert.Equal(t, expectedDeployment, actualDeployment)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {

	t.Run("Return simple service", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConf := readLdapDoguImageConfig(t)
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		// when
		actualService, err := generator.CreateDoguService(ldapDoguResource, ldapDogu, imageConf)

		assert.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedService(t), actualService)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConf := readLdapDoguImageConfig(t)

		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme, opts ...controllerutil.OwnerReferenceOption) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguService(ldapDoguResource, ldapDogu, imageConf)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func Test_getChownInitContainer(t *testing.T) {
	t.Run("success with whitespace in volume path", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "whitespace", Path: "/etc/ldap config/test", Owner: "100", Group: "100"}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}}
		expectedCommand := []string{"sh", "-c", "mkdir -p \"/etc/ldap config/test\" && chown -R 100:100 \"/etc/ldap config/test\""}
		resources := v1.ResourceRequirements{
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU: resource.MustParse("100m"),
			},
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}

		// when
		container, err := getChownInitContainer(dogu, doguResource, testInitContainerImage, resources)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedCommand, container.Command)
		require.Equal(t, resources, container.Resources)
	})

	t.Run("should return nil if volumes are only of type dogu-operator", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Clients: []core.VolumeClient{{Name: "k8s-dogu-operator"}}}}}
		resources := v1.ResourceRequirements{}

		// when
		container, err := getChownInitContainer(dogu, nil, testInitContainerImage, resources)

		// then
		require.NoError(t, err)
		require.Nil(t, container)
	})

	t.Run("should return error if owner cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "3sdf"}}}
		resources := v1.ResourceRequirements{}

		// when
		_, err := getChownInitContainer(dogu, nil, testInitContainerImage, resources)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse owner id 3sdf from volume test")
	})

	t.Run("should return error if group cannot be parsed", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "1", Group: "3sdf"}}}
		resources := v1.ResourceRequirements{}

		// when
		_, err := getChownInitContainer(dogu, nil, testInitContainerImage, resources)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse group id 3sdf from volume test")
	})

	t.Run("should return error if ids are not greater than 0", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "test", Owner: "0", Group: "-1"}}}
		resources := v1.ResourceRequirements{}

		// when
		_, err := getChownInitContainer(dogu, nil, testInitContainerImage, resources)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "owner 0 or group -1 are not greater than 0")
	})
	t.Run("should return no initContainer if the desired image is empty", func(t *testing.T) {
		// when
		actual, err := getChownInitContainer(nil, nil, "", v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		assert.Nil(t, actual)
	})
}

func Test_getStartupProbeTimeout(t *testing.T) {
	tests := []struct {
		name       string
		setEnv     bool
		timeoutEnv string
		want       int32
	}{
		{name: "should be 1 if not set", setEnv: false, want: 1},
		{name: "should be 1 if unparseable", setEnv: true, timeoutEnv: "banana", want: 1},
		{name: "should parse correctly", setEnv: true, timeoutEnv: "123", want: 123},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(startupProbeTimoutEnv, tt.timeoutEnv)
			}
			assert.Equalf(t, tt.want, getStartupProbeTimeout(), "getStartupProbeTimeout()")
		})
	}
}

func Test_CreateStartupProbe(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  []string
	}{
		{name: "should be ready if not set", state: "", want: []string{"bash", "-c", "[[ $(doguctl state) == \"ready\" ]]"}},
		{name: "should be custom if custom", state: "custom", want: []string{"bash", "-c", "[[ $(doguctl state) == \"custom\" ]]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dogu := &core.Dogu{HealthChecks: []core.HealthCheck{{
				Type:  "state",
				State: tt.state,
			}},
			}
			probe := CreateStartupProbe(dogu)
			assert.Equalf(t, tt.want, probe.ProbeHandler.Exec.Command, "CreateStartupProbe()")
		})
	}
}

func Test_getDataSeederContainer(t *testing.T) {
	t.Run("success with standard volume setup\n", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "testVolume", Path: "/etc/test", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "cm-1",
			Volume:     "testVolume",
			Subfolder:  "testSubFolder",
		}}}}

		expectedDataSeederContainerName := "dogu-data-seeder-init"
		expectedContainerImage := testInitContainerImage
		expectedArgs := []string{"copy", "-source=/datamount/cm-1", "-target=/dogumount/etc/test/testSubFolder"}
		expectedDataVolumeName := doguResource.GetDataVolumeName()

		resources := v1.ResourceRequirements{
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU: resource.MustParse("100m"),
			},
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, resources)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
		require.Equal(t, expectedContainerImage, container.Image)
		require.Equal(t, expectedDataSeederContainerName, container.Name)
		require.Equal(t, 2, len(container.VolumeMounts))
		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == expectedDataVolumeName &&
				mount.MountPath == "/dogumount/etc/test" &&
				mount.SubPath == "testVolume"
		}))

		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == "cm-1" && mount.MountPath == "/datamount/cm-1"
		}))

		require.Equal(t, resources, container.Resources)
	})

	t.Run("success with ephemeral volume setup", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "ephemeralVolume", Path: "/var/cache", NeedsBackup: false}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "redis"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "redis-config",
			Volume:     "ephemeralVolume",
			Subfolder:  "config",
		}}}}

		expectedContainerImage := testInitContainerImage
		expectedArgs := []string{"copy", "-source=/datamount/redis-config", "-target=/dogumount/var/cache/config"}
		expectedEphemeralVolumeName := doguResource.GetEphemeralDataVolumeName()

		resources := v1.ResourceRequirements{
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU: resource.MustParse("50m"),
			},
		}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, expectedContainerImage, resources)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
		require.Equal(t, 2, len(container.VolumeMounts))
		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == expectedEphemeralVolumeName &&
				mount.MountPath == "/dogumount/var/cache" &&
				mount.SubPath == "ephemeralVolume"
		}))
	})

	t.Run("contains all dogu descriptor volume mounts as targets", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{
			{Name: "ephemeralVolume", Path: "/var/cache", NeedsBackup: false},
			{Name: "ephemeralVolumeNotInCR", Path: "/tmp/dir1", NeedsBackup: false},
			{Name: "dataVolumeNotInCR", Path: "/tmp/dir2", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "redis"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "redis-config",
			Volume:     "ephemeralVolume",
			Subfolder:  "config",
		}}}}

		expectedContainerImage := testInitContainerImage
		expectedArgs := []string{"copy", "-source=/datamount/redis-config", "-target=/dogumount/var/cache/config"}
		expectedEphemeralVolumeName := doguResource.GetEphemeralDataVolumeName()
		expectedDataVolumeName := doguResource.GetDataVolumeName()

		resources := v1.ResourceRequirements{
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU: resource.MustParse("50m"),
			},
		}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, expectedContainerImage, resources)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
		require.Equal(t, 4, len(container.VolumeMounts))
		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == expectedEphemeralVolumeName &&
				mount.MountPath == "/dogumount/var/cache" &&
				mount.SubPath == "ephemeralVolume"
		}))
		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == expectedEphemeralVolumeName &&
				mount.MountPath == "/dogumount/tmp/dir1" &&
				mount.SubPath == "ephemeralVolumeNotInCR"
		}))
		require.True(t, slices.ContainsFunc(container.VolumeMounts, func(mount v1.VolumeMount) bool {
			return mount.Name == expectedDataVolumeName &&
				mount.MountPath == "/dogumount/tmp/dir2" &&
				mount.SubPath == "dataVolumeNotInCR"
		}))
	})

	t.Run("success with multiple source volumes mounted to a single target volume", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "configVolume", Path: "/etc/app", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{
			{
				SourceType: "ConfigMap",
				Name:       "main-config",
				Volume:     "configVolume",
				Subfolder:  "main",
			},
			{
				SourceType: "ConfigMap",
				Name:       "extra-config",
				Volume:     "configVolume",
				Subfolder:  "extra",
			},
		}}}

		expectedArgs := []string{
			"copy",
			"-source=/datamount/main-config",
			"-target=/dogumount/etc/app/main",
			"-source=/datamount/extra-config",
			"-target=/dogumount/etc/app/extra",
		}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
		require.Equal(t, 3, len(container.VolumeMounts)) // One target volume + two source volumes

		// The target volume should only be mounted once
		volumeCount := 0
		for _, mount := range container.VolumeMounts {
			if mount.Name == doguResource.GetDataVolumeName() {
				volumeCount++
			}
		}
		require.Equal(t, 1, volumeCount, "Target volume should only be mounted once")
	})

	t.Run("success with one source volume mounted to multiple target volumes", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{
			{Name: "configVolume", Path: "/etc/app/config", NeedsBackup: true},
			{Name: "dataVolume", Path: "/var/lib/app/data", NeedsBackup: true},
		}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{
			{
				SourceType: "ConfigMap",
				Name:       "shared-config",
				Volume:     "configVolume",
				Subfolder:  "shared",
			},
			{
				SourceType: "ConfigMap",
				Name:       "shared-config", // Same source
				Volume:     "dataVolume",    // Different target
				Subfolder:  "imported",
			},
		}}}

		expectedArgs := []string{
			"copy",
			"-source=/datamount/shared-config",
			"-target=/dogumount/etc/app/config/shared",
			"-source=/datamount/shared-config",
			"-target=/dogumount/var/lib/app/data/imported",
		}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)

		// The source volume should only be mounted once
		volumeCount := 0
		for _, mount := range container.VolumeMounts {
			if mount.Name == "shared-config" {
				volumeCount++
			}
		}
		require.Equal(t, 1, volumeCount, "Source volume should only be mounted once")
	})

	t.Run("success with whitespace in volume paths", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "spaceVolume", Path: "/etc/with spaces/conf", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "space-config",
			Volume:     "spaceVolume",
			Subfolder:  "with more spaces",
		}}}}

		expectedArgs := []string{"copy", "-source=/datamount/space-config", "-target=/dogumount/etc/with spaces/conf/with more spaces"}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
	})

	t.Run("failure with cr-Volume not matching a dogu volume name", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "existingVolume", Path: "/etc/test", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "test-config",
			Volume:     "nonExistingVolume", // This doesn't match any volume in dogu
			Subfolder:  "config",
		}}}}

		sut := &resourceGenerator{}

		// when
		_, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find volume name nonExistingVolume")
	})

	t.Run("success with empty subfolder", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "rootVolume", Path: "/etc/app", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{AdditionalMounts: []doguv2.DataMount{{
			SourceType: "ConfigMap",
			Name:       "root-config",
			Volume:     "rootVolume",
			Subfolder:  "", // Empty subfolder
		}}}}

		expectedArgs := []string{"copy", "-source=/datamount/root-config", "-target=/dogumount/etc/app"}

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		require.Equal(t, expectedArgs, container.Args)
	})

	t.Run("no data mounts results in container with only target mounts (for possible deletion)", func(t *testing.T) {
		// given
		dogu := &core.Dogu{Volumes: []core.Volume{{Name: "testVolume", Path: "/etc/test", NeedsBackup: true}}}
		doguResource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "app"}, Spec: doguv2.DoguSpec{
			// No data mounts specified
			AdditionalMounts: []doguv2.DataMount{},
		}}

		expectedDataSeederContainerName := "dogu-data-seeder-init"
		expectedContainerImage := testInitContainerImage
		expectedDataVolumeName := doguResource.GetDataVolumeName()

		sut := &resourceGenerator{}

		// when
		container, err := sut.BuildDataSeederContainer(dogu, doguResource, testInitContainerImage, v1.ResourceRequirements{})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, len(container.Args))
		require.Equal(t, dataSeederArg, container.Args[0])
		require.Equal(t, expectedContainerImage, container.Image)
		require.Equal(t, expectedDataSeederContainerName, container.Name)
		require.Equal(t, 1, len(container.VolumeMounts))
		resultVolumeMount := container.VolumeMounts[0]
		require.Equal(t, expectedDataVolumeName, resultVolumeMount.Name)
		require.Equal(t, "/dogumount/etc/test", resultVolumeMount.MountPath)
		require.Equal(t, "testVolume", resultVolumeMount.SubPath)
	})
}
