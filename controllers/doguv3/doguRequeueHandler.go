package doguv3

import (
	"context"
	"time"

	beta1 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	namespace     string
	recorder      record.EventRecorder
	doguInterface v3beta1.DoguInterface
	requeueTime   time.Duration
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguInterface v3beta1.DoguV3beta1Interface, recorder record.EventRecorder, operatorConfig *config.OperatorConfig) *doguRequeueHandler {
	return &doguRequeueHandler{
		// TODO HELP! Which client implements this doguInterface?
		doguInterface: nil,
		namespace:     operatorConfig.Namespace,
		recorder:      recorder,
		requeueTime:   operatorConfig.RequeueTimeForDoguReconciler,
	}
}

func (d *doguRequeueHandler) Handle(ctx context.Context, doguResource *beta1.Dogu, err error, reqTime time.Duration) (result ctrl.Result, requeueErr error) {
	//TODO implement me
	panic("Whoo, Dogu v3 Requeue Handler was called")
}
