package main

import (
	"github.com/cloudogu/ces-commons-lib/dogu"
	authRegClientV1 "github.com/cloudogu/k8s-auth-registration-lib/client/typed/api/v1"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v3/client"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	warpmenuentryv1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

//nolint:unused
//goland:noinspection GoUnusedType
type kubernetesInterface interface {
	kubernetes.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	corev1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type appsV1Interface interface {
	appsv1.AppsV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type deploymentInterface interface {
	appsv1.DeploymentInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type configMapInterface interface {
	corev1.ConfigMapInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type secretInterface interface {
	corev1.SecretInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type serviceInterface interface {
	corev1.ServiceInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type pvcInterface interface {
	corev1.PersistentVolumeClaimInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type podInterface interface {
	corev1.PodInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type restInterface interface {
	rest.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguClientsetInterface interface {
	doguClient.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguV2Interface interface {
	doguv2.DoguV2Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguRestartInterface interface {
	doguv2.DoguRestartInterface
}

// mocks for integration tests

//nolint:unused
//goland:noinspection GoUnusedType
type commandExecutor interface {
	exec.CommandExecutor
}

//nolint:unused
//goland:noinspection GoUnusedType
type remoteDoguDescriptorRepository interface {
	dogu.RemoteDoguDescriptorRepository
}

//nolint:unused
//goland:noinspection GoUnusedType
type imageRegistry interface {
	imageregistry.ImageRegistry
}

//nolint:unused
//goland:noinspection GoUnusedType
type authRegistrationClient interface {
	authRegClientV1.ApiV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type authRegistrationInterface interface {
	authRegClientV1.AuthRegistrationInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type warpMenuEntryClient interface {
	warpmenuentryv1.ApiV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type warpMenuEntryInterface interface {
	warpmenuentryv1.WarpMenuEntryInterface
}
