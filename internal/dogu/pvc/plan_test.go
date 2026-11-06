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

	t.Run("unmounts the growing claims too during the unmounting phase", func(t *testing.T) {
		plan := NewResizePlan(growing, true, true)

		assert.Equal(t, growing.ResizeRequests, plan.Grow)
		assert.Equal(t, []string{"data", "logs"}, plan.Unmount)
		assert.True(t, plan.HasWork())
	})

	t.Run("leaves the growing claims mounted after the unmounting phase", func(t *testing.T) {
		plan := NewResizePlan(growing, false, true)

		assert.Equal(t, []string{"data"}, plan.Unmount)
	})

	t.Run("requires no suspension for claims that only keep growing after the unmounting phase", func(t *testing.T) {
		plan := NewResizePlan(CheckResult{InProgress: growing.InProgress}, false, true)

		assert.Empty(t, plan.Unmount)
		assert.False(t, plan.HasWork())
	})

	t.Run("requires suspension for an outdated StatefulSet alone", func(t *testing.T) {
		plan := NewResizePlan(CheckResult{OutdatedStatefulSets: []types.UID{"sts-uid"}}, false, true)

		assert.True(t, plan.HasWork())
	})

	t.Run("unmounts nothing with online expansion", func(t *testing.T) {
		plan := NewResizePlan(growing, true, false)

		assert.Equal(t, growing.ResizeRequests, plan.Grow)
		assert.Empty(t, plan.Unmount)
		assert.True(t, plan.HasWork())
	})

	t.Run("requires no suspension for claims that only keep growing with online expansion", func(t *testing.T) {
		plan := NewResizePlan(CheckResult{InProgress: growing.InProgress}, true, false)

		assert.False(t, plan.HasWork())
	})
}
