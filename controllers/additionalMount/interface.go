package additionalMount

import (
	"context"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type configMapGetter interface {
	corev1.ConfigMapInterface
}

type secretGetter interface {
	corev1.SecretInterface
}

type Validator interface {
	ValidateAdditionalMounts(ctx context.Context, doguDescriptor *core.Dogu, doguResource *k8sv2.Dogu) error
}
