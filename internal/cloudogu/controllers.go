package cloudogu

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/v1"
	coreV1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// InstallManager includes functionality to install dogus in the cluster.
type InstallManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *v1.Dogu) error
}

// UpgradeManager includes functionality to upgrade dogus in the cluster.
type UpgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *v1.Dogu) error
}

// DeleteManager includes functionality to delete dogus from the cluster.
type DeleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *v1.Dogu) error
}

// SupportManager includes functionality to handle the support flag for dogus in the cluster.
type SupportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v1.Dogu) (bool, error)
}

// VolumeManager includes functionality to edit volumes for dogus in the cluster.
type VolumeManager interface {
	// SetDoguDataVolumeSize sets the volume size for the given dogu.
	SetDoguDataVolumeSize(ctx context.Context, doguResource *v1.Dogu) error
}

// AdditionalIngressAnnotationsManager includes functionality to edit additional ingress annotations for dogus in the cluster.
type AdditionalIngressAnnotationsManager interface {
	// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
	SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v1.Dogu) error
}

// StartDoguManager includes functionality to start (stopped) dogus.
type StartDoguManager interface {
	// StartDogu scales up a dogu to 1.
	StartDogu(ctx context.Context, doguResource *v1.Dogu) error
	// CheckStarted checks if the dogu has been successfully scaled to 1.
	CheckStarted(ctx context.Context, doguResource *v1.Dogu) error
}

// StopDoguManager includes functionality to stop running dogus.
type StopDoguManager interface {
	// StopDogu scales down a dogu to 0.
	StopDogu(ctx context.Context, doguResource *v1.Dogu) error
	// CheckStopped checks if the dogu has been successfully scaled to 0.
	CheckStopped(ctx context.Context, doguResource *v1.Dogu) error
}

// DoguStartStopManager includes functionality to start and stop dogus.
type DoguStartStopManager interface {
	StartDoguManager
	StopDoguManager
}

// DoguManager abstracts the simple dogu operations in a k8s CES.
type DoguManager interface {
	InstallManager
	UpgradeManager
	DeleteManager
	VolumeManager
	AdditionalIngressAnnotationsManager
	SupportManager
	StartDoguManager
	StopDoguManager
}

// RequeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type RequeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *v1.Dogu, err error, onRequeue func(dogu *v1.Dogu) error) (result ctrl.Result, requeueErr error)
}

// RequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type RequirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (coreV1.ResourceRequirements, error)
}
