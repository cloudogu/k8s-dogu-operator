package exposition

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Manager describes the Exposition lifecycle for a dogu.
type Manager interface {
	EnsureExposition(ctx context.Context, doguResource *doguv2.Dogu) error
	RemoveExposition(ctx context.Context, doguName cescommons.SimpleName) error
}

type expositionClient interface {
	Create(ctx context.Context, exposition *expv1.Exposition, opts metav1.CreateOptions) (*expv1.Exposition, error)
	Update(ctx context.Context, exposition *expv1.Exposition, opts metav1.UpdateOptions) (*expv1.Exposition, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*expv1.Exposition, error)
}

type localDoguFetcher interface {
	FetchForResource(ctx context.Context, doguResource *doguv2.Dogu) (*cesappcore.Dogu, error)
}

type imageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}
