package initfx

import (
	"fmt"
	"net/http"
	"path"
	"time"

	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

// chartDownloadTimeout caps a single chart-artifact download. Artifacts are small tarballs served
// by Flux's in-cluster source-controller, so 30s is generous.
const chartDownloadTimeout = 30 * time.Second

// NewChartHTTPClient provides the HTTP client used by the chart provider to download chart
// artifacts from the OCIRepository's resolved URL.
func NewChartHTTPClient() *http.Client {
	return &http.Client{Timeout: chartDownloadTimeout}
}

// NewChartCapabilities discovers the cluster's Helm capabilities (Kubernetes version and available
// API versions) once at startup, so local chart renders gate API versions the same way Flux will.
func NewChartCapabilities(clientSet kubernetes.Interface) (*chartutil.Capabilities, error) {
	discoveryClient := clientSet.Discovery()

	kubeVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to discover kubernetes server version: %w", err)
	}

	apiVersions, err := discoverVersionSet(discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to discover kubernetes api versions: %w", err)
	}

	return &chartutil.Capabilities{
		APIVersions: apiVersions,
		KubeVersion: chartutil.KubeVersion{
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		},
		HelmVersion: chartutil.DefaultCapabilities.HelmVersion,
	}, nil
}

// discoverVersionSet builds the set of available API versions from the cluster's discovery data.
//
// This is a faithful copy of Helm's action.GetVersionSet. We do NOT import
// helm.sh/helm/v3/pkg/action because that package bundles Helm's full install/upgrade/release-storage
// and registry machinery, so importing it to call this one function increases the surface of potential CVEs
// (kubectl, cli-runtime, kustomize, oras, and even SQL release-storage backends such as lib/pq).
// If Helm's discovery behavior ever changes, re-sync this function with action.GetVersionSet.
func discoverVersionSet(client discovery.ServerResourcesInterface) (chartutil.VersionSet, error) {
	groups, resources, err := client.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return chartutil.DefaultVersionSet, fmt.Errorf("could not get apiVersions from Kubernetes: %w", err)
	}

	// If no API versions are discovered, fall back to the default version set.
	if len(groups) == 0 && len(resources) == 0 {
		return chartutil.DefaultVersionSet, nil
	}

	versionMap := make(map[string]struct{})
	for _, g := range groups {
		for _, gv := range g.Versions {
			versionMap[gv.GroupVersion] = struct{}{}
		}
	}
	for _, resourceList := range resources {
		for _, resource := range resourceList.APIResources {
			id := path.Join(resourceList.GroupVersion, resource.Kind)
			if _, ok := versionMap[id]; !ok {
				versionMap[resourceList.GroupVersion] = struct{}{}
				versionMap[id] = struct{}{}
			}
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}

	return versions, nil
}
