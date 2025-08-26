package postinstall

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterReplicasStep = 5 * time.Second

type ReplicasStep struct {
	deploymentInterface deploymentInterface
	deploymentPatcher   *steps.DeploymentPatcher
	client              client.Client
}

func NewReplicasStep(client client.Client, mgrSet *util.ManagerSet, namespace string) *ReplicasStep {
	deploymentInt := mgrSet.ClientSet.AppsV1().Deployments(namespace)
	return &ReplicasStep{
		deploymentInterface: deploymentInt,
		client:              client,
		deploymentPatcher:   steps.NewDeploymentPatcher(deploymentInt),
	}
}

func (rs *ReplicasStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	scale, err := rs.deploymentInterface.GetScale(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	shouldBeStopped, err := rs.shouldBeStopped(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if shouldBeStopped && scale.Spec.Replicas == 0 || !shouldBeStopped && scale.Spec.Replicas == 1 {
		if doguResource.Spec.Stopped {
			// do not reconcile if the dogu was stopped manually
			return steps.Abort()
		}
		return steps.Continue()
	}

	scale.Spec.Replicas = 1
	if shouldBeStopped {
		scale.Spec.Replicas = 0
	}

	_, err = rs.deploymentInterface.UpdateScale(ctx, doguResource.Name, scale, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.RequeueAfter(requeueAfterReplicasStep)
}

func (rs *ReplicasStep) isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
	// Longhorn works this way and does not add the Condition "FileSystemResizePending" to the PVC
	// see https://github.com/longhorn/longhorn/issues/2749
	isRequestedCapacityAvailable := pvc.Status.Capacity.Storage().Value() >= quantity.Value()
	return isRequestedCapacityAvailable
}

func (rs *ReplicasStep) shouldBeStopped(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	pvc, err := doguResource.GetDataPVC(ctx, rs.client)
	if err != nil {
		return false, err
	}

	quantity, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		return false, err
	}

	return !rs.isPvcStorageResized(pvc, quantity) || doguResource.Spec.Stopped, nil
}
