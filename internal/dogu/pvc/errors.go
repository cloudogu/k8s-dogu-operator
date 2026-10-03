package pvc

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeShrinkError indicates that a rendered PVC requests less storage than the live PVC.
type VolumeShrinkError struct {
	PVC     client.ObjectKey
	Current resource.Quantity
	Desired resource.Quantity
}

func (e *VolumeShrinkError) Error() string {
	return fmt.Sprintf("cannot shrink PVC %q from %s to %s", e.PVC, e.Current.String(), e.Desired.String())
}

// StorageClassImmutableError indicates that a rendered PVC explicitly changes the live PVC's storage class.
type StorageClassImmutableError struct {
	PVC     client.ObjectKey
	Current *string
	Desired *string
}

func (e *StorageClassImmutableError) Error() string {
	return fmt.Sprintf("cannot change storage class of PVC %q from %s to %s", e.PVC, formatStorageClass(e.Current), formatStorageClass(e.Desired))
}

func formatStorageClass(storageClass *string) string {
	if storageClass == nil {
		return "<unset>"
	}

	return fmt.Sprintf("%q", *storageClass)
}
