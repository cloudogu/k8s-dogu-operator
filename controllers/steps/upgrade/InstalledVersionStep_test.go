package upgrade

import (
	"context"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

const namespace = "ecosystem"
const name = "test"

func TestNewInstalledVersionStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguInterfaceMock := newMockDoguInterface(t)

		step := NewInstalledVersionStep(doguInterfaceMock)

		assert.NotNil(t, step)
	})
}

func TestInstalledVersionStep_Run(t *testing.T) {
	type fields struct {
		doguInterfaceFn func(t *testing.T) doguInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to update status of dogu resource",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogu := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: name},
						Spec:       v2.DoguSpec{Version: "1.0.0"},
						Status: v2.DoguStatus{
							Status:           v2.DoguStatusInstalled,
							InstalledVersion: "1.0.0",
						},
					}
					mck.EXPECT().UpdateStatus(testCtx, dogu, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: name},
				Spec:       v2.DoguSpec{Version: "1.0.0"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update status of dogu resource",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogu := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: name},
						Spec:       v2.DoguSpec{Version: "1.0.0"},
						Status: v2.DoguStatus{
							Status:           v2.DoguStatusInstalled,
							InstalledVersion: "1.0.0",
						},
					}
					mck.EXPECT().UpdateStatus(testCtx, dogu, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: name},
				Spec:       v2.DoguSpec{Version: "1.0.0"},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ivs := &InstalledVersionStep{
				doguInterface: tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, ivs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
