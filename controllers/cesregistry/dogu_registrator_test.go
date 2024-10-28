package cesregistry

import (
	_ "embed"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	corev1 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCesDoguRegistrator_RegisterNewDogu(t *testing.T) {
	ldapCr := &corev1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "clusterns"},
		Spec: corev1.DoguSpec{
			Name:    "official/ldap",
			Version: "1.0.0",
		},
	}
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}
	coreLdapDoguVersion, lerr := ldapDogu.GetVersion()
	require.NoError(t, lerr)
	simpleLdapDoguName := dogu.SimpleDoguName("ldap")
	ldapDoguVersion := dogu.DoguVersion{
		Name:    simpleLdapDoguName,
		Version: coreLdapDoguVersion,
	}

	t.Run("successfully register a dogu", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(false, nil)
		mockDoguVersionRegistry.EXPECT().Enable(testCtx, ldapDoguVersion).Return(nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Add(testCtx, simpleLdapDoguName, ldapDogu).Return(nil)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(false, assert.AnError)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu ldap is enabled")
	})

	t.Run("skip registration because dogu is already registered", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(true, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(false, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Add(testCtx, simpleLdapDoguName, ldapDogu).Return(assert.AnError)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to register dogu")
	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(false, nil)
		mockDoguVersionRegistry.EXPECT().Enable(testCtx, ldapDoguVersion).Return(assert.AnError)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Add(testCtx, simpleLdapDoguName, ldapDogu).Return(nil)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable dogu")
	})
}

func TestCesDoguRegistrator_RegisterDoguVersion(t *testing.T) {
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}
	coreLdapDoguVersion, lerr := ldapDogu.GetVersion()
	require.NoError(t, lerr)
	simpleLdapDoguName := dogu.SimpleDoguName("ldap")
	ldapDoguVersion := dogu.DoguVersion{
		Name:    simpleLdapDoguName,
		Version: coreLdapDoguVersion,
	}

	t.Run("successfully register a new dogu version", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(true, nil)
		mockDoguVersionRegistry.EXPECT().Enable(testCtx, ldapDoguVersion).Return(nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Add(testCtx, simpleLdapDoguName, ldapDogu).Return(nil)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(true, assert.AnError)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu ldap is enabled")
	})

	t.Run("fail because the dogu is not enabled an no current key exists in upgrade process", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(false, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not register dogu version: previous version not found")
	})

	t.Run("fail because the dogu cant be registered", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleLdapDoguName).Return(ldapDoguVersion, nil)
		mockDoguVersionRegistry.EXPECT().IsEnabled(testCtx, ldapDoguVersion).Return(true, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Add(testCtx, simpleLdapDoguName, ldapDogu).Return(assert.AnError)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register dogu ldap")
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	simpleLdapDoguName := dogu.SimpleDoguName("ldap")
	t.Run("successfully unregister a dogu", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().DeleteAll(testCtx, simpleLdapDoguName).Return(nil)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.UnregisterDogu(testCtx, "ldap")

		// then
		require.NoError(t, err)
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().DeleteAll(testCtx, simpleLdapDoguName).Return(assert.AnError)

		registrator := NewCESDoguRegistrator(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		err := registrator.UnregisterDogu(testCtx, "ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to unregister dogu")
	})
}
