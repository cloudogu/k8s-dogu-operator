package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"k8s.io/client-go/tools/record"
)

type ExportModeStep struct {
	exportManager exportManager
}

func NewExportModeStep(mgrSet *util.ManagerSet, namespace string, eventRecorder record.EventRecorder) *ExportModeStep {
	doguInterface := mgrSet.EcosystemClient.Dogus(namespace)
	exportManager := manager.NewDoguExportManager(
		doguInterface,
		mgrSet.ClientSet.CoreV1().Pods(namespace),
		mgrSet.ClientSet.AppsV1().Deployments(namespace),
		mgrSet.ResourceUpserter,
		mgrSet.LocalDoguFetcher,
		eventRecorder,
	)
	return &ExportModeStep{
		exportManager: exportManager,
	}
}

func (ems *ExportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := ems.exportManager.UpdateExportMode(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
