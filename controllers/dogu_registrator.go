package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/cesapp/v5/keys"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CesDoguRegistrator is responsible for register dogus in the cluster
type CesDoguRegistrator struct {
	client          client.Client
	registry        cesregistry.Registry
	doguRegistry    cesregistry.DoguRegistry
	secretGenerator doguResourceGenerator
}

// newCESDoguRegistrator creates a new instance of the dogu registrator. It registers dogus in the dogu registry and
// generates keypairs
func newCESDoguRegistrator(client client.Client, registry cesregistry.Registry, secretGenerator doguResourceGenerator) *CesDoguRegistrator {
	return &CesDoguRegistrator{
		client:          client,
		registry:        registry,
		doguRegistry:    registry.DoguRegistry(),
		secretGenerator: secretGenerator,
	}
}

// RegisterDogu registers a dogu in a cluster. It generates key pairs and configures the dogu registry
func (c *CesDoguRegistrator) RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	enabled, err := c.doguRegistry.IsEnabled(dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is already installed and enabled: %w", err)
	}

	if enabled {
		logger.Info("Skipping dogu registration because it is already installed and enabled in the dogu registry")
		return nil
	}

	err = c.addDoguToRegistry(dogu)
	if err != nil {
		return fmt.Errorf("failed to add dogu to registry: %w", err)
	}

	err = c.registerKeys(ctx, dogu, doguResource)
	if err != nil {
		return fmt.Errorf("failed to register keys: %w", err)
	}

	return nil
}

// UnregisterDogu deletes a dogu from the dogu registry
func (c *CesDoguRegistrator) UnregisterDogu(dogu string) error {
	err := c.registry.DoguConfig(dogu).RemoveAll()
	if err != nil && !cesregistry.IsKeyNotFoundError(err) {
		return fmt.Errorf("failed to remove dogu config: %w", err)
	}

	err = c.doguRegistry.Unregister(dogu)
	if err != nil && !cesregistry.IsKeyNotFoundError(err) {
		return fmt.Errorf("failed to unregister dogu %s: %w", dogu, err)
	}

	return nil
}

func (c *CesDoguRegistrator) addDoguToRegistry(dogu *core.Dogu) error {
	err := c.doguRegistry.Register(dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu %s: %w", dogu.GetSimpleName(), err)
	}

	err = c.doguRegistry.Enable(dogu)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}

	return nil
}

func (c *CesDoguRegistrator) registerKeys(ctx context.Context, dogu *core.Dogu, doguResource *k8sv1.Dogu) error {
	keyPair, err := c.createKeypair()
	if err != nil {
		return fmt.Errorf("failed to create keypair: %w", err)
	}

	err = c.writePublicKey(keyPair.Public(), dogu)
	if err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	err = c.writePrivateKey(ctx, keyPair.Private(), doguResource)
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

func (c *CesDoguRegistrator) createKeypair() (*keys.KeyPair, error) {
	keyProvider, err := resource.GetKeyProvider(c.registry)
	if err != nil {
		return nil, err
	}

	keyPair, err := keyProvider.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return keyPair, nil
}

func (c *CesDoguRegistrator) writePrivateKey(ctx context.Context, privateKey *keys.PrivateKey, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	secretString, err := privateKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get private key as string: %w", err)
	}

	secret, err := c.secretGenerator.GetDoguSecret(doguResource, map[string]string{"private.pem": secretString})
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
