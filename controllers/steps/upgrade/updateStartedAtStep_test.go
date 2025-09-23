package upgrade

import (
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewUpdateStartedAtStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewUpdateStartedAtStep(
			newMockDoguInterface(t),
			newMockDeploymentManager(t),
		)

		assert.NotNil(t, step)
	})
}

func TestUpdateStartedAtStep_Run(t *testing.T) {
	type fields struct {
		deploymentManagerFn func(t *testing.T) deploymentManager
		doguInterfaceFn     func(t *testing.T) doguInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get last starting time",
			fields: fields{
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update last starting time in dogu resource",
			fields: fields{
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogu := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: v2.DoguStatus{
							StartedAt: v1.Time{Time: time.Time{}},
						},
					}
					mck.EXPECT().UpdateStatus(testCtx, dogu, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update last starting time in dogu resource",
			fields: fields{
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogu := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: v2.DoguStatus{
							StartedAt: v1.Time{Time: time.Time{}},
						},
					}
					mck.EXPECT().UpdateStatus(testCtx, dogu, v1.UpdateOptions{}).Return(&v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: v2.DoguStatus{
							StartedAt: v1.Time{Time: time.Time{}},
						},
					}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usas := &UpdateStartedAtStep{
				deploymentManager: tt.fields.deploymentManagerFn(t),
				doguInterface:     tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, usas.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
