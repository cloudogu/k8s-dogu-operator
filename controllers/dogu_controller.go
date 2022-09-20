package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type operation int

const operatorEventReason = "OperationThresholding"

const (
	InstallEventReason        = "Installation"
	ErrorOnInstallEventReason = "ErrInstallation"
)
const (
	UpgradeEventReason                      = "Upgrading"
	ErrorOnFailedPremisesUpgradeEventReason = "ErrUpgradePremises"
	ErrorOnFailedUpgradeEventReason         = "ErrUpgrade"
)
const (
	DeinstallEventReason      = "Deinstallation"
	ErrorDeinstallEventReason = "ErrDeinstallation"
)
const (
	RequeueEventReason        = "Requeue"
	ErrorOnRequeueEventReason = "ErrRequeue"
)

const (
	Install operation = iota
	Upgrade
	Delete
	Ignore
)

func (o operation) toString() string {
	switch o {
	case Install:
		return "Install"
	case Upgrade:
		return "Upgrade"
	case Delete:
		return "Delete"
	default:
		return "Ignore"
	}
}

// doguReconciler reconciles a Dogu object
type doguReconciler struct {
	client             client.Client
	doguManager        manager
	doguRequeueHandler requeueHandler
	recorder           record.EventRecorder
	fetcher            localDoguFetcher
}

// manager abstracts the simple dogu operations in a k8s CES.
type manager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// requeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type requeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error, onRequeue func(dogu *k8sv1.Dogu)) (ctrl.Result, error)
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, doguManager manager, eventRecorder record.EventRecorder, namespace string, localRegistry registry.DoguRegistry) (*doguReconciler, error) {
	doguRequeueHandler, err := NewDoguRequeueHandler(client, eventRecorder, namespace)
	if err != nil {
		return nil, err
	}

	localDoguFetcher := cesregistry.NewLocalDoguFetcher(localRegistry)
	return &doguReconciler{
		client:             client,
		doguManager:        doguManager,
		doguRequeueHandler: doguRequeueHandler,
		recorder:           eventRecorder,
		fetcher:            localDoguFetcher,
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *doguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &k8sv1.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been found", doguResource.Namespace, doguResource.Name))

	requiredOperation, err := r.evaluateRequiredOperation(ctx, doguResource)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to evaluate required operation: %w", err)
	}
	logger.Info(fmt.Sprintf("Required operation for Dogu %s/%s is: %s", doguResource.Namespace, doguResource.Name, requiredOperation.toString()))

	switch requiredOperation {
	case Install:
		installError := r.doguManager.Install(ctx, doguResource)
		contextMessageOnError := fmt.Sprintf("failed to install dogu %s", doguResource.Name)

		if installError != nil {
			printError := strings.Replace(installError.Error(), "\n", "", -1)
			r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnInstallEventReason, "Installation failed. Reason: %s.", printError)
		} else {
			r.recorder.Event(doguResource, v1.EventTypeNormal, InstallEventReason, "Installation successful.")
		}

		result, err := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, installError, func(dogu *k8sv1.Dogu) {
			doguResource.Status.Status = k8sv1.DoguStatusNotInstalled
		})
		if err != nil {
			r.recorder.Event(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason, "Failed to requeue the installation.")
			return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", err)
		}

		return result, nil
	case Upgrade:
		return r.performUpgradeOperation(ctx, doguResource)
	case Delete:
		deleteError := r.doguManager.Delete(ctx, doguResource)
		contextMessageOnError := fmt.Sprintf("failed to delete dogu %s", doguResource.Name)

		if deleteError != nil {
			printError := strings.Replace(deleteError.Error(), "\n", "", -1)
			r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorDeinstallEventReason, "Deinstallation failed. Reason: %s.", printError)
		} else {
			r.recorder.Event(doguResource, v1.EventTypeNormal, DeinstallEventReason, "Deinstallation successful.")
		}

		result, err := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, deleteError, func(dogu *k8sv1.Dogu) {
			doguResource.Status.Status = k8sv1.DoguStatusInstalled
		})
		if err != nil {
			r.recorder.Event(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason, "Failed to requeue the deinstallation.")
			return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", err)
		}

		return result, nil
	case Ignore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func (r *doguReconciler) evaluateRequiredOperation(ctx context.Context, doguResource *k8sv1.Dogu) (operation, error) {
	logger := log.FromContext(ctx)
	if !doguResource.DeletionTimestamp.IsZero() {
		return Delete, nil
	}

	switch doguResource.Status.Status {
	case k8sv1.DoguStatusNotInstalled:
		return Install, nil
	case k8sv1.DoguStatusInstalled:
		// Checking if the resource spec field has changed is unnecessary because we
		// use a predicate to filter update events where specs don't change
		upgradeable, err := checkUpgradeability(doguResource, r.fetcher)
		if err != nil {
			printError := strings.ReplaceAll(err.Error(), "\n", "")
			r.recorder.Eventf(doguResource, v1.EventTypeWarning, operatorEventReason, "Could not check if dogu needs to be upgraded: %s", printError)

			return Ignore, err
		}

		if upgradeable {
			return Upgrade, nil
		}

		return Ignore, nil
	case k8sv1.DoguStatusInstalling:
		return Ignore, nil
	case k8sv1.DoguStatusDeleting:
		return Ignore, nil
	default:
		logger.Info(fmt.Sprintf("Found unknown operation for dogu status: %s", doguResource.Status.Status))
		return Ignore, nil
	}
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		// Since we don't want to process dogus with same spec we use a generation change predicate
		// as a filter to reduce the reconcile calls.
		// The predicate implements a function that will be invoked of every update event that
		// the k8s api will fire. On writing the objects spec field the k8s api
		// increments the generation field. The function compares this field from the old
		// and new dogu resource. If they are equal the reconcile loop will not be called.
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *doguReconciler) performUpgradeOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	upgradeError := r.doguManager.Upgrade(ctx, doguResource)
	contextMessageOnError := fmt.Sprintf("failed to upgrade dogu %s", doguResource.Name)

	if upgradeError != nil {
		printError := strings.Replace(upgradeError.Error(), "\n", "", -1)
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnFailedUpgradeEventReason, "Dogu upgrade failed. Reason: %s.", printError)
	} else {
		r.recorder.Event(doguResource, v1.EventTypeNormal, UpgradeEventReason, "Dogu upgrade successful.")
	}

	result, err := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, upgradeError, func(dogu *k8sv1.Dogu) {
		// todo make the state transition more clear
		doguResource.Status.Status = k8sv1.DoguStatusInstalled
	})
	if err != nil {
		r.recorder.Event(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason, "Failed to requeue the dogu upgrade.")
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", err)
	}

	return result, nil
}

func checkUpgradeability(doguResource *k8sv1.Dogu, reg localDoguFetcher) (bool, error) {
	fromDogu, err := reg.FetchInstalled(doguResource.Name)
	if err != nil {
		return false, err
	}

	checker := &upgradeChecker{}
	toDogu := &core.Dogu{Name: doguResource.Spec.Name, Version: doguResource.Spec.Version}

	return checker.IsUpgradeable(fromDogu, toDogu, doguResource.Spec.UpgradeConfig.ForceUpgrade)
}
