package flux

import (
	"context"
	"errors"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
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

func TestHelmReleaseReader_StoredRelease(t *testing.T) {
	clientSet := kubernetesfake.NewClientset()
	reader := NewHelmReleaseReader(nil, clientSet)

	t.Run("no history means no stored revision", func(t *testing.T) {
		stored, err := reader.StoredRelease(helmRelease("cas", "ecosystem"))

		require.NoError(t, err)
		assert.Nil(t, stored)
	})

	t.Run("reads the revision referenced by Flux from the storage namespace", func(t *testing.T) {
		// Keep CR name, Helm release name and storage namespace distinct to catch incorrect lookups.
		hr := helmRelease("cas", "ecosystem")
		hr.Spec.StorageNamespace = "helm-storage"
		hr.Status.History = helmv2.Snapshots{
			{Name: "installed-cas", Version: 1},
			{Name: "installed-cas", Version: 2},
		}
		store := storage.Init(driver.NewSecrets(clientSet.CoreV1().Secrets("helm-storage")))
		for _, revision := range []int{1, 2, 3} {
			stored := &sdk.Release{
				Name:      "installed-cas",
				Namespace: "workloads",
				Version:   revision,
				Info:      &sdk.Info{Status: sdk.StatusDeployed},
				Config:    map[string]any{"mode": "old"},
			}
			if revision == 2 {
				stored.Info.Status = sdk.StatusFailed
				stored.Config["mode"] = "new"
				stored.Manifest = `
apiVersion: v1
kind: Service
metadata:
  name: cas
`
			}
			require.NoError(t, store.Create(stored))
		}

		// Read Flux's revision 2, not the newer revision 3 or the last successful revision.
		stored, err := reader.StoredRelease(hr)

		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, 2, stored.Version)
		assert.Equal(t, sdk.StatusFailed, stored.Info.Status)
		assert.Equal(t, map[string]any{"mode": "new"}, stored.Config)
		assert.Contains(t, stored.Manifest, "kind: Service")
	})
}

func TestHelmReleaseReader_ResolveOperation(t *testing.T) {
	ctx := context.Background()

	t.Run("no HelmRelease resolves to install at revision 1", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()

		op, err := NewHelmReleaseReader(cl, nil).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsInstall: true, Revision: 1}, op)
	})

	t.Run("HelmRelease without history resolves to install at revision 1", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(helmRelease("cas", "ecosystem")).Build()

		op, err := NewHelmReleaseReader(cl, nil).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsInstall: true, Revision: 1}, op)
	})

	t.Run("installed HelmRelease resolves to upgrade at the next revision", func(t *testing.T) {
		// Latest() sorts by version descending, so the order in the slice must not matter.
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(helmRelease("cas", "ecosystem", 1, 3, 2)).Build()

		op, err := NewHelmReleaseReader(cl, nil).ResolveOperation(ctx, "cas", "ecosystem")

		require.NoError(t, err)
		assert.Equal(t, ReleaseOperation{IsUpgrade: true, Revision: 4}, op)
	})

	t.Run("propagates non-NotFound errors", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errors.New("api down")
			},
		}).Build()

		_, err := NewHelmReleaseReader(cl, nil).ResolveOperation(ctx, "cas", "ecosystem")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get HelmRelease ecosystem/cas")
	})
}
