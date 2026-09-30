package install

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
)

type doguPatchAssembler interface {
	Assemble(ctx context.Context, cr *v3beta1.Dogu, patchTpl []byte) (values3.Values, error)
}
