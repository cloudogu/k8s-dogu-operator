package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewHealthCheckStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguInterfaceMock := newMockDoguInterface(t)
		ecoSystemV2InterfaceMock := newMockEcoSystemV2Interface(t)
		ecoSystemV2InterfaceMock.EXPECT().Dogus(namespace).Return(doguInterfaceMock)
		step := NewHealthCheckStep(
			newMockK8sClient(t),
			newMockDeploymentAvailabilityChecker(t),
			newMockDoguHealthStatusUpdater(t),
			&util.ManagerSet{
				LocalDoguFetcher: newMockLocalDoguFetcher(t),
				EcosystemClient:  ecoSystemV2InterfaceMock,
			},
			namespace,
		)

		assert.NotNil(t, step)
	})
}

func TestHealthCheckStep_Run(t *testing.T) {
	type fields struct {
		clientFn                  func(t *testing.T) k8sClient
		availabilityCheckerFn     func(t *testing.T) deploymentAvailabilityChecker
		doguHealthStatusUpdaterFn func(t *testing.T) doguHealthStatusUpdater
		doguFetcherFn             func(t *testing.T) localDoguFetcher
		doguInterfaceFn           func(t *testing.T) doguInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *doguv2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(assert.AnError)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					return newMockDeploymentAvailabilityChecker(t)
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					return newMockDoguHealthStatusUpdater(t)
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get deployment for dogu %s: %w", "test", assert.AnError)),
		},
		{
			name: "should not find deployment and continue",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					return newMockDeploymentAvailabilityChecker(t)
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					return newMockDoguHealthStatusUpdater(t)
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to get dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(nil)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					mck := newMockDeploymentAvailabilityChecker(t)
					mck.EXPECT().IsAvailable(&v2.Deployment{}).Return(true)
					return mck
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					return newMockDoguHealthStatusUpdater(t)
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get current dogu json to update health state configMap: %w", assert.AnError)),
		},
		{
			name: "should fail to update health config map",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(nil)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					mck := newMockDeploymentAvailabilityChecker(t)
					mck.EXPECT().IsAvailable(&v2.Deployment{}).Return(true)
					return mck
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					mck := newMockDoguHealthStatusUpdater(t)
					mck.EXPECT().UpdateHealthConfigMap(testCtx, &v2.Deployment{}, &cesappcore.Dogu{}).Return(assert.AnError)
					return mck
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to update health state configMap: %w", assert.AnError)),
		},
		{
			name: "should fail to get dogu resource",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(nil)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					mck := newMockDeploymentAvailabilityChecker(t)
					mck.EXPECT().IsAvailable(&v2.Deployment{}).Return(true)
					return mck
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					mck := newMockDoguHealthStatusUpdater(t)
					mck.EXPECT().UpdateHealthConfigMap(testCtx, &v2.Deployment{}, &cesappcore.Dogu{}).Return(nil)
					return mck
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update dogu resource",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(nil)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					mck := newMockDeploymentAvailabilityChecker(t)
					mck.EXPECT().IsAvailable(&v2.Deployment{}).Return(true)
					return mck
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					mck := newMockDoguHealthStatusUpdater(t)
					mck.EXPECT().UpdateHealthConfigMap(testCtx, &v2.Deployment{}, &cesappcore.Dogu{}).Return(nil)
					return mck
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace,
							Name:      "test",
						},
					}, nil)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace,
							Name:      "test",
						},
						Status: doguv2.DoguStatus{
							Health: "available",
							Conditions: []v1.Condition{
								{
									Type:               doguv2.ConditionHealthy,
									Status:             v1.ConditionTrue,
									Reason:             "DoguIsHealthy",
									Message:            "All replicas are available",
									LastTransitionTime: v1.Now().Rfc3339Copy(),
								},
							},
						},
					}, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update dogu resource",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: "test"}, &v2.Deployment{}).Return(nil)
					return mck
				},
				availabilityCheckerFn: func(t *testing.T) deploymentAvailabilityChecker {
					mck := newMockDeploymentAvailabilityChecker(t)
					mck.EXPECT().IsAvailable(&v2.Deployment{}).Return(true)
					return mck
				},
				doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
					mck := newMockDoguHealthStatusUpdater(t)
					mck.EXPECT().UpdateHealthConfigMap(testCtx, &v2.Deployment{}, &cesappcore.Dogu{}).Return(nil)
					return mck
				},
				doguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace,
							Name:      "test",
						},
					}, nil)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace,
							Name:      "test",
						},
						Status: doguv2.DoguStatus{
							Health: "available",
							Conditions: []v1.Condition{
								{
									Type:               doguv2.ConditionHealthy,
									Status:             v1.ConditionTrue,
									Reason:             "DoguIsHealthy",
									Message:            "All replicas are available",
									LastTransitionTime: v1.Now().Rfc3339Copy(),
								},
							},
						},
					}, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &doguv2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcs := &HealthCheckStep{
				client:                  tt.fields.clientFn(t),
				availabilityChecker:     tt.fields.availabilityCheckerFn(t),
				doguHealthStatusUpdater: tt.fields.doguHealthStatusUpdaterFn(t),
				doguFetcher:             tt.fields.doguFetcherFn(t),
				doguInterface:           tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, hcs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
