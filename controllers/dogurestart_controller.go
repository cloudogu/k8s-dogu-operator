package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/v2/api/v1"
)

// DoguRestartReconciler reconciles a DoguRestart object
type DoguRestartReconciler struct {
	doguInterface        cloudogu.DoguInterface
	doguRestartInterface cloudogu.DoguRestartInterface
	garbageCollector     DoguRestartGarbageCollector
	recorder             record.EventRecorder
}

type DoguRestartGarbageCollector interface {
	DoGarbageCollection(ctx context.Context, doguName string) error
}

func NewDoguRestartReconciler(doguRestartInterface ecoSystem.DoguRestartInterface, doguInterface ecoSystem.DoguInterface, recorder record.EventRecorder, gc DoguRestartGarbageCollector) *DoguRestartReconciler {
	return &DoguRestartReconciler{doguRestartInterface: doguRestartInterface, doguInterface: doguInterface, recorder: recorder, garbageCollector: gc}
}

// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *DoguRestartReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instruction := r.createRestartInstruction(ctx, req)

	result, err := instruction.execute(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to execute restart instruction for dogurestart %q: %w", req.NamespacedName, err)
	}

	// If the restart is in progress there is no need to collect garbage. Only on terminated objects (failed or successful restarts).
	// Garbage collection is not implement as a restart operation because the process does not belong to a single restart.
	if !result.Requeue && result.RequeueAfter == 0 && instruction.restart != nil {
		err = r.garbageCollector.DoGarbageCollection(ctx, instruction.restart.Spec.DoguName)
		if err != nil {
			return result, fmt.Errorf("failed to do garbagecollection for dogurestart %q: %w", req.NamespacedName, err)
		}
	}

	return result, nil
}

func (r *DoguRestartReconciler) createRestartInstruction(ctx context.Context, req ctrl.Request) (instruction restartInstruction) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.createRestartInstruction").
		WithValues("doguRestart", req.NamespacedName)

	instruction.req = req
	instruction.doguRestartInterface = r.doguRestartInterface
	instruction.doguInterface = r.doguInterface
	instruction.recorder = r.recorder

	doguRestart, err := r.doguRestartInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		instruction.op = handleGetDoguRestartFailed
		instruction.err = err
		return
	}
	logger.Info("dogu restart ressource has been found")
	instruction.restart = doguRestart

	if instruction.op = RestartOperationFromRestartStatusPhase(doguRestart.Status.Phase); instruction.op == ignore {
		// early exit to prevent unnecessary dogu loading
		return
	}

	dogu, err := r.doguInterface.Get(ctx, doguRestart.Spec.DoguName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			instruction.op = handleDoguNotFound
			instruction.err = err
			return
		}
		instruction.op = handleGetDoguFailed
		instruction.err = err
		return
	}

	instruction.dogu = dogu

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoguRestartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.DoguRestart{}).
		Complete(r)
}
