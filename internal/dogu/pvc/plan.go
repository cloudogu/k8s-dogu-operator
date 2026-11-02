package pvc

import "k8s.io/apimachinery/pkg/types"

// ResizePlan is what the volume resize of a dogu still has to do, as rendered by its chart.
type ResizePlan struct {
	// Grow are the claims whose storage request is below the rendered size.
	Grow []ResizeRequest
	// Detach are the claims to grow plus, while the release is suspended, those still growing, as many storage providers only expand unmounted volumes.
	Detach []string
	// OutdatedStatefulSets have immutable claim templates of another size than rendered, which Helm cannot upgrade.
	OutdatedStatefulSets []types.UID
}

// NewResizePlan derives the plan from the check of the rendered chart and the suspend state of the release.
func NewResizePlan(check CheckResult, releaseSuspended bool) ResizePlan {
	plan := ResizePlan{
		Grow:                 check.ResizeRequests,
		OutdatedStatefulSets: check.OutdatedStatefulSets,
	}
	for _, request := range check.ResizeRequests {
		plan.Detach = append(plan.Detach, request.PVC.Name)
	}
	// a claim grown in an earlier pass is no longer a resize request but stays detached until the release resumes with recreated workloads
	if releaseSuspended {
		for _, key := range check.InProgress {
			plan.Detach = append(plan.Detach, key.Name)
		}
	}

	return plan
}

// Active reports whether the release has to stay suspended until the plan is carried out.
func (p ResizePlan) Active() bool {
	return len(p.Grow) > 0 || len(p.Detach) > 0 || len(p.OutdatedStatefulSets) > 0
}
