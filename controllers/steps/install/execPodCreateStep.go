package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const podTemplateVersionKey = "dogu.version"

type ExecPodCreateStep struct {
	client           client.Client
	recorder         record.EventRecorder
	localDoguFetcher localDoguFetcher
	execPodFactory   exec.ExecPodFactory
}

func NewExecPodCreateStep(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *ExecPodCreateStep {
	return &ExecPodCreateStep{
		client:           client,
		recorder:         eventRecorder,
		localDoguFetcher: mgrSet.LocalDoguFetcher,
		execPodFactory:   mgrSet.ExecPodFactory,
	}
}

func (epcs *ExecPodCreateStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, epcs.client)
	if client.IgnoreNotFound(err) != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}

	// only create the exec pod on installation or upgrade
	if !(errors.IsNotFound(err) || deployment.Spec.Template.Labels[podTemplateVersionKey] == doguResource.Spec.Version) {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(nil)
	}

	dogu, err := epcs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("dogu not found in local registry: %w", err))
	}

	epcs.recorder.Eventf(doguResource, corev1.EventTypeNormal, controllers.InstallEventReason, "Starting execPod...")
	err = epcs.execPodFactory.Create(ctx, doguResource, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to create execPod for dogu %q: %w", dogu.GetSimpleName(), err))
	}

	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(nil)
}
