package cesregistry

import (
	"context"
	"encoding/json"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/retry-lib/retry"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

// localDoguFetcher abstracts the access to dogu structs from the local dogu registry.
type localDoguFetcher struct {
	doguVersionRegistry doguVersionRegistry
	doguRepository      localDoguDescriptorRepository
}

// localDoguFetcher abstracts the access to dogu structs from either the remote dogu registry or from a local DevelopmentDoguMap.
type resourceDoguFetcher struct {
	client                   client.Client
	doguRemoteRepository     remoteDoguDescriptorRepository
	doguDescriptorMaxRetries int
}

var defaultMaxTries = 20

const doguDescriptorMaxRetriesEnv = "DOGU_DESCRIPTOR_MAX_RETRIES"

// NewLocalDoguFetcher creates a new dogu fetcher that provides descriptors for dogus.
func NewLocalDoguFetcher(doguVersionRegistry doguVersionRegistry, doguDescriptorRepo localDoguDescriptorRepository) *localDoguFetcher {
	return &localDoguFetcher{doguVersionRegistry: doguVersionRegistry, doguRepository: doguDescriptorRepo}
}

// NewResourceDoguFetcher creates a new dogu fetcher that provides descriptors for dogus.
func NewResourceDoguFetcher(client client.Client, doguRemoteRepository remoteDoguDescriptorRepository) *resourceDoguFetcher {
	maxRetriesString, found := os.LookupEnv(doguDescriptorMaxRetriesEnv)
	maxRetries, err := strconv.Atoi(maxRetriesString)
	if !found || err != nil {
		logrus.Warningf("failed to read %s environment variable, using default value of %d", doguDescriptorMaxRetriesEnv, defaultMaxTries)
		maxRetries = defaultMaxTries
	}
	return &resourceDoguFetcher{client: client, doguRemoteRepository: doguRemoteRepository, doguDescriptorMaxRetries: maxRetries}
}

// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
// otherwise might be incompatible with K8s CES).
func (df *localDoguFetcher) FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *core.Dogu, err error) {
	installedDogu, err = df.getLocalDogu(ctx, doguName)
	if err != nil {
		return nil, fmt.Errorf("failed to get local dogu descriptor for %s: %w", doguName, err)
	}

	return replaceK8sIncompatibleDoguDependencies(installedDogu), nil
}

func (df *localDoguFetcher) Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error) {
	enabled, _, err := checkDoguVersionEnabled(ctx, df.doguVersionRegistry, doguName)
	return enabled, err
}

func (df *localDoguFetcher) getLocalDogu(ctx context.Context, fromDoguName cescommons.SimpleName) (*core.Dogu, error) {
	current, err := df.doguVersionRegistry.GetCurrent(ctx, fromDoguName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version for dogu %s: %w", fromDoguName, err)
	}

	get, err := df.doguRepository.Get(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("failed to get current dogu %s: %w", fromDoguName, err)
	}

	return get, nil
}

// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
func (rdf *resourceDoguFetcher) FetchWithResource(ctx context.Context, doguResource *doguv2.Dogu) (*core.Dogu, *doguv2.DevelopmentDoguMap, error) {
	developmentDoguMap, err := rdf.getDevelopmentDoguMap(ctx, doguResource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get development dogu map: %w", err)
	}

	if developmentDoguMap == nil {
		log.FromContext(ctx).Info("Fetching dogu from remote dogu registry...")
		version, err := core.ParseVersion(doguResource.Spec.Version)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse version: %w", err)
		}
		qualifiedName, err := cescommons.QualifiedNameFromString(doguResource.Spec.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse namespace and name: %w", err)
		}
		qualifiedDoguVersion := cescommons.QualifiedVersion{
			Version: version,
			Name:    qualifiedName,
		}

		remoteDogu, err := rdf.getDoguFromRemoteRegistry(ctx, qualifiedDoguVersion)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get dogu from remote or cache: %w", err)
		}

		patchedDogu := replaceK8sIncompatibleDoguDependencies(remoteDogu)
		return patchedDogu, nil, err
	}

	log.FromContext(ctx).Info("Fetching dogu from development dogu map...")
	remoteDogu, err := rdf.getFromDevelopmentDoguMap(developmentDoguMap)

	patchedDogu := replaceK8sIncompatibleDoguDependencies(remoteDogu)
	return patchedDogu, developmentDoguMap, err
}

func (rdf *resourceDoguFetcher) getDevelopmentDoguMap(ctx context.Context, doguResource *doguv2.Dogu) (*doguv2.DevelopmentDoguMap, error) {
	configMap := &corev1.ConfigMap{}
	err := rdf.client.Get(ctx, doguResource.GetDevelopmentDoguMapKey(), configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get development dogu map for dogu %s: %w", doguResource.Name, err)
		}
	} else {
		doguDevMap := doguv2.DevelopmentDoguMap(*configMap)
		return &doguDevMap, nil
	}
}

func (rdf *resourceDoguFetcher) getFromDevelopmentDoguMap(doguConfigMap *doguv2.DevelopmentDoguMap) (*core.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	configMapDogu := &core.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), configMapDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom dogu descriptor: %w", err)
	}

	return configMapDogu, nil
}

func (rdf *resourceDoguFetcher) getDoguFromRemoteRegistry(context context.Context, version cescommons.QualifiedVersion) (*core.Dogu, error) {
	remoteDogu := &core.Dogu{}
	err := retry.OnError(rdf.doguDescriptorMaxRetries, cloudoguerrors.IsConnectionError, func() error {
		var err error
		remoteDogu, err = rdf.doguRemoteRepository.Get(context, version)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from remote dogu registry: %w", err)
	}

	return remoteDogu, nil
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
