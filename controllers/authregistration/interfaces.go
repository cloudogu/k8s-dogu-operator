package authregistration

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	authRegV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	"github.com/cloudogu/k8s-registry-lib/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthRegistrationManager describes the AuthRegistration lifecycle for a dogu.
type Manager interface {
	// EnsureAuthRegistration creates/updates the AuthRegistration and syncs sensitive credentials.
	EnsureAuthRegistration(ctx context.Context, dogu *cesappcore.Dogu) error
	// RemoveAuthRegistration removes the AuthRegistration belonging to the given dogu.
	RemoveAuthRegistration(ctx context.Context, doguName cescommons.SimpleName) error
}

type authRegistrationClient interface {
	Create(ctx context.Context, authRegistration *authRegV1.AuthRegistration, opts metav1.CreateOptions) (*authRegV1.AuthRegistration, error)
	Update(ctx context.Context, authRegistration *authRegV1.AuthRegistration, opts metav1.UpdateOptions) (*authRegV1.AuthRegistration, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*authRegV1.AuthRegistration, error)
}

type secretClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
}

type sensitiveDoguConfigRepository interface {
	Get(ctx context.Context, name cescommons.SimpleName) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
}
