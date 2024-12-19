package controllers

import (
	"context"
	"fmt"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// SecurityContextChangeEventReason is the reason string for firing security context change events.
	SecurityContextChangeEventReason = "SecurityContextChange"
	// ErrorOnSecurityContextChangeEventReason is the error string for firing security context change error events.
	ErrorOnSecurityContextChangeEventReason = "ErrSecurityContextChange"
)

type doguSecurityContextManager struct {
	resourceDoguFetcher resourceDoguFetcher
	resourceUpserter    resource.ResourceUpserter
}

func NewDoguSecurityContextManager(mgrSet *util.ManagerSet) *doguSecurityContextManager {
	return &doguSecurityContextManager{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		resourceUpserter:    mgrSet.ResourceUpserter,
	}
}

func (d doguSecurityContextManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)
	logger.Info("Fetching dogu...")
	dogu, _, err := d.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to fetch dogu %s: %w", doguResource.Spec.Name, err)
	}

	logger.Info("Upserting deployment... ")
	_, err = d.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment with security context: %w", err)
	}
	return nil
}
