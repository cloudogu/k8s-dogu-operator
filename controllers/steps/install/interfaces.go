package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

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
	CreateDoguService(doguResource *v2.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile) (*coreV1.Service, error)
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

type localDoguDescriptorRepository interface {
	cescommons.LocalDoguDescriptorRepository
	SetOwnerReference(ctx context.Context, dName cescommons.SimpleName, owners []metav1.OwnerReference) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

type conditionUpdater interface {
	UpdateCondition(ctx context.Context, doguResource *v2.Dogu, condition metav1.Condition) error
	UpdateConditions(ctx context.Context, doguResource *v2.Dogu, conditions []metav1.Condition) error
}
