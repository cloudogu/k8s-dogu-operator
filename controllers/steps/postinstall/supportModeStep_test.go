package postinstall

import (
	"fmt"
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/apps/v1"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSupportModeStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		manager := newMockSupportManager(t)
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)

		step := NewSupportModeStep(manager, doguInterfaceMock, deploymentInterfaceMock)

		assert.NotNil(t, step)
	})
}

func TestSupportModeStep_Run(t *testing.T) {
	type fields struct {
		supportManagerFn      func(t *testing.T) supportManager
		doguInterfaceFn       func(t *testing.T) doguInterface
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *doguv2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to handle support mode",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{}).Return(false, assert.AnError)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					return newMockDeploymentInterface(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{},
			want:         steps.RequeueWithError(fmt.Errorf("failed to handle support mode: %w", assert.AnError)),
		},
		{
			name: "should fail to get deployment",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to set support mode condition",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v2.Deployment{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: doguv2.DoguStatus{
							Conditions: []v1.Condition{
								{
									Type:               doguv2.ConditionSupportMode,
									Status:             v1.ConditionFalse,
									Reason:             ReasonSupportModeInactive,
									Message:            "The Support mode is inactive",
									LastTransitionTime: v1.Now().Rfc3339Copy(),
								},
							},
						},
					}, v1.UpdateOptions{}).Return(&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, assert.AnError)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to set support mode condition",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					deployment := &v2.Deployment{
						Spec: v2.DeploymentSpec{
							Template: v3.PodTemplateSpec{
								Spec: v3.PodSpec{
									Containers: []v3.Container{
										{
											Env: []v3.EnvVar{
												{
													Name:  SupportModeEnvVar,
													Value: "false",
												},
											},
										},
									},
								},
							},
						},
					}
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(deployment, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: doguv2.DoguStatus{
							Conditions: []v1.Condition{
								{
									Type:               doguv2.ConditionSupportMode,
									Status:             v1.ConditionFalse,
									Reason:             ReasonSupportModeInactive,
									Message:            "The Support mode is inactive",
									LastTransitionTime: v1.Now().Rfc3339Copy(),
								},
							},
						},
					}, v1.UpdateOptions{}).Return(&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sms := &SupportModeStep{
				supportManager:      tt.fields.supportManagerFn(t),
				doguInterface:       tt.fields.doguInterfaceFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, sms.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
