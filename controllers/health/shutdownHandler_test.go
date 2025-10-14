package health

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func TestNewShutdownHandler(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)

		// when
		handler := NewShutdownHandler(doguInterfaceMock)

		// then
		assert.Equal(t, doguInterfaceMock, handler.doguInterface)
	})

}

func TestShutdownHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		doguInterfaceFn func(t *testing.T) client.DoguInterface
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to list dogus",
			doguInterfaceFn: func(t *testing.T) client.DoguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)
				return mck
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should fail to update dogu status",
			doguInterfaceFn: func(t *testing.T) client.DoguInterface {
				mck := newMockDoguInterface(t)
				ldapDogu := &v2.Dogu{
					ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
				}
				casDogu := &v2.Dogu{
					ObjectMeta: metav1.ObjectMeta{Name: "cas"},
				}
				mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&v2.DoguList{Items: []v2.Dogu{
					*ldapDogu,
					*casDogu,
				}}, nil)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, ldapDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, casDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
				return mck
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "failed to set health status and conditions of \"ldap\" to unknown") &&
					assert.ErrorContains(t, err, "failed to set health status and conditions of \"cas\" to unknown")
			},
		},
		{
			name: "should succeed to update dogu status",
			doguInterfaceFn: func(t *testing.T) client.DoguInterface {
				mck := newMockDoguInterface(t)
				ldapDogu := &v2.Dogu{
					ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
				}
				casDogu := &v2.Dogu{
					ObjectMeta: metav1.ObjectMeta{Name: "cas"},
				}
				mck.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&v2.DoguList{Items: []v2.Dogu{
					*ldapDogu,
					*casDogu,
				}}, nil)
				runAndReturnFn := func(ctx context.Context, dogu *v2.Dogu, f func(v2.DoguStatus) v2.DoguStatus, options metav1.UpdateOptions) (*v2.Dogu, error) {
					dogu.Status = f(dogu.Status)
					reason := "StoppingOperator"
					message := "The operator is shutting down"
					expectedConditions := []metav1.Condition{
						{
							Type:    v2.ConditionReady,
							Status:  metav1.ConditionUnknown,
							Reason:  reason,
							Message: message,
						},
						{
							Type:    v2.ConditionHealthy,
							Status:  metav1.ConditionUnknown,
							Reason:  reason,
							Message: message,
						},
						{
							Type:    v2.ConditionSupportMode,
							Status:  metav1.ConditionUnknown,
							Reason:  reason,
							Message: message,
						},
						{
							Type:    v2.ConditionMeetsMinVolumeSize,
							Status:  metav1.ConditionUnknown,
							Reason:  reason,
							Message: message,
						},
					}
					gomega.NewWithT(t).Expect(dogu.Status.Conditions).
						To(conditions.MatchConditions(expectedConditions, conditions.IgnoreLastTransitionTime(true)))

					assert.Equal(t, v2.HealthStatus("unknown"), dogu.Status.Health)

					return dogu, nil
				}
				mck.EXPECT().UpdateStatusWithRetry(testCtx, ldapDogu, mock.Anything, metav1.UpdateOptions{}).
					RunAndReturn(runAndReturnFn)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, casDogu, mock.Anything, metav1.UpdateOptions{}).
					RunAndReturn(runAndReturnFn)
				return mck
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShutdownHandler{
				doguInterface: tt.doguInterfaceFn(t),
			}
			tt.wantErr(t, s.Handle(testCtx), fmt.Sprintf("Handle(%v)", testCtx))
		})
	}
}
