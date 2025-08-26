package install

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const requeueAfterFinalizerExists = 5 * time.Second
const finalizerName = "dogu-finalizer"

type FinalizerExistsStep struct {
	client client.Client
}

func NewFinalizerExistsStep(client client.Client) *FinalizerExistsStep {
	return &FinalizerExistsStep{
		client: client,
	}
}

func (fs *FinalizerExistsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if !controllerutil.ContainsFinalizer(doguResource, finalizerName) {
		finalizers := []string{finalizerName}
		patch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"finalizers": finalizers,
			},
		}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to marshal patch for finalizer: %w", err))
		}

		err = fs.client.Patch(ctx, doguResource, client.RawPatch(types.MergePatchType, patchBytes))
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to update dogu: %w", err))
		}
	}

	return steps.Continue()
}
