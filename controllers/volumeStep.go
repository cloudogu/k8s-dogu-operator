package controllers

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterVolume = 10 * time.Second

type VolumeStep struct {
	doguVolumeManager *doguVolumeManager
	client            client.Client
}

func NewVolumeStep(client client.Client, eventRecorder record.EventRecorder, doguInterface doguClient.DoguInterface) *VolumeStep {
	return &VolumeStep{
		client:            client,
		doguVolumeManager: NewDoguVolumeManager(client, eventRecorder, doguInterface),
	}
}

func (vs *VolumeStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {

	pvc, err := doguResource.GetDataPVC(ctx, vs.client)
	if err != nil {
		return requeueAfterVolume, err
	}
	size, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		return requeueAfterVolume, err
	}
	if pvc.Status.Capacity[corev1.ResourceStorage] != size {
		err = vs.doguVolumeManager.SetDoguDataVolumeSize(ctx, doguResource)
		if err != nil {
			return 0, err
		}
		return requeueAfterVolume, nil
	}
	return 0, nil
}
