package doguv3

import (
	"context"
	"errors"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	namespace   string
	recorder    record.EventRecorder
	doguClient  K8sClient
	requeueTime time.Duration
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguClient K8sClient, recorder record.EventRecorder, operatorConfig *config.OperatorConfig) *doguRequeueHandler {
	return &doguRequeueHandler{
		doguClient:  doguClient,
		namespace:   operatorConfig.Namespace,
		recorder:    recorder,
		requeueTime: operatorConfig.RequeueTimeForDoguReconciler,
	}
}

// Handle acts on changes of the provided dogu v3 resource.
func (d *doguRequeueHandler) Handle(ctx context.Context, doguResource *v3beta1.Dogu, namespacedName types.NamespacedName) (result ctrl.Result, requeueErr error) {
	return ctrl.Result{}, errors.New("not implemented")
}
