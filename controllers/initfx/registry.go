package initfx

import (
	"fmt"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	reg "github.com/cloudogu/k8s-registry-lib/dogu"
	remotedogudescriptor "github.com/cloudogu/remote-dogu-descriptor-lib/repository"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDoguVersionRegistry(cmInterface v1.ConfigMapInterface) dogu.VersionRegistry {
	return reg.NewDoguVersionRegistry(cmInterface)
}

type LocalDoguDescriptorRepository interface {
	dogu.LocalDoguDescriptorRepository
	OwnerReferenceSetter
}

func NewLocalDoguDescriptorRepository(cmInterface v1.ConfigMapInterface) LocalDoguDescriptorRepository {
	return reg.NewLocalDoguDescriptorRepository(cmInterface)
}

func NewLocalDoguFetcher(registry dogu.VersionRegistry, repository dogu.LocalDoguDescriptorRepository) cesregistry.LocalDoguFetcher {
	return cesregistry.NewLocalDoguFetcher(registry, repository)
}

func NewResourceDoguFetcher(client client.Client, operatorConfig *config.OperatorConfig) (cesregistry.ResourceDoguFetcher, error) {
	remoteConfig, err := operatorConfig.GetRemoteConfiguration()
	if err != nil {
		return nil, err
	}

	doguRemoteRepository, err := remotedogudescriptor.NewRemoteDoguDescriptorRepository(remoteConfig, operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu repository: %w", err)
	}

	resourceDoguFetcher := cesregistry.NewResourceDoguFetcher(client, doguRemoteRepository)
	return resourceDoguFetcher, nil
}
