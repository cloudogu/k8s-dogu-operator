package controllers

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/go-logr/logr"
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
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

// doguSecretHandler is used to write potential secret from the setup.json registryConfigEncrypted
type doguSecretHandler interface {
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// imageRegistry is used to pull container images
type imageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// doguRegistrator is used to register dogus
type doguRegistrator interface {
	RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
	RegisterDoguVersion(dogu *cesappcore.Dogu) error
	UnregisterDogu(dogu string) error
}

// dependencyValidator is used to check if dogu dependencies are installed
type dependencyValidator interface {
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

// serviceAccountCreator is used to create service accounts for a given dogu
type serviceAccountCreator interface {
	CreateAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

// serviceAccountRemover is used to remove service accounts for a given dogu
type serviceAccountRemover interface {
	RemoveAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

// DoguSecretsHandler is used to write the encrypted secrets from the setup to the dogu config
type DoguSecretsHandler interface {
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
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
	// Check checks if dogu premises are met before a dogu upgrade
	Check(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

type upgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *cesappcore.Dogu) error
}
