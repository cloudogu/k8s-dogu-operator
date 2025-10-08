package upgrade

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testTime = metav1.Time{Time: time.Unix(112313, 0)}

func TestNewPreUpgradeStatusStep(t *testing.T) {
	step := NewPreUpgradeStatusStep(newMockUpgradeChecker(t), newMockDoguInterface(t))
	assert.NotEmpty(t, step)
}

func TestPreUpgradeStatusStep_Run(t *testing.T) {
	type fields struct {
		upgradeCheckerFn func(t *testing.T) upgradeChecker
		doguInterfaceFn  func(t *testing.T) doguInterface
	}
	tests := []struct {
		name     string
		fields   fields
		resource *v2.Dogu
		want     steps.StepResult
	}{
		{
			name: "should fail to check for upgrade",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				upgradeCheckerFn: func(t *testing.T) upgradeChecker {
					mck := newMockUpgradeChecker(t)
					mck.EXPECT().IsUpgrade(testCtx, &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}}).Return(false, assert.AnError)
					return mck
				},
			},
			resource: &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}},
			want:     steps.StepResult{Err: fmt.Errorf("failed to check if dogu is upgrading: %w", assert.AnError)},
		},
		{
			name: "should do nothing and continue if not upgrade",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				upgradeCheckerFn: func(t *testing.T) upgradeChecker {
					mck := newMockUpgradeChecker(t)
					mck.EXPECT().IsUpgrade(testCtx, &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}}).Return(false, nil)
					return mck
				},
			},
			resource: &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}},
			want:     steps.StepResult{Continue: true},
		},
		{
			name: "should fail on updating status on upgrade",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					expectedDogu := &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}, Status: v2.DoguStatus{
						Status: "upgrading",
						Health: "unavailable",
						Conditions: []metav1.Condition{
							{
								Type:               "healthy",
								Status:             metav1.ConditionFalse,
								Reason:             "Upgrading",
								Message:            "The spec version differs from the installed version, therefore an upgrade was scheduled.",
								LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
							},
							{
								Type:               "ready",
								Status:             metav1.ConditionFalse,
								Reason:             "Upgrading",
								Message:            "The spec version differs from the installed version, therefore an upgrade was scheduled.",
								LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
							},
						},
					}}
					mck.EXPECT().UpdateStatusWithRetry(testCtx, expectedDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				upgradeCheckerFn: func(t *testing.T) upgradeChecker {
					mck := newMockUpgradeChecker(t)
					mck.EXPECT().IsUpgrade(testCtx, &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}}).Return(true, nil)
					return mck
				},
			},
			resource: &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}},
			want:     steps.StepResult{Err: fmt.Errorf("failed to update dogu status before upgrade: %w", assert.AnError)},
		},
		{
			name: "should succeed on updating status on upgrade",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					expectedDogu := &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}, Status: v2.DoguStatus{
						Status: "upgrading",
						Health: "unavailable",
						Conditions: []metav1.Condition{
							{
								Type:               "healthy",
								Status:             metav1.ConditionFalse,
								Reason:             "Upgrading",
								Message:            "The spec version differs from the installed version, therefore an upgrade was scheduled.",
								LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
							},
							{
								Type:               "ready",
								Status:             metav1.ConditionFalse,
								Reason:             "Upgrading",
								Message:            "The spec version differs from the installed version, therefore an upgrade was scheduled.",
								LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
							},
						},
					}}
					mck.EXPECT().UpdateStatusWithRetry(testCtx, expectedDogu, mock.Anything, metav1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
						status := modifyStatusFn(dogu.Status)
						assert.Equal(t, expectedDogu.Status, status)
					}).Return(nil, nil)
					return mck
				},
				upgradeCheckerFn: func(t *testing.T) upgradeChecker {
					mck := newMockUpgradeChecker(t)
					mck.EXPECT().IsUpgrade(testCtx, &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}}).Return(true, nil)
					return mck
				},
			},
			resource: &v2.Dogu{Spec: v2.DoguSpec{Name: "test"}},
			want:     steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldNow := steps.Now
			defer func() { steps.Now = oldNow }()
			steps.Now = func() metav1.Time {
				return testTime
			}

			p := &PreUpgradeStatusStep{
				upgradeChecker: tt.fields.upgradeCheckerFn(t),
				doguInterface:  tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, p.Run(testCtx, tt.resource), "Run(%v, %v)", testCtx, tt.resource)
		})
	}
}
