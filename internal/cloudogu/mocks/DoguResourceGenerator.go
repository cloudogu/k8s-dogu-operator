// Code generated by mockery v2.20.0. DO NOT EDIT.

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

type DoguResourceGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguResourceGenerator) EXPECT() *DoguResourceGenerator_Expecter {
	return &DoguResourceGenerator_Expecter{mock: &_m.Mock}
}

// CreateDoguDeployment provides a mock function with given fields: doguResource, dogu
func (_m *DoguResourceGenerator) CreateDoguDeployment(doguResource *v1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	ret := _m.Called(doguResource, dogu)

	var r0 *appsv1.Deployment
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu) (*appsv1.Deployment, error)); ok {
		return rf(doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu) *appsv1.Deployment); ok {
		r0 = rf(doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguResourceGenerator_CreateDoguDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguDeployment'
type DoguResourceGenerator_CreateDoguDeployment_Call struct {
	*mock.Call
}

// CreateDoguDeployment is a helper method to define mock.On call
//   - doguResource *v1.Dogu
//   - dogu *core.Dogu
func (_e *DoguResourceGenerator_Expecter) CreateDoguDeployment(doguResource interface{}, dogu interface{}) *DoguResourceGenerator_CreateDoguDeployment_Call {
	return &DoguResourceGenerator_CreateDoguDeployment_Call{Call: _e.mock.On("CreateDoguDeployment", doguResource, dogu)}
}

func (_c *DoguResourceGenerator_CreateDoguDeployment_Call) Run(run func(doguResource *v1.Dogu, dogu *core.Dogu)) *DoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguDeployment_Call) Return(_a0 *appsv1.Deployment, _a1 error) *DoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguDeployment_Call) RunAndReturn(run func(*v1.Dogu, *core.Dogu) (*appsv1.Deployment, error)) *DoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Return(run)
	return _c
}

// CreateDoguPVC provides a mock function with given fields: doguResource
func (_m *DoguResourceGenerator) CreateDoguPVC(doguResource *v1.Dogu) (*corev1.PersistentVolumeClaim, error) {
	ret := _m.Called(doguResource)

	var r0 *corev1.PersistentVolumeClaim
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu) (*corev1.PersistentVolumeClaim, error)); ok {
		return rf(doguResource)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu) *corev1.PersistentVolumeClaim); ok {
		r0 = rf(doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeClaim)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu) error); ok {
		r1 = rf(doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguResourceGenerator_CreateDoguPVC_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguPVC'
type DoguResourceGenerator_CreateDoguPVC_Call struct {
	*mock.Call
}

// CreateDoguPVC is a helper method to define mock.On call
//   - doguResource *v1.Dogu
func (_e *DoguResourceGenerator_Expecter) CreateDoguPVC(doguResource interface{}) *DoguResourceGenerator_CreateDoguPVC_Call {
	return &DoguResourceGenerator_CreateDoguPVC_Call{Call: _e.mock.On("CreateDoguPVC", doguResource)}
}

func (_c *DoguResourceGenerator_CreateDoguPVC_Call) Run(run func(doguResource *v1.Dogu)) *DoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguPVC_Call) Return(_a0 *corev1.PersistentVolumeClaim, _a1 error) *DoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguPVC_Call) RunAndReturn(run func(*v1.Dogu) (*corev1.PersistentVolumeClaim, error)) *DoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Return(run)
	return _c
}

// CreateDoguSecret provides a mock function with given fields: doguResource, stringData
func (_m *DoguResourceGenerator) CreateDoguSecret(doguResource *v1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	ret := _m.Called(doguResource, stringData)

	var r0 *corev1.Secret
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu, map[string]string) (*corev1.Secret, error)); ok {
		return rf(doguResource, stringData)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu, map[string]string) *corev1.Secret); ok {
		r0 = rf(doguResource, stringData)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu, map[string]string) error); ok {
		r1 = rf(doguResource, stringData)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguResourceGenerator_CreateDoguSecret_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguSecret'
type DoguResourceGenerator_CreateDoguSecret_Call struct {
	*mock.Call
}

// CreateDoguSecret is a helper method to define mock.On call
//   - doguResource *v1.Dogu
//   - stringData map[string]string
func (_e *DoguResourceGenerator_Expecter) CreateDoguSecret(doguResource interface{}, stringData interface{}) *DoguResourceGenerator_CreateDoguSecret_Call {
	return &DoguResourceGenerator_CreateDoguSecret_Call{Call: _e.mock.On("CreateDoguSecret", doguResource, stringData)}
}

func (_c *DoguResourceGenerator_CreateDoguSecret_Call) Run(run func(doguResource *v1.Dogu, stringData map[string]string)) *DoguResourceGenerator_CreateDoguSecret_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu), args[1].(map[string]string))
	})
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguSecret_Call) Return(_a0 *corev1.Secret, _a1 error) *DoguResourceGenerator_CreateDoguSecret_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguSecret_Call) RunAndReturn(run func(*v1.Dogu, map[string]string) (*corev1.Secret, error)) *DoguResourceGenerator_CreateDoguSecret_Call {
	_c.Call.Return(run)
	return _c
}

