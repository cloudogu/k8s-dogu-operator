package warpmenuentry

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Manager interface {
	EnsureWarpMenuEntry(ctx context.Context, doguResource *doguv2.Dogu) error
	DeleteWarpMenuEntry(ctx context.Context, doguName *cescommons.SimpleName) error
}

type warpmenuentryClient interface {
	Create(ctx context.Context, warpMenuEntry *warpMenuEntryV1.WarpMenuEntry, opts metav1.CreateOptions) (*warpMenuEntryV1.WarpMenuEntry, error)
	Update(ctx context.Context, warpMenuEntry *warpMenuEntryV1.WarpMenuEntry, opts metav1.UpdateOptions) (*warpMenuEntryV1.WarpMenuEntry, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*warpMenuEntryV1.WarpMenuEntry, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type localDoguFetcher interface {
	FetchForResource(ctx context.Context, doguResource *doguv2.Dogu) (*cesappcore.Dogu, error)
}
