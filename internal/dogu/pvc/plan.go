package pvc

import "k8s.io/apimachinery/pkg/types"

// ResizePlan is what the volume resize of a dogu still has to do, as rendered by its chart.
type ResizePlan struct {
	// Grow are the claims whose storage request is below the rendered size.
	Grow []ResizeRequest
	// Unmount are, with offline expansion only, the claims to grow plus, during the unmounting phase, those still growing.
	Unmount []string
	// OutdatedStatefulSets have immutable claim templates of another size than rendered, which Helm cannot upgrade.
	OutdatedStatefulSets []types.UID
}

// NewResizePlan derives the plan from the check of the rendered chart, whether an offline resize is in its unmounting phase, and the expansion mode.
func NewResizePlan(check CheckResult, inUnmountingPhase, offlineExpansion bool) ResizePlan {
	plan := ResizePlan{
		Grow:                 check.ResizeRequests,
		OutdatedStatefulSets: check.OutdatedStatefulSets,
	}
	if !offlineExpansion {
		return plan
	}

	for _, request := range check.ResizeRequests {
		plan.Unmount = append(plan.Unmount, request.PVC.Name)
	}
	// a claim grown in an earlier pass is no longer a resize request but stays unmounted until the unmounting phase ends
	if inUnmountingPhase {
		for _, key := range check.InProgress {
			plan.Unmount = append(plan.Unmount, key.Name)
		}
	}

	return plan
}

// HasWork reports whether the release has to stay suspended until the plan is carried out.
func (p ResizePlan) HasWork() bool {
	return len(p.Grow) > 0 || len(p.Unmount) > 0 || len(p.OutdatedStatefulSets) > 0
}
