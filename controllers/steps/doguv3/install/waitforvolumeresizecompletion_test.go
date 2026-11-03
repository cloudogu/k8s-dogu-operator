package install

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWaitForVolumeResizeCompletionStep_Run(t *testing.T) {
	t.Run("continue when every volume has reached its desired capacity", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)

		result := NewWaitForVolumeResizeCompletionStep(cs, as, mockResizeCheck(t, pvc.CheckResult{}, nil)).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
	})

	t.Run("wait for volumes still to grow and volumes growing", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)
		check := mockResizeCheck(t, pvc.CheckResult{
			ResizeRequests: []pvc.ResizeRequest{{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}}},
			InProgress:     []client.ObjectKey{{Namespace: testNamespace, Name: "db-nexus-0"}},
		}, nil)

		result := NewWaitForVolumeResizeCompletionStep(cs, as, check).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, "waiting for volumes to reach their desired capacity: [data db-nexus-0]; if the storage only expands unmounted volumes, turn on offline volume expansion"), result)
	})

	t.Run("requeue with error when the chart cannot be rendered", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		cs, as := mockChartRendering(t, doguResource, nil, assert.AnError)

		result := NewWaitForVolumeResizeCompletionStep(cs, as, NewMockResizeChecker(t)).Run(testCtx, doguResource)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to render chart")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	t.Run("requeue with error when the volumes cannot be checked", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		cs, as := mockChartRendering(t, doguResource, renderedChart(t), nil)

		result := NewWaitForVolumeResizeCompletionStep(cs, as, mockResizeCheck(t, pvc.CheckResult{}, assert.AnError)).Run(testCtx, doguResource)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to check volumes for resizing")
		assert.Equal(t, v3beta1.ReasonResizingPVC, result.ReadyReason)
	})

	for name, modify := range map[string]func(*v3beta1.Dogu){
		"stopped": func(d *v3beta1.Dogu) { d.Spec.Stopped = true },
		"paused":  func(d *v3beta1.Dogu) { d.Spec.PauseReconciliation = true },
	} {
		t.Run("continue without waiting for a "+name+" dogu", func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			modify(doguResource)

			result := NewWaitForVolumeResizeCompletionStep(newMockChartService(t), newMockValueAssembler(t), NewMockResizeChecker(t)).Run(testCtx, doguResource)

			assert.Equal(t, doguv3.Continue(), result)
		})
	}
}
