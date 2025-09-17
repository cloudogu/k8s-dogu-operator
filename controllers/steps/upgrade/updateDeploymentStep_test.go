package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewUpdateDeploymentStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewUpdateDeploymentStep(
			newMockK8sClient(t),
			nil,
			newMockDeploymentInterface(t),
			newMockLocalDoguFetcher(t),
			newMockResourceDoguFetcher(t),
			newMockExecPodFactory(t),
			newMockCommandExecutor(t),
		)

		assert.NotNil(t, step)
	})
}

func TestUpdateDeploymentStep_Run(t *testing.T) {
	type fields struct {
		clientFn              func(t *testing.T) k8sClient
		upserterFn            func(t *testing.T) resourceUpserter
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
		resourceDoguFetcherFn func(t *testing.T) resourceDoguFetcher
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		execPodFactoryFn      func(t *testing.T) execPodFactory
		doguCommandExecutorFn func(t *testing.T) commandExecutor
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should continue if dogu version is in deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: ""},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
		{
			name: "should fail to fetch remote dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(nil, nil, assert.AnError)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should requeue if exec pod does not exist",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(false)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueAfter(requeueAfterUpdateDeployment),
		},
		{
			name: "should fail if exec pod is not ready",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(assert.AnError)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to check if exec pod is ready: %w", assert.AnError)),
		},
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should fail to upsert dogu deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}, mock.Anything).Return(nil, assert.AnError)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{}).Return(nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueAfter(requeueAfterUpdateDeployment),
		},
		{
			name: "should fail to get pod",
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
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", fmt.Errorf("failed to find pod for dogu %s:%s : %w", "test", "", fmt.Errorf("failed to get pods: %w", assert.AnError)))),
		},
		{
			name: "should fail to get pre-upgrade script from execpod",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					labels := client.MatchingLabels{
						"dogu.name":    "test",
						"dogu.version": "",
					}
					mck.EXPECT().List(testCtx, &v1.PodList{}, labels).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						pod := v1.Pod{}
						switch l := list.(type) {
						case *v1.PodList:
							l.Items = append(l.Items, pod)
						}
					}).Return(nil)
					return mck
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(nil)
					mck.EXPECT().Exec(
						testCtx,
						&v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPreUpgrade,
									Command:     "",
									Description: "",
								},
							},
						},
						exec.NewShellCommand("/bin/tar", "cf", "-", ""),
					).Return(nil, assert.AnError)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					return newMockCommandExecutor(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", fmt.Errorf("failed to get pre-upgrade script from execpod with command '%s', stdout: '<nil>':  %w", "/bin/tar cf - ", assert.AnError))),
		},
		{
			name: "should fail to create pre-upgrade target dir",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					labels := client.MatchingLabels{
						"dogu.name":    "test",
						"dogu.version": "",
					}
					mck.EXPECT().List(testCtx, &v1.PodList{}, labels).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						pod := v1.Pod{}
						switch l := list.(type) {
						case *v1.PodList:
							l.Items = append(l.Items, pod)
						}
					}).Return(nil)
					return mck
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(nil)
					mck.EXPECT().Exec(
						testCtx,
						&v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPreUpgrade,
									Command:     "",
									Description: "",
								},
							},
						},
						exec.NewShellCommand("/bin/tar", "cf", "-", ""),
					).Return(&bytes.Buffer{}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					mck := newMockCommandExecutor(t)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", fmt.Errorf("failed to create pre-upgrade target dir with command '/bin/mkdir -p /tmp/pre-upgrade', stdout: '<nil>': %w", assert.AnError))),
		},
		{
			name: "should fail to extract pre-upgrade script to dogu pod",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					labels := client.MatchingLabels{
						"dogu.name":    "test",
						"dogu.version": "",
					}
					mck.EXPECT().List(testCtx, &v1.PodList{}, labels).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						pod := v1.Pod{}
						switch l := list.(type) {
						case *v1.PodList:
							l.Items = append(l.Items, pod)
						}
					}).Return(nil)
					return mck
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(nil)
					mck.EXPECT().Exec(
						testCtx,
						&v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPreUpgrade,
									Command:     "",
									Description: "",
								},
							},
						},
						exec.NewShellCommand("/bin/tar", "cf", "-", ""),
					).Return(&bytes.Buffer{}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					mck := newMockCommandExecutor(t)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)).Return(&bytes.Buffer{}, nil)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommandWithStdin(&bytes.Buffer{}, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", fmt.Errorf("failed to extract pre-upgrade script to dogu pod with command '%s', stdout: '<nil>': %w", "/bin/tar xf - -C /tmp/pre-upgrade", assert.AnError))),
		},
		{
			name: "should fail to execute pre-upgrade script",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					labels := client.MatchingLabels{
						"dogu.name":    "test",
						"dogu.version": "",
					}
					mck.EXPECT().List(testCtx, &v1.PodList{}, labels).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						pod := v1.Pod{}
						switch l := list.(type) {
						case *v1.PodList:
							l.Items = append(l.Items, pod)
						}
					}).Return(nil)
					return mck
				},
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{podTemplateVersionKey: "1.0.0"},
								},
							},
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}).Return(&core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}, nil, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{
						Name: "official/test",
						ExposedCommands: []core.ExposedCommand{
							{
								Name:        core.ExposedCommandPreUpgrade,
								Command:     "",
								Description: "",
							},
						},
					}).Return(nil)
					mck.EXPECT().Exec(
						testCtx,
						&v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
						&core.Dogu{
							Name: "official/test",
							ExposedCommands: []core.ExposedCommand{
								{
									Name:        core.ExposedCommandPreUpgrade,
									Command:     "",
									Description: "",
								},
							},
						},
						exec.NewShellCommand("/bin/tar", "cf", "-", ""),
					).Return(&bytes.Buffer{}, nil)
					return mck
				},
				doguCommandExecutorFn: func(t *testing.T) commandExecutor {
					mck := newMockCommandExecutor(t)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)).Return(&bytes.Buffer{}, nil)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommandWithStdin(&bytes.Buffer{}, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)).Return(&bytes.Buffer{}, nil)
					mck.EXPECT().ExecCommandForPod(testCtx, &v1.Pod{}, exec.NewShellCommand(filepath.Join(preUpgradeScriptDir, filepath.Base("")), "", "")).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", fmt.Errorf("failed to execute '%s': output: '<nil>': %w", "/tmp/pre-upgrade  ", assert.AnError))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uds := &UpdateDeploymentStep{
				client:              tt.fields.clientFn(t),
				upserter:            tt.fields.upserterFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				resourceDoguFetcher: tt.fields.resourceDoguFetcherFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				execPodFactory:      tt.fields.execPodFactoryFn(t),
				doguCommandExecutor: tt.fields.doguCommandExecutorFn(t),
			}
			assert.Equalf(t, tt.want, uds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
