package doguv3

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3beta1client "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	namespace     string
	recorder      record.EventRecorder
	doguInterface v3beta1client.DoguInterface
	requeueTime   time.Duration
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguInterface v3beta1client.DoguInterface, recorder record.EventRecorder, operatorConfig *config.OperatorConfig) *doguRequeueHandler {
	return &doguRequeueHandler{
		doguInterface: doguInterface,
		namespace:     operatorConfig.Namespace,
		recorder:      recorder,
		requeueTime:   operatorConfig.RequeueTimeForDoguReconciler,
	}
}

// Handle acts on changes of the provided dogu v3 resource.
func (d *doguRequeueHandler) Handle(ctx context.Context, doguResource *v3beta1.Dogu, err error, reqTime time.Duration) (result ctrl.Result, requeueErr error) {
	panic("implement me")
}
