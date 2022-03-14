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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdDoguRegistrator is responsible for register dogus in the cluster
type EtcdDoguRegistrator struct {
	client.Client
}

// NewEtcdDoguRegistrator creates a new instance
func NewEtcdDoguRegistrator(client client.Client) *EtcdDoguRegistrator {
	return &EtcdDoguRegistrator{
		Client: client,
	}
}

// RegisterDogu register a dogu in a cluster. It generate key pairs and configure the dogu registry
func (e *EtcdDoguRegistrator) RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	keyPair, err := e.createKeypair()
	if err != nil {
		return fmt.Errorf("failed to create keypair: %w", err)
	}

	registry, err := cesregistry.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{"http://etcd.ecosystem.svc.cluster.local:4001"},
	})
	if err != nil {
		return fmt.Errorf("failed to get new registry: %w", err)
	}

	doguRegistry := registry.DoguRegistry()
	err = doguRegistry.Register(dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu "+dogu.GetSimpleName()+": %w", err)
	}
	err = doguRegistry.Enable(dogu)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}

	err = e.writePublicKey(keyPair.Public(), registry, dogu)
	if err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	err = e.writePrivateKey(keyPair.Private(), doguResource, ctx)
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

func (e *EtcdDoguRegistrator) createKeypair() (*keys.KeyPair, error) {
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

func (e *EtcdDoguRegistrator) writePrivateKey(privateKey *keys.PrivateKey, doguResource *k8sv1.Dogu, ctx context.Context) error {
	secretString, err := privateKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get bytes from private key: %w", err)
	}

	secret := &corev1.Secret{ObjectMeta: v1.ObjectMeta{Name: doguResource.GetPrivateVolumeName(), Namespace: doguResource.Namespace}}
	_, err = ctrl.CreateOrUpdate(ctx, e.Client, secret, func() error {
		secret.ObjectMeta.Labels = map[string]string{"app": cesLabel, "dogu": doguResource.Name}
		secret.StringData = map[string]string{"private.pem": secretString}
		return ctrl.SetControllerReference(doguResource, secret, e.Scheme())
	})

	if err != nil {
		return fmt.Errorf("failed to create dogu secret: %w", err)
	}

	return nil
}

func (e *EtcdDoguRegistrator) writePublicKey(publicKey *keys.PublicKey, registry cesregistry.Registry, dogu *core.Dogu) error {
	public, err := publicKey.AsString()
	if err != nil {
		return fmt.Errorf("failed to get public key as string: %w", err)
	}
	err = registry.DoguConfig(dogu.Name).Set(cesregistry.KeyDoguPublicKey, public)
	if err != nil {
		return fmt.Errorf("failed to write to registry: %w", err)
	}
	return nil
}
