package resource

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type doguSecretWriter struct {
	Client   client.Client
	Registry *registry.Registry
}

func (drr *doguSecretWriter) WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error {
	secretsMap, err := drr.getDoguSecrets(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to get secret for dogu [%s]: %w", doguResource.Name, err)
	}

	if secretsMap == nil {
		return nil
	}

	err = drr.writeSecretsToRegistry(secretsMap)
	if err != nil {
		return fmt.Errorf("failed to write secrets for dogu [%s]: %w", doguResource.Name, err)
	}

	return nil
}

func (drr *doguSecretWriter) writeSecretsToRegistry(secretsMap *corev1.ConfigMap) error {
	for etcdKey, secretValue := range secretsMap.Data {

	}
}

func (drr *doguSecretWriter) enryptAndWriteToRegistry(ctx context.Context, etcdKey string, value string) error {
	//casPublicKeyString, err := doguConfig.Get("public.pem")
	//if err == nil {
	//	keyProvider, err := keys.NewKeyProviderFromContext(cesappCtx)
	//	if err != nil {
	//		return errors.Wrap(err, "could not create key provider")
	//	}
	//
	//	casPublicKey, err := keyProvider.ReadPublicKeyFromString(casPublicKeyString)
	//	if err != nil {
	//		return errors.Wrap(err, "could not get public key from public.pem")
	//	}
	//
	//	passwordEnc, err := casPublicKey.Encrypt(conf.UserBackend.Password)
	//	if err != nil {
	//		return errors.Wrap(err, "could not encrypt password")
	//	}
	//
	//	err = doguConfig.Set("ldap/password", passwordEnc)
	//	if err != nil {
	//		return errors.Wrap(err, "could not set password")
	//	}
	//} else {
	//	// maybe cas is not installed, continue execution
	//	log.Warning("error while trying to get public.pem from cas: " + err.Error())
	//}
}

func (drr *doguSecretWriter) getDoguSecrets(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := drr.Client.Get(ctx, doguResource.GetSecretObjectKey(), configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get dogu secrets: %w", err)
		}
	} else {
		return configMap, err
	}
}
