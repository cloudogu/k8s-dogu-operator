package resource

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/registry"
	configkeys "github.com/cloudogu/cesapp/v5/config"
	"github.com/cloudogu/cesapp/v5/keys"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type doguSecretWriter struct {
	Client   client.Client
	Registry registry.Registry
}

// NewDoguSecretsWriter creates a new instance of doguSecretWriter to save key value pairs from secrets to the dogu config
func NewDoguSecretsWriter(client client.Client, registry registry.Registry) *doguSecretWriter {
	return &doguSecretWriter{
		Client:   client,
		Registry: registry,
	}
}

// WriteDoguSecretsToRegistry gets the dogu secret and writes the values in the dogu config. If no secret is found the
// method returns no error
func (drr *doguSecretWriter) WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error {
	secret, err := drr.getDoguSetupSecret(ctx, doguResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get dogu secrets: %w", err)
	}

	err = drr.writeSecretsToRegistry(secret, doguResource.Name)
	if err != nil {
		return fmt.Errorf("failed to write secrets for dogu [%s]: %w", doguResource.Name, err)
	}

	err = drr.Client.Delete(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to delete %s setup secret: %w", doguResource.Name, err)
	}

	return nil
}

func (drr *doguSecretWriter) writeSecretsToRegistry(secret *corev1.Secret, dogu string) error {
	doguPublicKey, err := GetPublicKey(drr.Registry, dogu)
	if err != nil {
		return err
	}
	doguConfig := drr.Registry.DoguConfig(dogu)

	for etcdKey, secretValue := range secret.Data {
		// setup writes keys from secret with "." instead of "/" because they are not allowed in keys for Kubernetes secrets
		key := strings.ReplaceAll(etcdKey, ".", "/")
		encryptedValue, err := doguPublicKey.Encrypt(string(secretValue))
		if err != nil {
			return fmt.Errorf("failed to encrypt value for key %s: %w", key, err)
		}

		err = doguConfig.Set(key, encryptedValue)
		if err != nil {
			return fmt.Errorf("failed to write key %s to registry: %w", key, err)
		}
	}

	return nil
}

func (drr *doguSecretWriter) getDoguSetupSecret(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := drr.Client.Get(ctx, doguResource.GetSecretObjectKey(), secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// GetPublicKey returns the public key from the dogu configuration.
func GetPublicKey(registry registry.Registry, dogu string) (*keys.PublicKey, error) {
	keyProvider, err := GetKeyProvider(registry)
	if err != nil {
		return nil, err
	}

	doguPublicKeyString, err := registry.DoguConfig(dogu).Get("public.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to get public key for dogu %s: %w", dogu, err)
	}

	doguPublicKey, err := keyProvider.ReadPublicKeyFromString(doguPublicKeyString)
	if err != nil {
		return nil, fmt.Errorf("could not get public key from public.pem: %w", err)
	}

	return doguPublicKey, nil
}

// GetKeyProvider returns the key provider from the global configuration.
func GetKeyProvider(registry registry.Registry) (*keys.KeyProvider, error) {
	keyProviderStr, err := registry.GlobalConfig().Get("key_provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get key provider: %w", err)
	}

	keyProvider, err := keys.NewKeyProvider(configkeys.Keys{Type: keyProviderStr})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyprovider: %w", err)
	}

	return keyProvider, nil
}
