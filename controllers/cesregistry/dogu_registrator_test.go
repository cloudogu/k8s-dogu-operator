package cesregistry

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

func TestEtcdDoguRegistrator_RegisterNewDogu(t *testing.T) {
	scheme := getTestScheme()

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

	t.Run("successfully register a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().Register(testCtx, ldapDogu).Return(nil)
		localDoguRegMock.EXPECT().Enable(testCtx, ldapDogu).Return(nil)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, assert.AnError)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("skip registration because dogu is already registered", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(true, nil)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().Register(testCtx, ldapDogu).Return(assert.AnError)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to register dogu")
	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().Register(testCtx, ldapDogu).Return(nil)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
		localDoguRegMock.EXPECT().Enable(testCtx, ldapDogu).Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.RegisterNewDogu(testCtx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable dogu")
	})
}

func TestEtcdDoguRegistrator_RegisterDoguVersion(t *testing.T) {
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a new dogu version", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(true, nil)
		localDoguRegMock.EXPECT().Register(testCtx, ldapDogu).Return(nil)
		localDoguRegMock.EXPECT().Enable(testCtx, ldapDogu).Return(nil)
		registrator := NewCESDoguRegistrator(nil, localDoguRegMock)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, assert.AnError)
		registrator := NewCESDoguRegistrator(nil, localDoguRegMock)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("fail because the dogu is not enabled an no current key exists in upgrade process", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
		registrator := NewCESDoguRegistrator(nil, localDoguRegMock)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not register dogu version: previous version not found")
	})

	t.Run("fail because the dogu cant be registered", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(true, nil)
		localDoguRegMock.EXPECT().Register(testCtx, ldapDogu).Return(assert.AnError)
		registrator := NewCESDoguRegistrator(nil, localDoguRegMock)

		// when
		err := registrator.RegisterDoguVersion(testCtx, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register dogu ldap")
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	t.Run("successfully unregister a dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().UnregisterAllVersions(testCtx, "ldap").Return(nil)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.UnregisterDogu(testCtx, "ldap")

		// then
		require.NoError(t, err)
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().UnregisterAllVersions(testCtx, "ldap").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, localDoguRegMock)

		// when
		err := registrator.UnregisterDogu(testCtx, "ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to unregister dogu")
	})
}
