package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

const podTemplateVersionKey = "dogu.version"
const InstallEventReason = "Installation"

// The CreateExecPodStep creates an exec pod if a dogu is in the installing or upgrading process.
// The exec pod will not be created if the dogu ist stopped with the `stopped` flag.
type CreateExecPodStep struct {
	client           k8sClient
	recorder         eventRecorder
	localDoguFetcher localDoguFetcher
	execPodFactory   execPodFactory
}

func NewCreateExecPodStep(client client.Client, eventRecorder record.EventRecorder, fetcher cesregistry.LocalDoguFetcher, factory exec.ExecPodFactory) *CreateExecPodStep {
	return &CreateExecPodStep{
		client:           client,
		recorder:         eventRecorder,
		localDoguFetcher: fetcher,
		execPodFactory:   factory,
	}
}

func (epcs *CreateExecPodStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, epcs.client)
	if client.IgnoreNotFound(err) != nil {
		return steps.RequeueWithError(err)
	}

	// only create the exec pod on installation or upgrade
	if doguResource.Spec.Stopped || (!errors.IsNotFound(err) && deployment.Spec.Template.Labels[podTemplateVersionKey] == doguResource.Spec.Version) {
		return steps.Continue()
	}

	dogu, err := epcs.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("dogu not found in local registry: %w", err))
	}

	epcs.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
	err = epcs.execPodFactory.Create(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to create execPod for dogu %q: %w", dogu.GetSimpleName(), err))
	}

	return steps.Continue()
}
