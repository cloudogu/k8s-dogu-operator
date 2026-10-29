package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	flux "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeResizePlan is what the volume resize of a dogu still has to do, as rendered by its chart.
type VolumeResizePlan struct {
	// Grow are the claims whose storage request is below the rendered size.
	Grow []pvc.ResizeRequest
	// Detach are the claims to grow plus, while the release is suspended, those still growing, as many storage providers only expand unmounted volumes.
	Detach []string
	// OutdatedStatefulSets have immutable claim templates of another size than rendered, which Helm cannot upgrade.
	OutdatedStatefulSets []types.UID
	rendered             []*unstructured.Unstructured
}

// Active reports whether the release has to stay suspended until ResizePVCStep has carried out the plan.
func (p VolumeResizePlan) Active() bool {
	return len(p.Grow) > 0 || len(p.Detach) > 0 || len(p.OutdatedStatefulSets) > 0
}

// VolumeResizePlanner derives the VolumeResizePlan, so that SuspendHelmReleaseStep and ResizePVCStep act on the same plan.
type VolumeResizePlanner struct {
	chartService  chartService
	assembler     valueAssembler
	resizeChecker ResizeChecker
	// apiReader reads the release uncached, so that both steps see the same suspend state.
	apiReader client.Reader
}

func NewVolumeResizePlanner(chartService chartService, assembler valueAssembler, resizeChecker ResizeChecker, apiReader client.Reader) *VolumeResizePlanner {
	return &VolumeResizePlanner{
		chartService:  chartService,
		assembler:     assembler,
		resizeChecker: resizeChecker,
		apiReader:     apiReader,
	}
}

func (vrp *VolumeResizePlanner) Plan(ctx context.Context, doguResource *v3beta1.Dogu) (VolumeResizePlan, error) {
	rendered, err := renderChart(ctx, vrp.chartService, vrp.assembler, doguResource)
	if err != nil {
		return VolumeResizePlan{}, err
	}

	checkResult, err := vrp.resizeChecker.Check(ctx, doguResource.Namespace, rendered)
	if err != nil {
		return VolumeResizePlan{}, fmt.Errorf("failed to check volumes: %w", err)
	}

	plan := VolumeResizePlan{
		Grow:                 checkResult.ResizeRequests,
		OutdatedStatefulSets: checkResult.OutdatedStatefulSets,
		rendered:             rendered,
	}
	for _, request := range checkResult.ResizeRequests {
		plan.Detach = append(plan.Detach, request.PVC.Name)
	}
	if len(checkResult.InProgress) == 0 {
		return plan, nil
	}

	// a claim grown in an earlier pass is no longer a resize request but stays detached until the release resumes with recreated workloads
	suspended, err := vrp.releaseSuspended(ctx, doguResource)
	if err != nil {
		return VolumeResizePlan{}, err
	}
	if suspended {
		for _, key := range checkResult.InProgress {
			plan.Detach = append(plan.Detach, key.Name)
		}
	}

	return plan, nil
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
