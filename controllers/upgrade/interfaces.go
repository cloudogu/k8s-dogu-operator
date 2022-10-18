package upgrade

import (
	"context"

	"github.com/cloudogu/cesapp-lib/core"

	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type imageRegistry interface {
	// PullImageConfig pulls a given container image by name.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into a map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, execpod util.ExecPod) (map[string]string, error)
}

type serviceAccountCreator interface {
	// CreateAll creates K8s services accounts for a dogu
	CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error
}

type doguRegistrator interface {
	// RegisterDoguVersion registers a certain dogu in a CES instance.
	RegisterDoguVersion(dogu *core.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}

type resourceUpserter interface {
	// ApplyDoguResource generates K8s resources from a given dogu and creates/updates them in the cluster.
	ApplyDoguResource(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error
}

type execPodFactory interface {
	NewExecPod(doguResource *k8sv1.Dogu, dogu *core.Dogu) (util.ExecPod, error)
}
