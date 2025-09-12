package postinstall

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
)

func TestNewExportModeStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		podInterfaceMock := newMockPodInterface(t)
		appV1InterfaceMock := newMockAppV1Interface(t)
		coreV1InterfaceMock := newMockCoreV1Interface(t)
		clientSetMock := newMockClientSet(t)
		appV1InterfaceMock.EXPECT().Deployments(namespace).Return(deploymentInterfaceMock)
		coreV1InterfaceMock.EXPECT().Pods(namespace).Return(podInterfaceMock)
		clientSetMock.EXPECT().AppsV1().Return(appV1InterfaceMock)
		clientSetMock.EXPECT().CoreV1().Return(coreV1InterfaceMock)

		doguInterfaceMock := newMockDoguInterface(t)
		ecosystemInterfaceMock := newMockEcosystemInterface(t)
		ecosystemInterfaceMock.EXPECT().Dogus(namespace).Return(doguInterfaceMock)

		eventRecorderMock := newMockEventRecorder(t)
		step := NewExportModeStep(
			&util.ManagerSet{
				ClientSet:        clientSetMock,
				EcosystemClient:  ecosystemInterfaceMock,
				LocalDoguFetcher: newMockLocalDoguFetcher(t),
				ResourceUpserter: newMockResourceUpserter(t),
			},
			namespace,
			eventRecorderMock,
		)

		assert.NotNil(t, step)
	})
}

func TestExportModeStep_Run(t *testing.T) {
	tests := []struct {
		name            string
		exportManagerFn func(t *testing.T) exportManager
		doguResource    *v2.Dogu
		want            steps.StepResult
	}{
		{
			name: "should fail to update export mode",
			exportManagerFn: func(t *testing.T) exportManager {
				mck := newMockExportManager(t)
				mck.EXPECT().UpdateExportMode(testCtx, &v2.Dogu{}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update export mode",
			exportManagerFn: func(t *testing.T) exportManager {
				mck := newMockExportManager(t)
				mck.EXPECT().UpdateExportMode(testCtx, &v2.Dogu{}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ems := &ExportModeStep{
				exportManager: tt.exportManagerFn(t),
			}
			assert.Equalf(t, tt.want, ems.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
