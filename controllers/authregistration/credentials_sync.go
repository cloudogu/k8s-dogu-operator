package authregistration

import (
	"context"
	"fmt"
	"strings"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	regLibErr "github.com/cloudogu/ces-commons-lib/errors"
	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	"github.com/cloudogu/k8s-registry-lib/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type sensitiveConfigCredentialsSyncer struct {
	secretClient      secretClient
	sensitiveDoguRepo sensitiveDoguConfigRepository
}

func (s *sensitiveConfigCredentialsSyncer) SyncCredentials(ctx context.Context, authReg *authRegApiV1.AuthRegistration, doguName string, serviceAccountType string) error {
	if authReg == nil {
		return nil
	}

	secretName := strings.TrimSpace(authReg.Status.ResolvedSecretRef)
	if secretName == "" {
		return fmt.Errorf("auth-registration %q has no resolved secretRef yet", authReg.Name)
	}

	secret, err := s.secretClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %q: %w", secretName, err)
	}
	if !hasCredentialData(secret) {
		return fmt.Errorf("auth-registration secret %q has no credentials yet", secretName)
	}

	sensitiveConfig, err := s.sensitiveDoguRepo.Get(ctx, cescommons.SimpleName(doguName))
	if err != nil {
		return fmt.Errorf("failed to get sensitive dogu config for %q: %w", doguName, err)
	}

	credentialsChanged := false
	for key, value := range secret.Data {
		path := fmt.Sprintf("/sa-%s/%s", serviceAccountType, key)
		newValue := string(value)
		existingValue, exists := sensitiveConfig.Get(config.Key(path))
		if exists && existingValue.String() == newValue {
			continue
		}

		updatedCfg, setErr := sensitiveConfig.Set(config.Key(path), config.Value(newValue))
		if setErr != nil {
			return fmt.Errorf("failed to set value for path %q: %w", path, setErr)
		}
		sensitiveConfig.Config = updatedCfg
		credentialsChanged = true
	}

	if !credentialsChanged {
		return nil
	}

	if err = s.writeSensitiveDoguConfig(ctx, &sensitiveConfig); err != nil {
		return fmt.Errorf("failed to write sensitive dogu config for %q: %w", doguName, err)
	}

	return nil
}

func (s *sensitiveConfigCredentialsSyncer) writeSensitiveDoguConfig(ctx context.Context, cfg *config.DoguConfig) error {
	update, err := s.sensitiveDoguRepo.Update(ctx, *cfg)
	if err != nil {
		if regLibErr.IsConflictError(err) {
			mergedCfg, mergeErr := s.sensitiveDoguRepo.SaveOrMerge(ctx, *cfg)
			if mergeErr != nil {
				return fmt.Errorf("unable to save and merge sensitive config for dogu %s after conflict error: %w", cfg.DoguName, mergeErr)
			}

			*cfg = mergedCfg
			return nil
		}

		return fmt.Errorf("unable to update sensitive config for dogu %s: %w", cfg.DoguName, err)
	}

	*cfg = update
	return nil
}

func hasCredentialData(secret *corev1.Secret) bool {
	for _, value := range secret.Data {
		if strings.TrimSpace(string(value)) != "" {
			return true
		}
	}

	return false
}
