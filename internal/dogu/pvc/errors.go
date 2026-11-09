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

// GrowRejectedError indicates that the cluster does not allow growing the PVC, e.g. as its StorageClass does not allow volume expansion.
type GrowRejectedError struct {
	PVC client.ObjectKey
	Err error
}

func (e *GrowRejectedError) Error() string {
	return e.Err.Error()
}

func (e *GrowRejectedError) Unwrap() error {
	return e.Err
}

func formatStorageClass(storageClass *string) string {
	if storageClass == nil {
		return "<unset>"
	}

	return fmt.Sprintf("%q", *storageClass)
}
