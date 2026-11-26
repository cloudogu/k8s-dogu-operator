package upgrade

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "ecosystem"

func testDogu(version string) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name: "nexus", Namespace: testNamespace,
		Spec: v3beta1.DoguSpec{Name: "nexus", DoguNamespace: "official", Version: version},
	}
}

func helmRelease(history ...*fluxhelm.Snapshot) *fluxhelm.HelmRelease {
	return &fluxhelm.HelmRelease{
		Name: "nexus", Namespace: testNamespace,
		Status: fluxhelm.HelmReleaseStatus{History: history},
	}
}

func snapshot(revision int, chartVersion, status string) *fluxhelm.Snapshot {
	return &fluxhelm.Snapshot{Version: revision, ChartVersion: chartVersion, Status: status}
}

func detect(t *testing.T, dogu *v3beta1.Dogu, objects ...client.Object) (Change, error) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, fluxhelm.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	return NewDetector(cl).Detect(context.Background(), dogu)
}

func TestDetector_Detect(t *testing.T) {
	t.Run("dogu without HelmRelease is a new installation", func(t *testing.T) {
		change, err := detect(t, testDogu("2.0.0"))

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: NoChange, Target: "2.0.0"}, change)
	})

	t.Run("newer requested version than the deployed chart version is an upgrade", func(t *testing.T) {
		release := helmRelease(snapshot(1, "1.5.0+ebb48ffcdce3", "deployed"))

		change, err := detect(t, testDogu("2.0.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: Upgrade, Installed: "1.5.0", Target: "2.0.0"}, change)
	})

	t.Run("older requested version than the deployed chart version is a downgrade", func(t *testing.T) {
		release := helmRelease(snapshot(1, "2.0.0", "deployed"))

		change, err := detect(t, testDogu("1.5.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: Downgrade, Installed: "2.0.0", Target: "1.5.0"}, change)
	})

	t.Run("requested version with v prefix is compared as semantic version", func(t *testing.T) {
		release := helmRelease(snapshot(1, "1.5.0", "deployed"))

		change, err := detect(t, testDogu("v2.0.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: Upgrade, Installed: "1.5.0", Target: "v2.0.0"}, change)
	})

	t.Run("deployed requested version is no change", func(t *testing.T) {
		release := helmRelease(snapshot(1, "2.0.0", "deployed"))

		change, err := detect(t, testDogu("2.0.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: NoChange, Installed: "2.0.0", Target: "2.0.0"}, change)
	})

	t.Run("HelmRelease without deployed revision is a new installation", func(t *testing.T) {
		release := helmRelease(snapshot(1, "1.5.0", "failed"))

		change, err := detect(t, testDogu("2.0.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: NoChange, Target: "2.0.0"}, change)
	})

	t.Run("failed revisions are ignored for the installed version", func(t *testing.T) {
		release := helmRelease(snapshot(3, "2.0.0", "failed"), snapshot(2, "1.5.0", "deployed"), snapshot(1, "1.0.0", "superseded"))

		change, err := detect(t, testDogu("2.0.0"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: Upgrade, Installed: "1.5.0", Target: "2.0.0"}, change)
	})

	t.Run("unparsable requested version is an invalid version change", func(t *testing.T) {
		release := helmRelease(snapshot(1, "1.5.0", "deployed"))

		change, err := detect(t, testDogu("two"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: InvalidVersion, Installed: "1.5.0", Target: "two"}, change)
	})

	t.Run("unparsable but unchanged version is no change", func(t *testing.T) {
		release := helmRelease(snapshot(1, "two", "deployed"))

		change, err := detect(t, testDogu("two"), release)

		require.NoError(t, err)
		assert.Equal(t, Change{Kind: NoChange, Installed: "two", Target: "two"}, change)
	})
}
