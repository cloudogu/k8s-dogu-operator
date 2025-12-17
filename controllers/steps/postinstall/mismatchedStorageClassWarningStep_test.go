package postinstall

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewMismatchedStorageClassWarningStep(t *testing.T) {
	// given
	clientMock := newMockK8sClient(t)
	recorderMock := newMockEventRecorder(t)

	// when
	step := NewMismatchedStorageClassWarningStep(clientMock, recorderMock)

	// then
	require.NotNil(t, step)
	assert.Same(t, clientMock, step.client)
	assert.Same(t, recorderMock, step.recorder)
}

func TestMismatchedStorageClassWarningStep_Run(t *testing.T) {
	type fields struct {
		client   func(t *testing.T) k8sClient
		recorder func(t *testing.T) eventRecorder
	}
	type args struct {
		ctx      context.Context
		resource *v2.Dogu
	}
	// shared pointer to simulate exact pointer equality when needed
	scFast := "fast"

	tests := []struct {
		name   string
		fields fields
		args   args
		want   steps.StepResult
	}{
		{
			name: "get pvc returns error -> requeue with error",
			fields: fields{
				client: func(t *testing.T) k8sClient {
					c := newMockK8sClient(t)
					c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
					return c
				},
				recorder: func(t *testing.T) eventRecorder { return newMockEventRecorder(t) },
			},
			args: args{
				ctx:      context.TODO(),
				resource: &v2.Dogu{Spec: v2.DoguSpec{Resources: v2.DoguResources{StorageClassName: strPtr("fast")}}},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get data pvc for dogu : %w", assert.AnError)),
		},
		{
			name: "no storage class specified in dogu -> continue, no events",
			fields: fields{
				client:   func(t *testing.T) k8sClient { return newMockK8sClient(t) },
				recorder: func(t *testing.T) eventRecorder { return newMockEventRecorder(t) },
			},
			args: args{
				ctx: context.TODO(),
				resource: &v2.Dogu{ // StorageClassName == nil
					Spec: v2.DoguSpec{Resources: v2.DoguResources{}},
				},
			},
			want: steps.Continue(),
		},
		{
			name: "pvc storage class not yet set -> continue, no events",
			fields: fields{
				client: func(t *testing.T) k8sClient {
					c := newMockK8sClient(t)
					c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(
						func(_ context.Context, _ types.NamespacedName, obj crclient.Object, _ ...crclient.GetOption) {
							// populate the provided PVC with nil StorageClassName
							if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
								*pvc = corev1.PersistentVolumeClaim{Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: nil}}
							}
						},
					)
					return c
				},
				recorder: func(t *testing.T) eventRecorder { return newMockEventRecorder(t) },
			},
			args: args{
				ctx: context.TODO(),
				resource: &v2.Dogu{ // StorageClassName set in dogu
					Spec: v2.DoguSpec{Resources: v2.DoguResources{StorageClassName: strPtr("fast")}},
				},
			},
			want: steps.Continue(),
		},
		{
			name: "matching storage class -> continue, no events",
			fields: fields{
				client: func(t *testing.T) k8sClient {
					c := newMockK8sClient(t)
					c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(
						func(_ context.Context, _ types.NamespacedName, obj crclient.Object, _ ...crclient.GetOption) {
							if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
								// use the very same pointer value as in the dogu spec
								*pvc = corev1.PersistentVolumeClaim{Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: &scFast}}
							}
						},
					)
					return c
				},
				recorder: func(t *testing.T) eventRecorder { return newMockEventRecorder(t) },
			},
			args: args{
				ctx:      context.TODO(),
				resource: &v2.Dogu{Spec: v2.DoguSpec{Resources: v2.DoguResources{StorageClassName: &scFast}}},
			},
			want: steps.Continue(),
		},
		{
			name: "mismatched storage classes -> continue, warning events emitted",
			fields: fields{
				client: func(t *testing.T) k8sClient {
					c := newMockK8sClient(t)
					c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(
						func(_ context.Context, _ types.NamespacedName, obj crclient.Object, _ ...crclient.GetOption) {
							if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
								sc := "slow"
								*pvc = corev1.PersistentVolumeClaim{Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: &sc}}
							}
						},
					)
					return c
				},
				recorder: func(t *testing.T) eventRecorder {
					r := newMockEventRecorder(t)
					// Expect two warning events with the same message, one for PVC and one for Dogu
					doguSC := "fast"
					pvcSC := "slow"
					msg := "Mismatched storage class name between dogu and pvc resource: {dogu: " + doguSC + ", pvc: " + pvcSC + "}"
					// Expect warnings on both PVC and Dogu objects
					r.EXPECT().Event(mock.Anything, corev1.EventTypeWarning, "StorageClassMismatch", msg)
					r.EXPECT().Event(mock.Anything, corev1.EventTypeWarning, "StorageClassMismatch", msg)
					return r
				},
			},
			args: args{
				ctx:      context.TODO(),
				resource: &v2.Dogu{Spec: v2.DoguSpec{Resources: v2.DoguResources{StorageClassName: strPtr("fast")}}},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MismatchedStorageClassWarningStep{
				client:   tt.fields.client(t),
				recorder: tt.fields.recorder(t),
			}
			assert.Equalf(t, tt.want, m.Run(tt.args.ctx, tt.args.resource), "Run(%v, %v)", tt.args.ctx, tt.args.resource)
		})
	}
}

// helpers for tests
func strPtr(s string) *string { return &s }

// mock argument helpers to keep expectations readable; these accept any value since we only care about obj population
// We rely on mock.Anything from testify/mock but keep wrappers to avoid importing types not needed in test cases.
type anyArg struct{}

func mockCtx() interface{} { return mock.Anything }
func mockKey() interface{} { return mock.Anything }
func mockPVC() interface{} { return mock.Anything }
