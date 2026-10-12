package flux

import (
	"context"
	"errors"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, helmv2.AddToScheme(scheme))
	return scheme
}

func helmRelease(name, namespace string, revisions ...int) *helmv2.HelmRelease {
	hr := &helmv2.HelmRelease{Name: name, Namespace: namespace}
	for _, r := range revisions {
		hr.Status.History = append(hr.Status.History, &helmv2.Snapshot{Version: r})
	}
	return hr
}

func TestHelmReleaseReader_ResolveOperation(t *testing.T) {
	ctx := context.Background()

	t.Run("no HelmRelease resolves to install at revision 1", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()

		op, err := NewHelmReleaseReader(cl).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsInstall: true, Revision: 1}, op)
	})

	t.Run("HelmRelease without history resolves to install at revision 1", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(helmRelease("cas", "ecosystem")).Build()

		op, err := NewHelmReleaseReader(cl).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsInstall: true, Revision: 1}, op)
	})

	t.Run("installed HelmRelease resolves to upgrade at the next revision", func(t *testing.T) {
		// Latest() sorts by version descending, so the order in the slice must not matter.
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(helmRelease("cas", "ecosystem", 1, 3, 2)).Build()

		op, err := NewHelmReleaseReader(cl).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsUpgrade: true, Revision: 4}, op)
	})

	t.Run("propagates non-NotFound errors", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errors.New("api down")
			},
		}).Build()

		_, err := NewHelmReleaseReader(cl).ResolveOperation(ctx, "cas", "ecosystem")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get HelmRelease ecosystem/cas")
	})
}
