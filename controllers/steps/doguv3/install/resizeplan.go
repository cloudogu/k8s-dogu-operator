package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	flux "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeResizePlanner derives the pvc.ResizePlan of a dogu, so that SuspendHelmReleaseStep and RequestVolumeResizeStep act on the same plan.
type VolumeResizePlanner struct {
	chartService  chartService
	assembler     valueAssembler
	resizeChecker ResizeChecker
	// apiReader reads the release uncached, so that both steps see the same suspend state.
	apiReader        client.Reader
	offlineExpansion bool
}

func NewVolumeResizePlanner(chartService chartService, assembler valueAssembler, resizeChecker ResizeChecker, apiReader client.Reader, offlineExpansion bool) *VolumeResizePlanner {
	return &VolumeResizePlanner{
		chartService:     chartService,
		assembler:        assembler,
		resizeChecker:    resizeChecker,
		apiReader:        apiReader,
		offlineExpansion: offlineExpansion,
	}
}

func (vrp *VolumeResizePlanner) Plan(ctx context.Context, doguResource *v3beta1.Dogu) (pvc.ResizePlan, error) {
	rendered, err := renderChart(ctx, vrp.chartService, vrp.assembler, doguResource)
	if err != nil {
		return pvc.ResizePlan{}, fmt.Errorf("failed to render chart: %w", err)
	}

	checkResult, err := vrp.resizeChecker.Check(ctx, doguResource.Namespace, rendered)
	if err != nil {
		return pvc.ResizePlan{}, fmt.Errorf("failed to check volumes for resizing: %w", err)
	}

	// the suspend state only matters for claims still growing offline, so skip reading the release otherwise
	suspended := false
	if vrp.offlineExpansion && len(checkResult.InProgress) > 0 {
		suspended, err = vrp.releaseSuspended(ctx, doguResource)
		if err != nil {
			return pvc.ResizePlan{}, err
		}
	}

	return pvc.NewResizePlan(checkResult, suspended, vrp.offlineExpansion), nil
}

func (vrp *VolumeResizePlanner) releaseSuspended(ctx context.Context, doguResource *v3beta1.Dogu) (bool, error) {
	release := &flux.HelmRelease{}
	err := vrp.apiReader.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get HelmRelease %q: %w", doguResource.Spec.Name, err)
	}

	return release.Spec.Suspend, nil
}
