// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	appsv1 "k8s.io/api/apps/v1"

	core "github.com/cloudogu/cesapp/v4/core"

	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	pkgv1 "github.com/google/go-containerregistry/pkg/v1"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguResourceGenerator is an autogenerated mock type for the DoguResourceGenerator type
type DoguResourceGenerator struct {
	mock.Mock
}

// GetDoguDeployment provides a mock function with given fields: doguResource, dogu
func (_m *DoguResourceGenerator) GetDoguDeployment(doguResource *v1.Dogu, dogu *core.Dogu) *appsv1.Deployment {
	ret := _m.Called(doguResource, dogu)

	var r0 *appsv1.Deployment
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu) *appsv1.Deployment); ok {
		r0 = rf(doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
		}
	}

	return r0
}

// GetDoguService provides a mock function with given fields: doguResource, imageConfig
func (_m *DoguResourceGenerator) GetDoguService(doguResource *v1.Dogu, imageConfig *pkgv1.ConfigFile) (*corev1.Service, error) {
	ret := _m.Called(doguResource, imageConfig)

	var r0 *corev1.Service
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *pkgv1.ConfigFile) *corev1.Service); ok {
		r0 = rf(doguResource, imageConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu, *pkgv1.ConfigFile) error); ok {
		r1 = rf(doguResource, imageConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
