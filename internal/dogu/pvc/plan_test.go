package pvc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewResizePlan(t *testing.T) {
	growing := CheckResult{
		ResizeRequests: []ResizeRequest{{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}}},
		InProgress:     []client.ObjectKey{{Namespace: testNamespace, Name: "logs"}},
	}

	t.Run("detaches the growing claims too while the release is suspended", func(t *testing.T) {
		plan := NewResizePlan(growing, true)

		assert.Equal(t, growing.ResizeRequests, plan.Grow)
		assert.Equal(t, []string{"data", "logs"}, plan.Detach)
		assert.True(t, plan.Active())
	})

	t.Run("leaves the growing claims attached once the release is resumed", func(t *testing.T) {
		plan := NewResizePlan(growing, false)

		assert.Equal(t, []string{"data"}, plan.Detach)
	})

	t.Run("is inactive for claims that only keep growing in a resumed release", func(t *testing.T) {
		plan := NewResizePlan(CheckResult{InProgress: growing.InProgress}, false)

		assert.Empty(t, plan.Detach)
		assert.False(t, plan.Active())
	})

	t.Run("is active for an outdated StatefulSet alone", func(t *testing.T) {
		plan := NewResizePlan(CheckResult{OutdatedStatefulSets: []types.UID{"sts-uid"}}, false)

		assert.True(t, plan.Active())
	})
}
