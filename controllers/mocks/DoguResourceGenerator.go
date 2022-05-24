// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	appsv1 "k8s.io/api/apps/v1"

	core "github.com/cloudogu/cesapp-lib/core"

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
func (_m *DoguResourceGenerator) GetDoguDeployment(doguResource *v1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	ret := _m.Called(doguResource, dogu)

	var r0 *appsv1.Deployment
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu) *appsv1.Deployment); ok {
		r0 = rf(doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDoguExposedServices provides a mock function with given fields: doguResource, dogu
func (_m *DoguResourceGenerator) GetDoguExposedServices(doguResource *v1.Dogu, dogu *core.Dogu) ([]corev1.Service, error) {
	ret := _m.Called(doguResource, dogu)

	var r0 []corev1.Service
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu) []corev1.Service); ok {
		r0 = rf(doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]corev1.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDoguPVC provides a mock function with given fields: doguResource
func (_m *DoguResourceGenerator) GetDoguPVC(doguResource *v1.Dogu) (*corev1.PersistentVolumeClaim, error) {
	ret := _m.Called(doguResource)

	var r0 *corev1.PersistentVolumeClaim
	if rf, ok := ret.Get(0).(func(*v1.Dogu) *corev1.PersistentVolumeClaim); ok {
		r0 = rf(doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeClaim)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu) error); ok {
		r1 = rf(doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDoguSecret provides a mock function with given fields: doguResource, stringData
func (_m *DoguResourceGenerator) GetDoguSecret(doguResource *v1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	ret := _m.Called(doguResource, stringData)

	var r0 *corev1.Secret
	if rf, ok := ret.Get(0).(func(*v1.Dogu, map[string]string) *corev1.Secret); ok {
		r0 = rf(doguResource, stringData)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu, map[string]string) error); ok {
		r1 = rf(doguResource, stringData)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