// CreateDoguService provides a mock function with given fields: doguResource, imageConfig
func (_m *DoguResourceGenerator) CreateDoguService(doguResource *v1.Dogu, imageConfig *pkgv1.ConfigFile) (*corev1.Service, error) {
	ret := _m.Called(doguResource, imageConfig)

	var r0 *corev1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)); ok {
		return rf(doguResource, imageConfig)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *pkgv1.ConfigFile) *corev1.Service); ok {
		r0 = rf(doguResource, imageConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu, *pkgv1.ConfigFile) error); ok {
		r1 = rf(doguResource, imageConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguResourceGenerator_CreateDoguService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguService'
type DoguResourceGenerator_CreateDoguService_Call struct {
	*mock.Call
}

// CreateDoguService is a helper method to define mock.On call
//   - doguResource *v1.Dogu
//   - imageConfig *pkgv1.ConfigFile
func (_e *DoguResourceGenerator_Expecter) CreateDoguService(doguResource interface{}, imageConfig interface{}) *DoguResourceGenerator_CreateDoguService_Call {
	return &DoguResourceGenerator_CreateDoguService_Call{Call: _e.mock.On("CreateDoguService", doguResource, imageConfig)}
}

func (_c *DoguResourceGenerator_CreateDoguService_Call) Run(run func(doguResource *v1.Dogu, imageConfig *pkgv1.ConfigFile)) *DoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu), args[1].(*pkgv1.ConfigFile))
	})
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguService_Call) Return(_a0 *corev1.Service, _a1 error) *DoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguResourceGenerator_CreateDoguService_Call) RunAndReturn(run func(*v1.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)) *DoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Return(run)
	return _c
}

// GetPodTemplate provides a mock function with given fields: doguResource, dogu, chownInitImage
func (_m *DoguResourceGenerator) GetPodTemplate(doguResource *v1.Dogu, dogu *core.Dogu, chownInitImage string) (*corev1.PodTemplateSpec, error) {
	ret := _m.Called(doguResource, dogu, chownInitImage)

	var r0 *corev1.PodTemplateSpec
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu, string) (*corev1.PodTemplateSpec, error)); ok {
		return rf(doguResource, dogu, chownInitImage)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu, string) *corev1.PodTemplateSpec); ok {
		r0 = rf(doguResource, dogu, chownInitImage)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PodTemplateSpec)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu, *core.Dogu, string) error); ok {
		r1 = rf(doguResource, dogu, chownInitImage)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguResourceGenerator_GetPodTemplate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPodTemplate'
type DoguResourceGenerator_GetPodTemplate_Call struct {
	*mock.Call
}

// GetPodTemplate is a helper method to define mock.On call
//   - doguResource *v1.Dogu
//   - dogu *core.Dogu
//   - chownInitImage string
func (_e *DoguResourceGenerator_Expecter) GetPodTemplate(doguResource interface{}, dogu interface{}, chownInitImage interface{}) *DoguResourceGenerator_GetPodTemplate_Call {
	return &DoguResourceGenerator_GetPodTemplate_Call{Call: _e.mock.On("GetPodTemplate", doguResource, dogu, chownInitImage)}
}

func (_c *DoguResourceGenerator_GetPodTemplate_Call) Run(run func(doguResource *v1.Dogu, dogu *core.Dogu, chownInitImage string)) *DoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu), args[1].(*core.Dogu), args[2].(string))
	})
	return _c
}

func (_c *DoguResourceGenerator_GetPodTemplate_Call) Return(_a0 *corev1.PodTemplateSpec, _a1 error) *DoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguResourceGenerator_GetPodTemplate_Call) RunAndReturn(run func(*v1.Dogu, *core.Dogu, string) (*corev1.PodTemplateSpec, error)) *DoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguResourceGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguResourceGenerator creates a new instance of DoguResourceGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguResourceGenerator(t mockConstructorTestingTNewDoguResourceGenerator) *DoguResourceGenerator {
	mock := &DoguResourceGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
