package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
)

const offlineExpansionHint = "if the storage only expands unmounted volumes, turn on offline volume expansion"

// WaitForVolumeResizeCompletionStep keeps the dogu in ResizingPVC until Kubernetes has grown every rendered volume to its desired capacity.
// It runs after the release resumed, as storage often grows the file system only when a recreated pod mounts the volume.
type WaitForVolumeResizeCompletionStep struct {
	chartService  chartService
	assembler     valueAssembler
	resizeChecker ResizeChecker
}

func NewWaitForVolumeResizeCompletionStep(chartService chartService, assembler valueAssembler, resizeChecker ResizeChecker) *WaitForVolumeResizeCompletionStep {
	return &WaitForVolumeResizeCompletionStep{
		chartService:  chartService,
		assembler:     assembler,
		resizeChecker: resizeChecker,
	}
}

func (wvr *WaitForVolumeResizeCompletionStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	rendered, err := renderChart(ctx, wvr.chartService, wvr.assembler, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	// check the rendered claims instead of the status, as it omits claims whose workloads are gone
	check, err := wvr.resizeChecker.Check(ctx, doguResource.Namespace, rendered)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to check volumes for resizing: %w", err), v3beta1.ReasonResizingPVC)
	}

	var resizing []string
	for _, request := range check.ResizeRequests {
		resizing = append(resizing, request.PVC.Name)
	}
	for _, key := range check.InProgress {
		resizing = append(resizing, key.Name)
	}
	if len(resizing) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for volumes to reach their desired capacity: %v; %s", resizing, offlineExpansionHint))
	}

	return stepsv3.Continue()
}
