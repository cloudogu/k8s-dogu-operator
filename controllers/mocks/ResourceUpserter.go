// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import (
	context "context"

	appsv1 "k8s.io/api/apps/v1"

	core "github.com/cloudogu/cesapp-lib/core"

	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	pkgv1 "github.com/google/go-containerregistry/pkg/v1"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// ResourceUpserter is an autogenerated mock type for the ResourceUpserter type
type ResourceUpserter struct {
	mock.Mock
}

// UpsertDoguDeployment provides a mock function with given fields: ctx, doguResource, dogu, customDeployment
func (_m *ResourceUpserter) UpsertDoguDeployment(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	ret := _m.Called(ctx, doguResource, dogu, customDeployment)

	var r0 *appsv1.Deployment
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu, *appsv1.Deployment) *appsv1.Deployment); ok {
		r0 = rf(ctx, doguResource, dogu, customDeployment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *core.Dogu, *appsv1.Deployment) error); ok {
		r1 = rf(ctx, doguResource, dogu, customDeployment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpsertDoguExposedServices provides a mock function with given fields: ctx, doguResource, dogu
func (_m *ResourceUpserter) UpsertDoguExposedServices(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu) ([]*corev1.Service, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	var r0 []*corev1.Service
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) []*corev1.Service); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*corev1.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpsertDoguPVCs provides a mock function with given fields: ctx, doguResource, dogu
func (_m *ResourceUpserter) UpsertDoguPVCs(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu) (*corev1.PersistentVolumeClaim, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	var r0 *corev1.PersistentVolumeClaim
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) *corev1.PersistentVolumeClaim); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeClaim)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpsertDoguService provides a mock function with given fields: ctx, doguResource, image
func (_m *ResourceUpserter) UpsertDoguService(ctx context.Context, doguResource *v1.Dogu, image *pkgv1.ConfigFile) (*corev1.Service, error) {
	ret := _m.Called(ctx, doguResource, image)

	var r0 *corev1.Service
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *pkgv1.ConfigFile) *corev1.Service); ok {
		r0 = rf(ctx, doguResource, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *pkgv1.ConfigFile) error); ok {
		r1 = rf(ctx, doguResource, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTnewResourceUpserter interface {
	mock.TestingT
	Cleanup(func())
}

// newResourceUpserter creates a new instance of ResourceUpserter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newResourceUpserter(t mockConstructorTestingTnewResourceUpserter) *ResourceUpserter {
	mock := &ResourceUpserter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
