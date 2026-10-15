package deletion

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	fluxsource "github.com/fluxcd/source-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testNamespace     = "ecosystem"
	testVersion       = "1.0.0"
	testDoguName      = "nexus"
	testDoguNamespace = "testing"
	// fluxFinalizer is used in tests to keep a flux resource around with a deletion timestamp, so the
	// fake client simulates the window where flux is still running its uninstall.
	fluxFinalizer = "finalizers.fluxcd.io"
)

var (
	testCtx            = context.Background()
	testNamespacedName = types.NamespacedName{Namespace: testNamespace, Name: testDoguName}

	testScheme = runtime.NewScheme()
	_          = v3beta1.AddToScheme(testScheme)
	_          = fluxhelm.AddToScheme(testScheme)
	_          = fluxsource.AddToScheme(testScheme)
)

func newTestDoguResource() *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name:       testDoguName,
		Namespace:  testNamespace,
		Finalizers: []string{stepsv3.FinalizerName},
		Spec: v3beta1.DoguSpec{
			Name:          testDoguName,
			DoguNamespace: testDoguNamespace,
			Version:       testVersion,
		},
	}
}

func newTestHelmRelease() *fluxhelm.HelmRelease {
	return &fluxhelm.HelmRelease{
		Name: testDoguName, Namespace: testNamespace,
	}
}

func newTestOCIRepository() *fluxsource.OCIRepository {
	return &fluxsource.OCIRepository{
		Name: testDoguName, Namespace: testNamespace,
	}
}
