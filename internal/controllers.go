package internal

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/api/v1"
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

// DoguManager abstracts the simple dogu operations in a k8s CES.
type DoguManager interface {
	InstallManager
	UpgradeManager
	DeleteManager
	SupportManager
}
