package authregistration

import (
	"context"
	"reflect"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	authRegFakeClient "github.com/cloudogu/k8s-auth-registration-lib/client/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestNewManager(t *testing.T) {
	t.Run("should create manager with sensitive config credentials syncer", func(t *testing.T) {
		fakeClient := authRegFakeClient.NewSimpleClientset().ApiV1().AuthRegistrations("ecosystem")
		secretClient := k8sfake.NewClientset().CoreV1().Secrets("ecosystem")
		repo := newMockSensitiveDoguConfigRepository(t)

		manager := NewManager(fakeClient, secretClient, repo)

		require.NotNil(t, manager)
		assert.NotNil(t, manager.client)
		assert.Equal(t, fakeClient, manager.client)

		storedSyncer, ok := manager.credentialsSyncer.(*sensitiveConfigCredentialsSyncer)
		require.True(t, ok)

		require.NotNil(t, storedSyncer.secretClient)
		assert.Equal(t, reflect.TypeOf(secretClient), reflect.TypeOf(storedSyncer.secretClient))

		storedRepo, ok := storedSyncer.sensitiveDoguRepo.(*mockSensitiveDoguConfigRepository)
		require.True(t, ok)
		assert.Same(t, repo, storedRepo)
	})
}

func TestAuthRegistrationManager_EnsureAuthRegistration(t *testing.T) {
	ctx := context.Background()

	t.Run("should fail if dogu is nil", func(t *testing.T) {
		manager := &AuthRegistrationManager{}

		err := manager.EnsureAuthRegistration(ctx, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu must not be nil")
	})

	t.Run("should skip if dogu has no CAS service account", func(t *testing.T) {
		manager := &AuthRegistrationManager{
			client:            newMockAuthRegistrationClient(t),
			credentialsSyncer: newMockCredentialsSyncer(t),
		}
		dogu := &cesappcore.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []cesappcore.ServiceAccount{
				{Type: "postgresql"},
				{Type: "k8s-prometheus", Kind: "k8s"},
			},
		}

		err := manager.EnsureAuthRegistration(ctx, dogu)

		require.NoError(t, err)
	})

	t.Run("should fail if protocol parameter is invalid", func(t *testing.T) {
		manager := &AuthRegistrationManager{
			client:            newMockAuthRegistrationClient(t),
			credentialsSyncer: newMockCredentialsSyncer(t),
		}

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"invalid"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse CAS service account parameters")
		assert.ErrorContains(t, err, `unsupported protocol value "invalid"`)
	})

	t.Run("should return error for key value style params", func(t *testing.T) {
		manager := &AuthRegistrationManager{
			client:            newMockAuthRegistrationClient(t),
			credentialsSyncer: newMockCredentialsSyncer(t),
		}

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"protocol=OAUTH", "logoutURL=https://example.org/logout", "service=service-a"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of CAS service account params")
	})

	t.Run("should return get error", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(nil, assert.AnError)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should create auth registration if it does not exist", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")

		desired := &authRegApiV1.AuthRegistration{
			ObjectMeta: metav1.ObjectMeta{Name: expectedName},
			Spec: authRegApiV1.AuthRegistrationSpec{
				Protocol: authRegApiV1.AuthProtocolCAS,
				Consumer: "redmine",
			},
		}

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(nil, newNotFoundErr(expectedName))
		client.EXPECT().Create(
			ctx,
			mock.MatchedBy(func(arg *authRegApiV1.AuthRegistration) bool {
				return arg != nil && arg.Name == desired.Name && reflect.DeepEqual(arg.Spec, desired.Spec)
			}),
			metav1.CreateOptions{},
		).Return(desired, nil)
		syncer.EXPECT().SyncCredentials(ctx, desired, "redmine", "cas").Return(nil)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.NoError(t, err)
	})

	t.Run("should return create error", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(nil, newNotFoundErr(expectedName))
		client.EXPECT().Create(ctx, mock.Anything, metav1.CreateOptions{}).Return(nil, assert.AnError)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should not update when current spec already matches", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")
		existing := &authRegApiV1.AuthRegistration{
			Spec: authRegApiV1.AuthRegistrationSpec{Protocol: authRegApiV1.AuthProtocolCAS, Consumer: "redmine"},
		}

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(existing, nil)
		syncer.EXPECT().SyncCredentials(ctx, existing, "redmine", "cas").Return(nil)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.NoError(t, err)
	})

	t.Run("should update auth registration when spec differs", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")
		existing := &authRegApiV1.AuthRegistration{
			Spec: authRegApiV1.AuthRegistrationSpec{Protocol: authRegApiV1.AuthProtocolCAS, Consumer: "old-consumer"},
		}
		updated := &authRegApiV1.AuthRegistration{
			Spec: authRegApiV1.AuthRegistrationSpec{Protocol: authRegApiV1.AuthProtocolCAS, Consumer: "redmine"},
		}

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(
			ctx,
			mock.MatchedBy(func(arg *authRegApiV1.AuthRegistration) bool {
				return arg != nil && arg.Spec.Protocol == authRegApiV1.AuthProtocolCAS && arg.Spec.Consumer == "redmine"
			}),
			metav1.UpdateOptions{},
		).Return(updated, nil)
		syncer.EXPECT().SyncCredentials(ctx, updated, "redmine", "cas").Return(nil)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.NoError(t, err)
	})

	t.Run("should return update error", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")
		existing := &authRegApiV1.AuthRegistration{
			Spec: authRegApiV1.AuthRegistrationSpec{Protocol: authRegApiV1.AuthProtocolCAS, Consumer: "old-consumer"},
		}

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(existing, nil)
		client.EXPECT().Update(ctx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return sync error", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		syncer := newMockCredentialsSyncer(t)
		manager := &AuthRegistrationManager{client: client, credentialsSyncer: syncer}
		expectedName := createAuthRegistrationName("redmine")
		existing := &authRegApiV1.AuthRegistration{
			Spec: authRegApiV1.AuthRegistrationSpec{Protocol: authRegApiV1.AuthProtocolCAS, Consumer: "redmine"},
		}

		client.EXPECT().Get(ctx, expectedName, metav1.GetOptions{}).Return(existing, nil)
		syncer.EXPECT().SyncCredentials(ctx, existing, "redmine", "cas").Return(assert.AnError)

		err := manager.EnsureAuthRegistration(ctx, newCASDogu([]string{"cas"}))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to synchronize auth registration credentials into sensitive dogu config")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestAuthRegistrationManager_RemoveAuthRegistration(t *testing.T) {
	ctx := context.Background()

	t.Run("should fail if client is nil", func(t *testing.T) {
		manager := &AuthRegistrationManager{}

		err := manager.RemoveAuthRegistration(ctx, "redmine")

		require.Error(t, err)
		assert.ErrorContains(t, err, "auth registration client is not configured")
	})

	t.Run("should remove auth registration", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		manager := &AuthRegistrationManager{client: client}

		client.EXPECT().Delete(ctx, "redmine-authregistration", metav1.DeleteOptions{}).Return(nil)

		err := manager.RemoveAuthRegistration(ctx, "redmine")

		require.NoError(t, err)
	})

	t.Run("should ignore not found on delete", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		manager := &AuthRegistrationManager{client: client}

		client.EXPECT().Delete(ctx, "redmine-authregistration", metav1.DeleteOptions{}).Return(newNotFoundErr("redmine-authregistration"))

		err := manager.RemoveAuthRegistration(ctx, "redmine")

		require.NoError(t, err)
	})

	t.Run("should return delete error", func(t *testing.T) {
		client := newMockAuthRegistrationClient(t)
		manager := &AuthRegistrationManager{client: client}

		client.EXPECT().Delete(ctx, "redmine-authregistration", metav1.DeleteOptions{}).Return(assert.AnError)

		err := manager.RemoveAuthRegistration(ctx, "redmine")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete AuthRegistration")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestCreateAuthRegistrationName(t *testing.T) {
	assert.Equal(t, "redmine-authregistration", createAuthRegistrationName("redmine"))
}

func TestParseLegacyCASServiceAccountParams(t *testing.T) {
	t.Run("should parse account type only", func(t *testing.T) {
		protocol, logoutURL, err := parseLegacyCASServiceAccountParams([]string{"cas"})

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolCAS, protocol)
		assert.Nil(t, logoutURL)
	})

	t.Run("should parse account type and logout uri", func(t *testing.T) {
		protocol, logoutURL, err := parseLegacyCASServiceAccountParams([]string{"oidc", "https://dogu.example/logout"})

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolOIDC, protocol)
		require.NotNil(t, logoutURL)
		assert.Equal(t, "https://dogu.example/logout", *logoutURL)
	})

	t.Run("should parse mixed case account type", func(t *testing.T) {
		protocol, logoutURL, err := parseLegacyCASServiceAccountParams([]string{"oAuTh"})

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolOAuth, protocol)
		assert.Nil(t, logoutURL)
	})

	t.Run("should return error for invalid number of params", func(t *testing.T) {
		_, _, err := parseLegacyCASServiceAccountParams(nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of CAS service account params")
	})

	t.Run("should return error for empty account type", func(t *testing.T) {
		_, _, err := parseLegacyCASServiceAccountParams([]string{"   "})

		require.Error(t, err)
		assert.ErrorContains(t, err, "account_type must not be empty")
	})

	t.Run("should return error for invalid protocol", func(t *testing.T) {
		_, _, err := parseLegacyCASServiceAccountParams([]string{"invalid"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported protocol value")
	})
}

func TestParseProtocol(t *testing.T) {
	t.Run("should parse protocol case-insensitive", func(t *testing.T) {
		result, err := parseProtocol("cAs")

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolCAS, result)
	})

	t.Run("should parse protocol with surrounding spaces", func(t *testing.T) {
		result, err := parseProtocol("  oidc  ")

		require.NoError(t, err)
		assert.Equal(t, authRegApiV1.AuthProtocolOIDC, result)
	})

	t.Run("should return error for unsupported protocol", func(t *testing.T) {
		_, err := parseProtocol("saml")

		require.Error(t, err)
		assert.ErrorContains(t, err, `unsupported protocol value "saml"`)
	})
}

func newCASDogu(params []string) *cesappcore.Dogu {
	return &cesappcore.Dogu{
		Name: "official/redmine",
		ServiceAccounts: []cesappcore.ServiceAccount{
			{Type: "cas", Params: params},
		},
	}
}

func newNotFoundErr(name string) error {
	return k8sErr.NewNotFound(schema.GroupResource{Group: "api.k8s.cloudogu.com", Resource: "authregistrations"}, name)
}
