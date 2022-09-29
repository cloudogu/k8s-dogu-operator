package controllers

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
)

type installManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type upgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type deleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

type doguSecretHandler interface {
	// WriteDoguSecretsToRegistry is used to write potential secret from the setup.json registryConfigEncrypted to the
	// respective dogu configurations.
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type imageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type doguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(dogu string) error
}

type dependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type serviceAccountCreator interface {
	// CreateAll is used to create all necessary service accounts for the given dogu.
	CreateAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

type serviceAccountRemover interface {
	// RemoveAll is used to remove all existing service accounts for the given dogu.
	RemoveAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}

type resourceUpserter interface {
	// ApplyDoguResource generates K8s resources from a given dogu and creates/updates them in the cluster.
	ApplyDoguResource(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error
}

type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(doguName string) (installedDogu *cesappcore.Dogu, err error)
}

type resourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *k8sv1.Dogu) (*cesappcore.Dogu, *k8sv1.DevelopmentDoguMap, error)
}

type premisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

type upgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *cesappcore.Dogu) error
}
