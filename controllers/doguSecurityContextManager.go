package controllers

import (
	"context"
	"fmt"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
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
	securityValidator   securityValidator
	recorder            eventRecorder
}

func NewDoguSecurityContextManager(mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *doguSecurityContextManager {
	return &doguSecurityContextManager{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		resourceUpserter:    mgrSet.ResourceUpserter,
		securityValidator:   mgrSet.SecurityValidator,
		recorder:            eventRecorder,
	}
}

func (d doguSecurityContextManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)

	logger.Info("Fetching dogu...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Fetching dogu...")
	dogu, _, err := d.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to fetch dogu %s: %w", doguResource.Spec.Name, err)
	}

	logger.Info("Validating dogu security...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Validating dogu security...")
	err = d.securityValidator.ValidateSecurity(dogu, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Upserting deployment...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Upserting deployment...")
	_, err = d.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment with security context: %w", err)
	}
	return nil
}
