package controllers

import (
	"context"
	"fmt"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// SecurityContextChangeEventReason is the reason string for firing security context change events.
	SecurityContextChangeEventReason = "SecurityContextChange"
	// ErrorOnSecurityContextChangeEventReason is the error string for firing security context change error events.
	ErrorOnSecurityContextChangeEventReason = "ErrSecurityContextChange"
)

type doguSecurityContextManager struct {
	doguResourceGenerator resource.DoguResourceGenerator
	resourceDoguFetcher   resourceDoguFetcher
	resourceUpserter      resource.ResourceUpserter
	client                client.Client
	eventRecorder         record.EventRecorder
}

func NewDoguSecurityContextManager(k8sClient client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *doguSecurityContextManager {
	return &doguSecurityContextManager{
		doguResourceGenerator: mgrSet.DoguResourceGenerator,
		resourceUpserter:      mgrSet.ResourceUpserter,
		client:                k8sClient,
		eventRecorder:         eventRecorder,
	}
}

func (d doguSecurityContextManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)
	logger.Info("Fetching dogu...")
	dogu, _, err := d.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Upserting deployment... ")
	_, err = d.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment with security context: %w", err)
	}
	return nil
}
