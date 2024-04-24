package cesregistry

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/keys"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/localregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// CesDoguRegistrator is responsible for register dogus in the cluster
type CesDoguRegistrator struct {
	client            client.Client
	registry          cesregistry.Registry
	localDoguRegistry localregistry.LocalDoguRegistry
	secretGenerator   cloudogu.SecretResourceGenerator
}

// NewCESDoguRegistrator creates a new instance of the dogu registrator. It registers dogus in the dogu registry and
// generates keypairs
func NewCESDoguRegistrator(
	client client.Client,
	localDoguRegistry localregistry.LocalDoguRegistry,
	registry cesregistry.Registry,
	secretGenerator cloudogu.SecretResourceGenerator,
) *CesDoguRegistrator {
	return &CesDoguRegistrator{
		client:            client,
		registry:          registry,
		localDoguRegistry: localDoguRegistry,
		secretGenerator:   secretGenerator,
	}
}

// RegisterNewDogu registers a completely new dogu in a cluster. Use RegisterDoguVersion() for upgrades of an existing
// dogu.
func (c *CesDoguRegistrator) RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	enabled, err := c.localDoguRegistry.IsEnabled(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is already installed and enabled: %w", err)
	}

	if enabled {
		logger.Info("Skipping dogu registration because it is already installed and enabled in the dogu registry")
		return nil
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	err = c.registerNewKeys(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register keys: %w", err)
	}

	return c.enableDoguInRegistry(nil, dogu)
}

// RegisterDoguVersion registers an upgrade of an existing dogu in a cluster. Use RegisterNewDogu() to complete new
// dogu installations.
func (c *CesDoguRegistrator) RegisterDoguVersion(ctx context.Context, dogu *core.Dogu) error {
	enabled, err := c.localDoguRegistry.IsEnabled(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is already installed and enabled: %w", err)
	}

	if !enabled {
		return errors.New("could not register dogu version: previous version not found")
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	return c.enableDoguInRegistry(ctx, dogu)
}

// UnregisterDogu deletes a dogu from the dogu registry
func (c *CesDoguRegistrator) UnregisterDogu(ctx context.Context, dogu string) error {
	err := c.localDoguRegistry.UnregisterAllVersions(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to unregister dogu %s: %w", dogu, err)
	}

	return nil
}

func (c *CesDoguRegistrator) enableDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.localDoguRegistry.Enable(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}
	return nil
}

func (c *CesDoguRegistrator) registerDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.localDoguRegistry.Register(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu %s: %w", dogu.GetSimpleName(), err)
	}
	return nil
}

// registerNewKeys creates a new key pair and registers it with the dogu in the registry. If the private key exists it
// will be used to generate a new public key.
func (c *CesDoguRegistrator) registerNewKeys(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	keyProvider, err := resource.GetKeyProvider(c.registry)
	if err != nil {
		return err
	}

	secret, err := doguResource.GetPrivateKeySecret(ctx, c.client)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if apierrors.IsNotFound(err) {
		return c.createKeyPair(ctx, doguResource, dogu, keyProvider)
	}

	return c.recreatePubKey(secret, keyProvider, dogu)
}

func (c *CesDoguRegistrator) recreatePubKey(secret *corev1.Secret, keyProvider *keys.KeyProvider, dogu *core.Dogu) error {
	existingPrivateKey := secret.Data["private.pem"]
	keyPair, err := keyProvider.FromPrivateKey(existingPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create keypair from existing private key: %w", err)
	}

	err = c.writePublicKey(keyPair.Public(), dogu)
	if err != nil {
		return fmt.Errorf("failed to write public key from existing private key: %w", err)
	}

	return nil
}

func (c *CesDoguRegistrator) createKeyPair(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, keyProvider *keys.KeyProvider) error {
	keyPair, err := keyProvider.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	return c.writeKeyPair(ctx, doguResource, dogu, keyPair)
}

func (c *CesDoguRegistrator) writeKeyPair(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, keyPair *keys.KeyPair) error {
	err := c.writePrivateKey(ctx, keyPair.Private(), doguResource)
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	err = c.writePublicKey(keyPair.Public(), dogu)
	if err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	return nil
}

func (c *CesDoguRegistrator) writePrivateKey(ctx context.Context, privateKey *keys.PrivateKey, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	secretString, err := privateKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get private key as string: %w", err)
	}

	secret, err := c.secretGenerator.CreateDoguSecret(doguResource, map[string]string{"private.pem": secretString})
	if err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}

	err = c.client.Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	logger.Info(fmt.Sprintf("Secret %s/%s has been : %s", secret.Namespace, secret.Name, controllerutil.OperationResultCreated))

	return nil
}

func (c *CesDoguRegistrator) writePublicKey(publicKey *keys.PublicKey, dogu *core.Dogu) error {
	public, err := publicKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get public key as string: %w", err)
	}

	err = c.registry.DoguConfig(dogu.GetSimpleName()).Set(cesregistry.KeyDoguPublicKey, public)
	if err != nil {
		return fmt.Errorf("failed to write to registry: %w", err)
	}

	return nil
}
