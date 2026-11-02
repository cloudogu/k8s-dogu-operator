package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func sensitiveConfigWith(configYaml string) *corev1.Secret {
	return &corev1.Secret{Data: map[string][]byte{"config.yaml": []byte(configYaml)}}
}

func TestServiceAccountCredentials(t *testing.T) {
	t.Run("flattens the credentials of every service account and ignores the rest", func(t *testing.T) {
		config := sensitiveConfigWith(`
sa-postgresql:
  username: nexus_uiuxsp
  password: secret1
sa-cas:
  username: nexus
admin_password: secret2
ldap:
  bind:
    password: secret3
`)

		credentials, err := ServiceAccountCredentials(config)

		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"postgresql_username": []byte("nexus_uiuxsp"),
			"postgresql_password": []byte("secret1"),
			"cas_username":        []byte("nexus"),
		}, credentials)
	})

	t.Run("ignores service account entries which are no plain string credential", func(t *testing.T) {
		config := sensitiveConfigWith(`
sa-postgresql:
  username: nexus
  port: 5432
  nested:
    key: value
sa-cas: plain
`)

		credentials, err := ServiceAccountCredentials(config)

		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{"postgresql_username": []byte("nexus")}, credentials)
	})

	t.Run("fails if two service account entries result in the same key", func(t *testing.T) {
		config := sensitiveConfigWith(`
sa-ldap_bind:
  password: secret1
sa-ldap:
  bind_password: secret2
`)

		_, err := ServiceAccountCredentials(config)

		assert.ErrorContains(t, err, `sensitive config paths "sa-ldap.bind_password" and "sa-ldap_bind.password" both result in the migration secret key "ldap_bind_password"`)
	})

	t.Run("extracts the credentials of a real v2 sensitive config", func(t *testing.T) {
		config := sensitiveConfigWith(`
sa-cas:
    cas_client_id: baseline
sa-nexus:
    password: nexus-password
    repository: baselines
    username: service_account_baseline_e71emq
sa-scm:
    password: scm-password
    username: baseline_5m4wij
`)

		credentials, err := ServiceAccountCredentials(config)

		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"cas_cas_client_id": []byte("baseline"),
			"nexus_password":    []byte("nexus-password"),
			"nexus_repository":  []byte("baselines"),
			"nexus_username":    []byte("service_account_baseline_e71emq"),
			"scm_password":      []byte("scm-password"),
			"scm_username":      []byte("baseline_5m4wij"),
		}, credentials)
	})

	t.Run("returns no credentials for an empty sensitive config", func(t *testing.T) {
		credentials, err := ServiceAccountCredentials(sensitiveConfigWith(""))

		require.NoError(t, err)
		assert.Empty(t, credentials)
	})

	t.Run("fails if the sensitive config is no map", func(t *testing.T) {
		_, err := ServiceAccountCredentials(sensitiveConfigWith("- a\n- b"))

		assert.ErrorContains(t, err, "failed to parse sensitive config")
	})

	t.Run("fails on invalid yaml", func(t *testing.T) {
		_, err := ServiceAccountCredentials(sensitiveConfigWith("sa-postgresql: [unclosed"))

		assert.ErrorContains(t, err, "failed to parse sensitive config")
	})
}
