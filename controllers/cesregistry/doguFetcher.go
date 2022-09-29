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

// localDoguFetcher abstracts the access to dogu structs from the local dogu registry.
type localDoguFetcher struct {
	doguLocalRegistry registry.DoguRegistry
}

// localDoguFetcher abstracts the access to dogu structs from either the remote dogu registry or from a local DevelopmentDoguMap.
type resourceDoguFetcher struct {
	client             client.Client
	doguRemoteRegistry cesremote.Registry
}

// NewLocalDoguFetcher creates a new dogu fetcher that provides descriptors for dogus.
func NewLocalDoguFetcher(doguLocalRegistry registry.DoguRegistry) *localDoguFetcher {
	return &localDoguFetcher{doguLocalRegistry: doguLocalRegistry}
}

// NewResourceDoguFetcher creates a new dogu fetcher that provides descriptors for dogus.
func NewResourceDoguFetcher(client client.Client, doguRemoteRegistry cesremote.Registry) *resourceDoguFetcher {
	return &resourceDoguFetcher{client: client, doguRemoteRegistry: doguRemoteRegistry}
}

// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
// otherwise might be incompatible with K8s CES).
func (df *localDoguFetcher) FetchInstalled(doguName string) (installedDogu *core.Dogu, err error) {
	installedDogu, err = df.getLocalDogu(doguName)
	if err != nil {
		return nil, fmt.Errorf("failed to get local dogu descriptor for %s: %w", doguName, err)
	}

	return replaceK8sIncompatibleDoguDependencies(installedDogu), nil
}

func (df *localDoguFetcher) getLocalDogu(fromDoguName string) (*core.Dogu, error) {
	return df.doguLocalRegistry.Get(fromDoguName)
}

// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
func (rdf *resourceDoguFetcher) FetchWithResource(ctx context.Context, doguResource *k8sv1.Dogu) (*core.Dogu, *k8sv1.DevelopmentDoguMap, error) {
	developmentDoguMap, err := rdf.getDevelopmentDoguMap(ctx, doguResource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get development dogu map: %w", err)
	}

	if developmentDoguMap == nil {
		log.FromContext(ctx).Info("Fetching dogu from remote dogu registry...")
		dogu, err := rdf.getDoguFromRemoteRegistry(doguResource)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get dogu from remote or cache: %w", err)
		}

		patchedDogu := replaceK8sIncompatibleDoguDependencies(dogu)
		return patchedDogu, nil, err
	}

	log.FromContext(ctx).Info("Fetching dogu from development dogu map...")
	dogu, err := rdf.getFromDevelopmentDoguMap(developmentDoguMap)

	patchedDogu := replaceK8sIncompatibleDoguDependencies(dogu)
	return patchedDogu, developmentDoguMap, err
}

func (rdf *resourceDoguFetcher) getDevelopmentDoguMap(ctx context.Context, doguResource *k8sv1.Dogu) (*k8sv1.DevelopmentDoguMap, error) {
	configMap := &corev1.ConfigMap{}
	err := rdf.client.Get(ctx, doguResource.GetDevelopmentDoguMapKey(), configMap)
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

func (rdf *resourceDoguFetcher) getFromDevelopmentDoguMap(doguConfigMap *k8sv1.DevelopmentDoguMap) (*core.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	dogu := &core.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom dogu descriptor: %w", err)
	}

	return dogu, nil
}

func (rdf *resourceDoguFetcher) getDoguFromRemoteRegistry(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	dogu, err := rdf.doguRemoteRegistry.GetVersion(doguResource.Spec.Name, doguResource.Spec.Version)
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
