package migration

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	migrationType = "migration"
	doguKind      = "Dogu"

	sensitiveConfigDataKey = "config.yaml"
	serviceAccountPrefix   = "sa-"
	credentialKeySeparator = "_"
	configPathSeparator    = "."
)

// MigrationSecret returns the migration secret of the dogu or nil, if it does not exist.
func MigrationSecret(ctx context.Context, reader client.Reader, dogu *v3beta1.Dogu) (*corev1.Secret, error) {
	return getSecret(ctx, reader, migrationResourceKey(dogu))
}

// SensitiveConfig returns the v2 sensitive config secret of the dogu or nil, if it does not exist.
// A secret with the same name but without the sensitive config type label is no sensitive config.
func SensitiveConfig(ctx context.Context, reader client.Reader, dogu *v3beta1.Dogu) (*corev1.Secret, error) {
	secret, err := getSecret(ctx, reader, sensitiveConfigKey(dogu))
	if err != nil {
		return nil, fmt.Errorf("failed to get sensitive config secret %s: %w", sensitiveConfigKey(dogu), err)
	}

	if secret == nil || secret.Labels[configTypeLabel] != sensitiveConfigType {
		return nil, nil
	}

	return secret, nil
}

func getSecret(ctx context.Context, reader client.Reader, key client.ObjectKey) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := reader.Get(ctx, key, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	return secret, nil
}

// ServiceAccountCredentials extracts the service account credentials from the v2 sensitive config.
// All other entries of the sensitive config are ignored.
func ServiceAccountCredentials(sensitiveConfig *corev1.Secret) (map[string][]byte, error) {
	var config map[string]any
	if pErr := yaml.Unmarshal(sensitiveConfig.Data[sensitiveConfigDataKey], &config); pErr != nil {
		return nil, fmt.Errorf("failed to parse sensitive config: %w", pErr)
	}

	credentials := make(map[string][]byte)

	// configPaths maps every credential key to the config path it was taken from to detect collisions.
	configPaths := map[string]string{}
	for section, entries := range config {
		producer, isServiceAccount := strings.CutPrefix(section, serviceAccountPrefix)
		if !isServiceAccount {
			continue
		}

		// v2 service accounts are flat string maps, anything else is no credential.
		entryMap, _ := entries.(map[string]any)
		for key, value := range entryMap {
			credential, ok := value.(string)
			if !ok {
				continue
			}

			configPath := section + configPathSeparator + key
			credentialKey := producer + credentialKeySeparator + key
			if other, exists := configPaths[credentialKey]; exists {
				paths := []string{other, configPath}
				slices.Sort(paths)

				return nil, fmt.Errorf("sensitive config paths %q and %q both result in the migration secret key %q", paths[0], paths[1], credentialKey)
			}

			configPaths[credentialKey] = configPath
			credentials[credentialKey] = []byte(credential)
		}
	}

	return credentials, nil
}

// NewMigrationSecret builds the apply configuration of the dogu's migration secret holding the given data.
func NewMigrationSecret(dogu *v3beta1.Dogu, data map[string][]byte) *corev1ac.SecretApplyConfiguration {
	key := migrationResourceKey(dogu)
	return corev1ac.Secret(key.Name, key.Namespace).
		WithLabels(map[string]string{
			resource.CloudoguLabelKeyApp: resource.LabelValueCes,
			v3beta1.DoguLabelName:        dogu.Spec.Name,
			configTypeLabel:              migrationType,
		}).
		WithType(corev1.SecretTypeOpaque).
		WithData(data)
}
