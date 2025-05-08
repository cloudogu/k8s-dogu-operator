package controllers

import (
	"context"
	"fmt"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
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

type DoguSecurityContextManager struct {
	localDoguFetcher  localDoguFetcher
	resourceUpserter  resource.ResourceUpserter
	securityValidator securityValidator
	recorder          eventRecorder
}

// NewDoguSecurityContextManager creates a new *DoguSecurityContextManager.
func NewDoguSecurityContextManager(mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *DoguSecurityContextManager {
	return &DoguSecurityContextManager{
		localDoguFetcher:  mgrSet.LocalDoguFetcher,
		resourceUpserter:  mgrSet.ResourceUpserter,
		securityValidator: mgrSet.SecurityValidator,
		recorder:          eventRecorder,
	}
}

// UpdateDeploymentWithSecurityContext regenerates the security context of a dogu deployment.
func (d DoguSecurityContextManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *doguv2.Dogu) error {
	logger := log.FromContext(ctx)

	logger.Info("Getting local dogu descriptor...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Getting local dogu descriptor...")
	dogu, err := d.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err)
	}

	logger.Info("Validating dogu security...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Validating dogu security...")
	err = d.securityValidator.ValidateSecurity(dogu, doguResource)
	if err != nil {
		return fmt.Errorf("validation of security context failed for dogu %q: %w", doguResource.Name, err)
	}

	logger.Info("Upserting deployment...")
	d.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Upserting deployment...")
	_, err = d.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment with security context for dogu %q: %w", doguResource.Name, err)
	}

	return nil
}
