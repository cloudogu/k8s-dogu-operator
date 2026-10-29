package install

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func planFor(t *testing.T, checkResult pvc.CheckResult, apiReader client.Reader) (VolumeResizePlan, error) {
	t.Helper()
	doguResource := testDoguResource.DeepCopy()
	chartService, assembler := mockChartRendering(t, doguResource, nil, nil)
	checker := NewMockResizeChecker(t)
	checker.EXPECT().Check(mock.Anything, testNamespace, mock.Anything).Return(checkResult, nil)

	return NewVolumeResizePlanner(chartService, assembler, checker, apiReader).Plan(testCtx, doguResource)
}

func TestVolumeResizePlanner_Plan(t *testing.T) {
	growing := pvc.CheckResult{
		ResizeRequests: []pvc.ResizeRequest{{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}}},
		InProgress:     []client.ObjectKey{{Namespace: testNamespace, Name: "logs"}},
	}

	t.Run("detaches the growing claims too while the release is suspended", func(t *testing.T) {
		plan, err := planFor(t, growing, fake.NewClientBuilder().WithScheme(resizeScheme).WithObjects(helmRelease(true)).Build())

		require.NoError(t, err)
		assert.Equal(t, growing.ResizeRequests, plan.Grow)
		assert.Equal(t, []string{"data", "logs"}, plan.Detach)
		assert.True(t, plan.Active())
	})

	t.Run("leaves the growing claims attached once the release is resumed", func(t *testing.T) {
		plan, err := planFor(t, growing, fake.NewClientBuilder().WithScheme(resizeScheme).WithObjects(helmRelease(false)).Build())

		require.NoError(t, err)
		assert.Equal(t, []string{"data"}, plan.Detach)
	})

	t.Run("leaves the growing claims attached without a release", func(t *testing.T) {
		plan, err := planFor(t, pvc.CheckResult{InProgress: growing.InProgress}, fake.NewClientBuilder().WithScheme(resizeScheme).Build())

		require.NoError(t, err)
		assert.Empty(t, plan.Detach)
		assert.False(t, plan.Active())
	})

	t.Run("is active for an outdated StatefulSet alone", func(t *testing.T) {
		plan, err := planFor(t, pvc.CheckResult{OutdatedStatefulSets: []types.UID{"sts-uid"}}, nil)

		require.NoError(t, err)
		assert.True(t, plan.Active())
	})

	t.Run("fails when the release cannot be read", func(t *testing.T) {
		failing := fake.NewClientBuilder().WithScheme(resizeScheme).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errors.New("get failed")
			},
		}).Build()

		_, err := planFor(t, growing, failing)

		require.ErrorContains(t, err, "failed to get HelmRelease")
	})
}
