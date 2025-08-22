package controllers

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type removeFinalizerStep struct {
	client client.Client
}

func NewRemoveFinalizerStep(client client.Client) *removeFinalizerStep {
	return &removeFinalizerStep{
		client: client,
	}
}

func (rf *removeFinalizerStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err = rf.client.Update(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to update dogu: %w", err)
	}
	return 0, nil
}
