package garbagecollection

import (
	"context"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type doguRestartInterface interface {
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	List(ctx context.Context, opts metav1.ListOptions) (*v2.DoguRestartList, error)
}
