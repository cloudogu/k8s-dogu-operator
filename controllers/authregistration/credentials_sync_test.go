package authregistration

import (
	"context"
	"fmt"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	regLibErr "github.com/cloudogu/ces-commons-lib/errors"
	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSensitiveConfigCredentialsSyncer_SyncCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("should skip if auth registration is nil", func(t *testing.T) {
		syncer := &sensitiveConfigCredentialsSyncer{
			secretClient:      newMockSecretClient(t),
			sensitiveDoguRepo: newMockSensitiveDoguConfigRepository(t),
		}

		err := syncer.SyncCredentials(ctx, nil, "redmine", "cas")

		require.NoError(t, err)
	})

	t.Run("should return error if resolvedSecretRef is empty", func(t *testing.T) {
		syncer := &sensitiveConfigCredentialsSyncer{
			secretClient:      newMockSecretClient(t),
			sensitiveDoguRepo: newMockSensitiveDoguConfigRepository(t),
		}

		err := syncer.SyncCredentials(ctx, &authRegApiV1.AuthRegistration{}, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `has no resolved secretRef yet`)
	})

	t.Run("should return error when reading secret fails", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(nil, assert.AnError)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `failed to get secret "redmine-auth-secret"`)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error when secret has no data", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{}}, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `secret "redmine-auth-secret" has no credentials yet`)
	})

	t.Run("should return error when secret values are empty", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("   "), "password": []byte("")}}, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `secret "redmine-auth-secret" has no credentials yet`)
	})

	t.Run("should return error when sensitive config cannot be read", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("john")}}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(config.DoguConfig{}, assert.AnError)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `failed to get sensitive dogu config for "redmine"`)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error when setting credentials into config fails", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{
			config.Key("sa-cas"): config.Value("already-set-as-value"),
		})

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("john")}}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, `failed to set value for path "/sa-cas/username"`)
	})

	t.Run("should update sensitive config with secret data", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{})

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{
			Data: map[string][]byte{
				"username": []byte("john"),
				"password": []byte("secret"),
			},
		}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)
		repo.EXPECT().Update(ctx, mock.MatchedBy(func(cfg config.DoguConfig) bool {
			username, ok := cfg.Get(config.Key("/sa-cas/username"))
			if !ok || username.String() != "john" {
				return false
			}
			password, ok := cfg.Get(config.Key("/sa-cas/password"))
			return ok && password.String() == "secret"
		})).Return(doguCfg, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.NoError(t, err)
	})

	t.Run("should not update sensitive config when credentials are unchanged", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{
			config.Key("sa-cas/username"): config.Value("john"),
			config.Key("sa-cas/password"): config.Value("secret"),
		})

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{
			Data: map[string][]byte{
				"username": []byte("john"),
				"password": []byte("secret"),
			},
		}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.NoError(t, err)
	})

	t.Run("should saveOrMerge when update returns conflict", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{})
		conflictErr := regLibErr.NewConflictError(fmt.Errorf("conflict"))

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("john")}}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(config.DoguConfig{}, conflictErr)
		repo.EXPECT().SaveOrMerge(ctx, mock.Anything).Return(doguCfg, nil)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.NoError(t, err)
	})

	t.Run("should return error when saveOrMerge fails after conflict", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{})
		conflictErr := regLibErr.NewConflictError(fmt.Errorf("conflict"))

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("john")}}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(config.DoguConfig{}, conflictErr)
		repo.EXPECT().SaveOrMerge(ctx, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to save and merge sensitive config for dogu redmine")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error when update fails without conflict", func(t *testing.T) {
		secretClient := newMockSecretClient(t)
		repo := newMockSensitiveDoguConfigRepository(t)
		syncer := &sensitiveConfigCredentialsSyncer{secretClient: secretClient, sensitiveDoguRepo: repo}
		authReg := &authRegApiV1.AuthRegistration{Status: authRegApiV1.AuthRegistrationStatus{ResolvedSecretRef: "redmine-auth-secret"}}
		doguCfg := config.CreateDoguConfig(cescommons.SimpleName("redmine"), config.Entries{})

		secretClient.EXPECT().Get(ctx, "redmine-auth-secret", metav1.GetOptions{}).Return(&corev1.Secret{Data: map[string][]byte{"username": []byte("john")}}, nil)
		repo.EXPECT().Get(ctx, cescommons.SimpleName("redmine")).Return(doguCfg, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		err := syncer.SyncCredentials(ctx, authReg, "redmine", "cas")

		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to update sensitive config for dogu redmine")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
