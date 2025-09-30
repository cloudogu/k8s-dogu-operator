package upgrade

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewRevertStartupProbeStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRevertStartupProbeStep(
			newMockK8sClient(t),
			nil,
			nil,
			newMockCommandExecutor(t),
		)

		assert.NotNil(t, step)
	})
}

func TestRevertStartupProbeStep_Run(t *testing.T) {
	type fields struct {
		clientFn              func(t *testing.T) k8sClient
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
		doguCommandExecutorFn func(t *testing.T) commandExecutor
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch remote dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					return newMockDeploymentInterface(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(nil, assert.AnError)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should fail to fetch dogu deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(nil, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch deployment: %w", assert.AnError)),
		},
		{
			name: "should continue if startup probe has default value",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "test",
										},
									},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{Name: "official/test"}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
		{
			name: "should fail to revert startup probe",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:         "test",
											StartupProbe: &v1.Probe{},
										},
									},
								},
							},
						},
					}, nil)
					mck.EXPECT().Update(testCtx, &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:         "test",
											StartupProbe: nil,
										},
									},
								},
							},
						},
					}, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{Name: "official/test"}, nil)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "official/test"}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to revert startup probe",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:         "test",
											StartupProbe: &v1.Probe{},
										},
									},
								},
							},
						},
					}, nil)
					mck.EXPECT().Update(testCtx, &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:         "test",
											StartupProbe: nil,
										},
									},
								},
							},
						},
					}, metav1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{Name: "official/test"}, nil)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "official/test"}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueAfter(requeueAfterRevertStartupProbe),
		},
		{
			name: "should fail to get pod on executing post upgrade script",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					labels := client.MatchingLabels{
						"dogu.name":    "test",
						"dogu.version": "",
					}
					mck.EXPECT().List(testCtx, &v1.PodList{}, labels).Return(assert.AnError)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPostUpgrade,
									Command:     "",
									Description: "",
								},
							},
						}, nil)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "official/test"}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("post-upgrade failed: %w", fmt.Errorf("failed to get new %s pod for post upgrade: %w", "test", fmt.Errorf("failed to get pods: %w", assert.AnError)))),
		},
		{
			name: "should fail to get installed dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPostUpgrade,
									Command:     "",
									Description: "",
								},
							},
						}, nil)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch installed dogu: %w", assert.AnError)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsps := &RevertStartupProbeStep{
				client:              tt.fields.clientFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				doguCommandExecutor: tt.fields.doguCommandExecutorFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, rsps.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
