package main

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//nolint:unused
//goland:noinspection GoUnusedType
type ControllerManager interface {
	manager.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type DoguReconciler interface {
	controllers.DoguReconciler
}

//nolint:unused
//goland:noinspection GoUnusedType
type GenericReconciler interface {
	controllers.GenericReconciler
}
