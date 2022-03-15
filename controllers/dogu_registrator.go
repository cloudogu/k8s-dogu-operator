package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/keys"
	cesregistry "github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CESDoguRegistrator is responsible for register dogus in the cluster
type CESDoguRegistrator struct {
	client.Client
	registry     cesregistry.Registry
	doguRegistry cesregistry.DoguRegistry
}

// NewCESDoguRegistrator creates a new instance
func NewCESDoguRegistrator(client client.Client, registry cesregistry.Registry) *CESDoguRegistrator {
	return &CESDoguRegistrator{
		Client:       client,
		registry:     registry,
		doguRegistry: registry.DoguRegistry(),
	}
}

// RegisterDogu registers a dogu in a cluster. It generates key pairs and configures the dogu registry
func (c *CESDoguRegistrator) RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	err := c.doguRegistry.Register(dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu "+dogu.GetSimpleName()+": %w", err)
	}

	err = c.doguRegistry.Enable(dogu)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}

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

func (c *CESDoguRegistrator) createKeypair() (*keys.KeyPair, error) {
	keyProvider, err := keys.NewKeyProvider(core.Keys{Type: "pkcs1v15"})
	if err != nil {
		return nil, fmt.Errorf("failed to create key provider: %w", err)
	}

	keyPair, err := keyProvider.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return keyPair, nil
}

func (c *CESDoguRegistrator) writePrivateKey(ctx context.Context, privateKey *keys.PrivateKey, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	secretString, err := privateKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get private key as string: %w", err)
	}

	secret := &corev1.Secret{ObjectMeta: v1.ObjectMeta{
		Name:      doguResource.GetPrivateVolumeName(),
		Namespace: doguResource.Namespace,
		Labels:    map[string]string{"app": cesLabel, "dogu": doguResource.Name}},
		StringData: map[string]string{"private.pem": secretString}}
	err = c.Client.Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	logger.Info(fmt.Sprintf("Secret %s/%s has been : %s", secret.Namespace, secret.Name, controllerutil.OperationResultCreated))

	return nil
}

func (c *CESDoguRegistrator) writePublicKey(publicKey *keys.PublicKey, dogu *core.Dogu) error {
	public, err := publicKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get public key as string: %w", err)
	}
	err = c.registry.DoguConfig(dogu.Name).Set(cesregistry.KeyDoguPublicKey, public)
	if err != nil {
		return fmt.Errorf("failed to write to registry: %w", err)
	}
	return nil
}
