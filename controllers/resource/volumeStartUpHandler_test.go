package resource

import (
	"context"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestVolumeStartUpHandler(t *testing.T) {

	t.Run("simple constructor", func(t *testing.T) {
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		_ = NewVolumeStartupHandler(clientMock, doguClient)
	})

}

func TestVolumeStartUpHandler_Start(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		dogu := &doguv2.Dogu{}
		requests := make(map[corev1.ResourceName]resource.Quantity)
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: requests}}, Status: corev1.PersistentVolumeClaimStatus{Capacity: requests}}

		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		doguClient.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, mock.Anything).Return(dogu, nil)

		doguClient.EXPECT().List(testCtx, mock.Anything).Return(&doguv2.DoguList{Items: []doguv2.Dogu{
			*dogu,
		}}, nil)

		mockClient := NewMockVolumeStartUpHandlerClient(t)

		mockClient.On("Get", context.TODO(), dogu.GetObjectKey(), &corev1.PersistentVolumeClaim{}).
			Run(func(args mock.Arguments) {
				out := args.Get(2).(*corev1.PersistentVolumeClaim)
				*out = *doguPvc
			}).
			Return(nil)

		vsh := NewVolumeStartupHandler(mockClient, doguClient)

		err := vsh.Start(testCtx)
		require.NoError(t, err)
	})
	t.Run("error on getting dogus", func(t *testing.T) {
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		doguClient.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)
		mockClient := NewMockVolumeStartUpHandlerClient(t)
		vsh := NewVolumeStartupHandler(mockClient, doguClient)

		err := vsh.Start(testCtx)
		require.Error(t, err)
	})
}

func TestVolumeStartUpHandler_SetCurrentDataVolumeSize(t *testing.T) {
	t.Run("error no pvc", func(t *testing.T) {
		dogu := &doguv2.Dogu{}
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		doguClient.EXPECT().UpdateStatusWithRetry(context.TODO(), dogu, mock.Anything, mock.Anything).Return(nil, nil)

		err := SetCurrentDataVolumeSize(context.TODO(), doguClient, dogu, nil)

		require.NoError(t, err)
	})
	t.Run("error getting mindatasize", func(t *testing.T) {
		dogu := &doguv2.Dogu{
			Spec: doguv2.DoguSpec{
				Resources: doguv2.DoguResources{
					DataVolumeSize: "invalid",
				},
			},
		}
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)

		err := SetCurrentDataVolumeSize(context.TODO(), doguClient, dogu, nil)

		require.Error(t, err)
	})
	t.Run("error update status", func(t *testing.T) {
		dogu := &doguv2.Dogu{}
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		doguClient.EXPECT().UpdateStatusWithRetry(context.TODO(), dogu, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		err := SetCurrentDataVolumeSize(context.TODO(), doguClient, dogu, nil)

		require.Error(t, err)
	})
	t.Run("run inline function", func(t *testing.T) {
		dogu := &doguv2.Dogu{}
		doguClient := NewMockVolumeStartUpHandlerDoguInterface(t)
		doguClient.EXPECT().UpdateStatusWithRetry(context.TODO(), dogu, mock.Anything, mock.Anything).Run(
			func(ctx context.Context, dogu *doguv2.Dogu, modifyStatusFn func(doguv2.DoguStatus) doguv2.DoguStatus, opts metav1.UpdateOptions) {
				modifyStatusFn(dogu.Status)
			}).Return(nil, nil)

		err := SetCurrentDataVolumeSize(context.TODO(), doguClient, dogu, nil)

		require.NoError(t, err)
	})
}
