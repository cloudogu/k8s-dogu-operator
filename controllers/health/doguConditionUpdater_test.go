package health

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguConditionUpdater(t *testing.T) {
	t.Run("Successfully created dogu condition updater", func(t *testing.T) {
		step := NewDoguConditionUpdater(
			newMockDoguInterface(t),
		)

		assert.NotNil(t, step)
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
			name: "should fail to get dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			condition:    v1.Condition{},
			wantErr:      assert.Error,
		},
		{
			name: "should fail to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
				mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{
							{},
						},
					},
				}, v1.UpdateOptions{}).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			condition:    v1.Condition{},
			wantErr:      assert.Error,
		},
		{
			name: "should succeed to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
				mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{
							{},
						},
					},
				}, v1.UpdateOptions{}).Return(&v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{
							{},
						},
					},
				}, nil)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			condition:    v1.Condition{},
			wantErr:      assert.NoError,
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
			name: "should fail to get dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			conditions:   []v1.Condition{},
			wantErr:      assert.Error,
		},
		{
			name: "should fail to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v2.Dogu{}, nil)
				mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{
					Status: v2.DoguStatus{
						Conditions: make([]v1.Condition, 1),
					},
				}, v1.UpdateOptions{}).Return(nil, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			conditions: []v1.Condition{
				{},
				{},
			},
			wantErr: assert.Error,
		},
		{
			name: "should succeed to update dogu resource",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v2.Dogu{}, nil)
				mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{}, v1.UpdateOptions{}).Return(&v2.Dogu{}, nil)
				return mck
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			conditions:   []v1.Condition{},
			wantErr:      assert.NoError,
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
