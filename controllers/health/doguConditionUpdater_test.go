package health

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func TestNewDoguConditionUpdater(t *testing.T) {
	t.Run("Successfully created dogu condition updater", func(t *testing.T) {
		mck := newMockDoguInterface(t)
		step := NewDoguConditionUpdater(
			mck,
		)

		assert.NotEmpty(t, step)
		assert.Same(t, mck, step.doguInterface)
	})
}

func TestDoguConditionUpdater_UpdateCondition(t *testing.T) {
	tests := []struct {
		name            string
		doguInterfaceFn func(t *testing.T) doguInterface
		doguResource    *v2.Dogu
		condition       v1.Condition
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "should succeed to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, fn func(status v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) {
					g := gomega.NewWithT(t)
					status := fn(dogu.Status)
					g.Expect(status.Conditions).
						To(conditions.MatchConditions([]v1.Condition{{
							Type:    "test",
							Status:  "test",
							Reason:  "test",
							Message: "test",
						}}, conditions.IgnoreLastTransitionTime(true)))
				}).Return(&v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{
							{
								Type:               "test",
								Status:             "test",
								Reason:             "test",
								Message:            "test",
								LastTransitionTime: v1.Now(),
							},
						},
					},
				}, nil)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			condition: v1.Condition{
				Type:    "test",
				Status:  "test",
				Reason:  "test",
				Message: "test",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dcu := &DoguConditionUpdater{
				doguInterface: tt.doguInterfaceFn(t),
			}
			tt.wantErr(t, dcu.UpdateCondition(testCtx, tt.doguResource, tt.condition), fmt.Sprintf("UpdateCondition(%v, %v, %v)", testCtx, tt.doguResource, tt.condition))
		})
	}
}

func TestDoguConditionUpdater_UpdateConditions(t *testing.T) {
	tests := []struct {
		name            string
		doguInterfaceFn func(t *testing.T) doguInterface
		doguResource    *v2.Dogu
		conditions      []v1.Condition
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, fn func(status v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) {
					g := gomega.NewWithT(t)
					status := fn(dogu.Status)
					g.Expect(status.Conditions).
						To(conditions.MatchConditions([]v1.Condition{
							{
								Type:    "test",
								Status:  "test",
								Reason:  "test",
								Message: "test",
							},
							{
								Type:    "test2",
								Status:  "test2",
								Reason:  "test2",
								Message: "test2",
							},
						}, conditions.IgnoreLastTransitionTime(true)))
				}).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			conditions: []v1.Condition{
				{
					Type:    "test",
					Status:  "test",
					Reason:  "test",
					Message: "test",
				},
				{
					Type:    "test2",
					Status:  "test2",
					Reason:  "test2",
					Message: "test2",
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "should succeed to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, fn func(status v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) {
					g := gomega.NewWithT(t)
					status := fn(dogu.Status)
					g.Expect(status.Conditions).
						To(conditions.MatchConditions([]v1.Condition{
							{
								Type:    "test",
								Status:  "test",
								Reason:  "test",
								Message: "test",
							},
							{
								Type:    "test2",
								Status:  "test2",
								Reason:  "test2",
								Message: "test2",
							},
						}, conditions.IgnoreLastTransitionTime(true)))
				}).Return(&v2.Dogu{}, nil)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			conditions: []v1.Condition{
				{
					Type:    "test",
					Status:  "test",
					Reason:  "test",
					Message: "test",
				},
				{
					Type:    "test2",
					Status:  "test2",
					Reason:  "test2",
					Message: "test2",
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dcu := &DoguConditionUpdater{
				doguInterface: tt.doguInterfaceFn(t),
			}
			tt.wantErr(t, dcu.UpdateConditions(testCtx, tt.doguResource, tt.conditions), fmt.Sprintf("UpdateConditions(%v, %v, %v)", testCtx, tt.doguResource, tt.conditions))
		})
	}
}
