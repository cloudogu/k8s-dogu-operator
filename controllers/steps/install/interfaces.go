package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	apps "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CreateDoguConfigStep interface {
	steps.Step
}

type CreateSensitiveDoguConfigStep interface {
	steps.Step
}

type DoguConfigOwnerReferenceStep interface {
	steps.Step
}

type SensitiveDoguConfigOwnerReferenceStep interface {
	steps.Step
}

type LocalDoguDescriptorOwnerReferenceStep interface {
	steps.Step
}

// premisesChecker includes functionality to check if the premises for an upgrade are met.
//
//nolint:unused
//goland:noinspection GoUnusedType
type premisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
}

// resourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type resourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *v2.Dogu) (*cesappcore.Dogu, *v2.DevelopmentDoguMap, error)
}

type securityValidator interface {
	ValidateSecurity(doguDescriptor *cesappcore.Dogu, doguResource *v2.Dogu) error
}

type doguAdditionalMountsValidator interface {
	ValidateAdditionalMounts(ctx context.Context, doguDescriptor *cesappcore.Dogu, doguResource *v2.Dogu) error
}

type doguConfigRepository interface {
	Get(ctx context.Context, name cescommons.SimpleName) (config.DoguConfig, error)
	Create(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Delete(ctx context.Context, name cescommons.SimpleName) error
	Watch(ctx context.Context, dName cescommons.SimpleName, filters ...config.WatchFilter) (<-chan repository.DoguConfigWatchResult, error)
	SetOwnerReference(ctx context.Context, dName cescommons.SimpleName, owners []metav1.OwnerReference) error
}

type netPolUpserter interface {
	UpsertDoguNetworkPolicies(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
}

type serviceGenerator interface {
	resource.DoguResourceGenerator
}

// imageRegistry abstracts the use of a container registry and includes functionality to pull container images.
type imageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type serviceInterface interface {
	v1.ServiceInterface
}

// doguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type doguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

type ConditionUpdater interface {
	UpdateCondition(ctx context.Context, doguResource *v2.Dogu, condition metav1.Condition) error
	UpdateConditions(ctx context.Context, doguResource *v2.Dogu, conditions []metav1.Condition) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecoSystemV2Interface interface {
	doguClient.EcoSystemV2Interface
}

type k8sClient interface {
	client.Client
}

type resourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except deploymentPatch which may be nil.
	// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
	UpsertDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu, deploymentPatch func(*apps.Deployment)) (*apps.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu, image *imagev1.ConfigFile) (*coreV1.Service, error)
	// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) (*coreV1.PersistentVolumeClaim, error)
	UpsertDoguNetworkPolicies(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
}

type ownerReferenceSetter interface {
	SetOwnerReference(ctx context.Context, dName cescommons.SimpleName, owners []metav1.OwnerReference) error
}

type deploymentAvailabilityChecker interface {
	IsAvailable(deployment *apps.Deployment) bool
}

type doguHealthStatusUpdater interface {
	UpdateStatus(ctx context.Context, doguName types.NamespacedName, available bool) error
	UpdateHealthConfigMap(ctx context.Context, deployment *apps.Deployment, doguJson *cesappcore.Dogu) error
}

type serviceAccountCreator interface {
	CreateAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type clientSet interface {
	kubernetes.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	v1.CoreV1Interface
}

type dependencyValidator interface {
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type persistentVolumeClaimInterface interface {
	v1.PersistentVolumeClaimInterface
}

type execPodFactory interface {
	CreateOrUpdate(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
	Exists(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) bool
	CheckReady(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
}

type eventRecorder interface {
	record.EventRecorder
}

type fileExtractor interface {
	// ExtractK8sResourcesFromExecPod copies files from an exec pod into map of strings.
	ExtractK8sResourcesFromExecPod(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *v2.Dogu) error
}
