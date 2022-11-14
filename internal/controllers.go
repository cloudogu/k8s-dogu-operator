package internal

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type InstallManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *v1.Dogu) error
}

type UpgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *v1.Dogu) error
}

type DeleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *v1.Dogu) error
}

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
