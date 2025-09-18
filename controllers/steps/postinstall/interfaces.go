package postinstall

import (
	"context"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
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

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

// exportManager includes functionality to handle the export flag for dogus in the cluster.
type exportManager interface {
	// UpdateExportMode activates/deactivates the export mode for the dogu
	UpdateExportMode(ctx context.Context, doguResource *v2.Dogu) error
}

// supportManager includes functionality to handle the support flag for dogus in the cluster.
type supportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error)
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

type additionalMountManager interface {
	AdditionalMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error)
	UpdateAdditionalMounts(ctx context.Context, doguResource *v2.Dogu) error
}

type doguRestartManager interface {
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}

type k8sClient interface {
	client.Client
}

type podInterface interface {
	v1.PodInterface
}

type securityContextGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu, doguResource *v2.Dogu) (*corev1.PodSecurityContext, *corev1.SecurityContext)
}

type ingressAnnotator interface {
	AppendIngressAnnotationsToService(service *corev1.Service, additionalIngressAnnotations v2.IngressAnnotations) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type appV1Interface interface {
	appsv1client.AppsV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type clientSet interface {
	kubernetes.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type eventRecorder interface {
	record.EventRecorder
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	v1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type resourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except deploymentPatch which may be nil.
	// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
	UpsertDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu, deploymentPatch func(*apps.Deployment)) (*apps.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu, image *imagev1.ConfigFile) (*corev1.Service, error)
	// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) (*corev1.PersistentVolumeClaim, error)
	UpsertDoguNetworkPolicies(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguResourceGenerator interface {
	resource.DoguResourceGenerator
}

type deploymentManager interface {
	GetLastStartingTime(ctx context.Context, deploymentName string) (*time.Time, error)
}
