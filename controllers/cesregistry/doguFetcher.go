package cesregistry

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type doguFetcher struct {
	client             client.Client
	doguLocalRegistry  registry.DoguRegistry
	doguRemoteRegistry cesremote.Registry
}

// NewDoguFetcher creates a new dogu fetcher that provides descriptors for dogus.
func NewDoguFetcher(client client.Client, doguLocalRegistry registry.DoguRegistry, doguRemoteRegistry cesremote.Registry) *doguFetcher {
	return &doguFetcher{client: client, doguLocalRegistry: doguLocalRegistry, doguRemoteRegistry: doguRemoteRegistry}
}

func (df *doguFetcher) FetchInstalled(doguName string) (installedDogu *core.Dogu, err error) {
	installedDogu, err = df.getLocalDogu(doguName)
	if err != nil {
		return nil, fmt.Errorf("failed to get local dogu descriptor for %s: %w", doguName, err)
	}

	return replaceK8sIncompatibleDoguDependencies(installedDogu), nil
}

func (df *doguFetcher) FetchWithResource(ctx context.Context, doguResource *k8sv1.Dogu) (*core.Dogu, *k8sv1.DevelopmentDoguMap, error) {
	developmentDoguMap, err := df.getDevelopmentDoguMap(ctx, doguResource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get development dogu map: %w", err)
	}

	if developmentDoguMap == nil {
		log.FromContext(ctx).Info("Fetching dogu from remote dogu registry...")
		dogu, err := df.getDoguFromRemoteRegistry(doguResource)

		patchedDogu := replaceK8sIncompatibleDoguDependencies(dogu)
		return patchedDogu, nil, err
	}

	log.FromContext(ctx).Info("Fetching dogu from development dogu map...")
	dogu, err := df.getFromDevelopmentDoguMap(developmentDoguMap)

	patchedDogu := replaceK8sIncompatibleDoguDependencies(dogu)
	return patchedDogu, developmentDoguMap, err
}

func (df *doguFetcher) getLocalDogu(fromDoguName string) (*core.Dogu, error) {
	return df.doguLocalRegistry.Get(fromDoguName)
}

func (df *doguFetcher) getDevelopmentDoguMap(ctx context.Context, doguResource *k8sv1.Dogu) (*k8sv1.DevelopmentDoguMap, error) {
	configMap := &corev1.ConfigMap{}
	err := df.client.Get(ctx, doguResource.GetDevelopmentDoguMapKey(), configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get development dogu map for dogu %s: %w", doguResource.Name, err)
		}
	} else {
		doguDevMap := k8sv1.DevelopmentDoguMap(*configMap)
		return &doguDevMap, nil
	}
}

func (df *doguFetcher) getFromDevelopmentDoguMap(doguConfigMap *k8sv1.DevelopmentDoguMap) (*core.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	dogu := &core.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom dogu descriptor: %w", err)
	}

	return dogu, nil
}

func (df *doguFetcher) getDoguFromRemoteRegistry(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	dogu, err := df.doguRemoteRegistry.GetVersion(doguResource.Spec.Name, doguResource.Spec.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from remote dogu registry: %w", err)
	}

	return dogu, nil
}

func replaceK8sIncompatibleDoguDependencies(dogu *core.Dogu) *core.Dogu {
	dogu.Dependencies = patchDependencies(dogu.Dependencies)
	dogu.OptionalDependencies = patchDependencies(dogu.OptionalDependencies)

	return dogu
}

func patchDependencies(deps []core.Dependency) []core.Dependency {
	patchedDependencies := make([]core.Dependency, 0)

	for _, doguDep := range deps {
		name := doguDep.Name
		if name == "registrator" {
			continue
		}

		if name == "nginx" {
			ingress := core.Dependency{
				Name: "nginx-ingress",
				Type: core.DependencyTypeDogu,
			}
			staticNginx := core.Dependency{
				Name: "nginx-static",
				Type: core.DependencyTypeDogu,
			}
			patchedDependencies = append(patchedDependencies, ingress)
			patchedDependencies = append(patchedDependencies, staticNginx)

			continue
		}

		patchedDependencies = append(patchedDependencies, doguDep)
	}
	return patchedDependencies
}
