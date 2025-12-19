package postinstall

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const storageClassMismatchReason = "StorageClassMismatch"

type MismatchedStorageClassWarningStep struct {
	client   k8sClient
	recorder eventRecorder
}

func NewMismatchedStorageClassWarningStep(client client.Client, recorder record.EventRecorder) *MismatchedStorageClassWarningStep {
	return &MismatchedStorageClassWarningStep{client: client, recorder: recorder}
}

func (m *MismatchedStorageClassWarningStep) Run(ctx context.Context, resource *v2.Dogu) steps.StepResult {
	if resource.Spec.Resources.StorageClassName == nil {
		// cannot validate storage class as it is set in the PVC by the cluster
		return steps.Continue()
	}

	pvc, err := resource.GetDataPVC(ctx, m.client)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if pvc.Spec.StorageClassName == nil {
		// storage class might not yet be set by cluster
		return steps.Continue()
	}

	if !equalPtr(pvc.Spec.StorageClassName, resource.Spec.Resources.StorageClassName) {
		message := fmt.Sprintf(
			"Mismatched storage class name between dogu and pvc resource: {dogu: %s, pvc: %s}",
			*resource.Spec.Resources.StorageClassName, *pvc.Spec.StorageClassName)
		m.recorder.Event(pvc, corev1.EventTypeWarning, storageClassMismatchReason, message)
		m.recorder.Event(resource, corev1.EventTypeWarning, storageClassMismatchReason, message)
	}

	return steps.Continue()
}

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b // both nil → true; one nil → false
	}
	return *a == *b
}
