package install

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

type CustomK8sResourceStep struct {
	recorder         record.EventRecorder
	localDoguFetcher localDoguFetcher
	execPodFactory   exec.ExecPodFactory
	fileExtractor    exec.FileExtractor
	collectApplier   resource.CollectApplier
}

func NewCustomK8sResourceStep(mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *CustomK8sResourceStep {
	return &CustomK8sResourceStep{
		recorder:         eventRecorder,
		localDoguFetcher: mgrSet.LocalDoguFetcher,
		execPodFactory:   mgrSet.ExecPodFactory,
		fileExtractor:    mgrSet.FileExtractor,
		collectApplier:   mgrSet.CollectApplier,
	}
}

func (ses *CustomK8sResourceStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := ses.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor"))
	}

	execPodExists := ses.execPodFactory.Exists(ctx, doguResource, dogu)
	if !execPodExists {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(nil)
	}

	err = ses.execPodFactory.CheckReady(ctx, doguResource, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to check if exec pod is ready: %w", err))
	}

	customK8sResources, err := ses.fileExtractor.ExtractK8sResourcesFromExecPod(ctx, doguResource, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to extract customK8sResources: %w", err))
	}

	if len(customK8sResources) > 0 {
		ses.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", util.GetMapKeysAsString(customK8sResources))
	}
	err = ses.collectApplier.CollectApply(ctx, customK8sResources, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to apply customK8sResources: %w", err))
	}

	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(nil)
}
