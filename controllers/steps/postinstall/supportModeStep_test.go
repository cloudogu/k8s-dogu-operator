package postinstall

import (
	"fmt"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testTime = metav1.Time{Time: time.Unix(112313, 0)}

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
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to set support mode condition",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
						Status: doguv2.DoguStatus{
							Conditions: []metav1.Condition{
								{
									Type:               doguv2.ConditionSupportMode,
									Status:             metav1.ConditionFalse,
									Reason:             ReasonSupportModeInactive,
									Message:            "The Support mode is inactive",
									LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
								},
							},
						},
					}, metav1.UpdateOptions{}).Return(&doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, assert.AnError)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to set support mode condition",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					deployment := &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{
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
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(deployment, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
						Status: doguv2.DoguStatus{
							Conditions: []metav1.Condition{
								{
									Type:               doguv2.ConditionSupportMode,
									Status:             metav1.ConditionFalse,
									Reason:             ReasonSupportModeInactive,
									Message:            "The Support mode is inactive",
									LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
								},
							},
						},
					}, metav1.UpdateOptions{}).Return(&doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
		{
			name: "should succeed to set support mode condition if deployment in support mode",
			fields: fields{
				supportManagerFn: func(t *testing.T) supportManager {
					mck := newMockSupportManager(t)
					mck.EXPECT().HandleSupportMode(testCtx, &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(false, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					deployment := &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{
												{
													Name:  SupportModeEnvVar,
													Value: "true",
												},
											},
										},
									},
								},
							},
						},
					}
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(deployment, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
						Status: doguv2.DoguStatus{
							Health: "unavailable",
							Conditions: []metav1.Condition{
								{
									Type:               doguv2.ConditionHealthy,
									Status:             metav1.ConditionFalse,
									Reason:             "SupportModeActive",
									Message:            "The Support mode is active",
									LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
								},
								{
									Type:               doguv2.ConditionSupportMode,
									Status:             metav1.ConditionTrue,
									Reason:             "SupportModeActive",
									Message:            "The Support mode is active",
									LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
								},
							},
						},
					}, metav1.UpdateOptions{}).Return(&doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.Abort(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldNow := steps.Now
			defer func() { steps.Now = oldNow }()
			steps.Now = func() metav1.Time {
				return testTime
			}

			sms := &SupportModeStep{
				supportManager:      tt.fields.supportManagerFn(t),
				doguInterface:       tt.fields.doguInterfaceFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, sms.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
