// Code generated by mockery v2.53.3. DO NOT EDIT.

package controllers

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	pkgv1 "github.com/google/go-containerregistry/pkg/v1"

	v1 "k8s.io/api/apps/v1"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

// MockDoguResourceGenerator is an autogenerated mock type for the DoguResourceGenerator type
type MockDoguResourceGenerator struct {
	mock.Mock
}

type MockDoguResourceGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDoguResourceGenerator) EXPECT() *MockDoguResourceGenerator_Expecter {
	return &MockDoguResourceGenerator_Expecter{mock: &_m.Mock}
}

// CreateDoguDeployment provides a mock function with given fields: ctx, doguResource, dogu
func (_m *MockDoguResourceGenerator) CreateDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu) (*v1.Deployment, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for CreateDoguDeployment")
	}

	var r0 *v1.Deployment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) (*v1.Deployment, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) *v1.Deployment); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Deployment)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguResourceGenerator_CreateDoguDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguDeployment'
type MockDoguResourceGenerator_CreateDoguDeployment_Call struct {
	*mock.Call
}

// CreateDoguDeployment is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *MockDoguResourceGenerator_Expecter) CreateDoguDeployment(ctx interface{}, doguResource interface{}, dogu interface{}) *MockDoguResourceGenerator_CreateDoguDeployment_Call {
	return &MockDoguResourceGenerator_CreateDoguDeployment_Call{Call: _e.mock.On("CreateDoguDeployment", ctx, doguResource, dogu)}
}

func (_c *MockDoguResourceGenerator_CreateDoguDeployment_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu)) *MockDoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguDeployment_Call) Return(_a0 *v1.Deployment, _a1 error) *MockDoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguDeployment_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu) (*v1.Deployment, error)) *MockDoguResourceGenerator_CreateDoguDeployment_Call {
	_c.Call.Return(run)
	return _c
}

// CreateDoguPVC provides a mock function with given fields: doguResource
func (_m *MockDoguResourceGenerator) CreateDoguPVC(doguResource *v2.Dogu) (*corev1.PersistentVolumeClaim, error) {
	ret := _m.Called(doguResource)

	if len(ret) == 0 {
		panic("no return value specified for CreateDoguPVC")
	}

	var r0 *corev1.PersistentVolumeClaim
	var r1 error
	if rf, ok := ret.Get(0).(func(*v2.Dogu) (*corev1.PersistentVolumeClaim, error)); ok {
		return rf(doguResource)
	}
	if rf, ok := ret.Get(0).(func(*v2.Dogu) *corev1.PersistentVolumeClaim); ok {
		r0 = rf(doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeClaim)
		}
	}

	if rf, ok := ret.Get(1).(func(*v2.Dogu) error); ok {
		r1 = rf(doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguResourceGenerator_CreateDoguPVC_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguPVC'
type MockDoguResourceGenerator_CreateDoguPVC_Call struct {
	*mock.Call
}

// CreateDoguPVC is a helper method to define mock.On call
//   - doguResource *v2.Dogu
func (_e *MockDoguResourceGenerator_Expecter) CreateDoguPVC(doguResource interface{}) *MockDoguResourceGenerator_CreateDoguPVC_Call {
	return &MockDoguResourceGenerator_CreateDoguPVC_Call{Call: _e.mock.On("CreateDoguPVC", doguResource)}
}

func (_c *MockDoguResourceGenerator_CreateDoguPVC_Call) Run(run func(doguResource *v2.Dogu)) *MockDoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v2.Dogu))
	})
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguPVC_Call) Return(_a0 *corev1.PersistentVolumeClaim, _a1 error) *MockDoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguPVC_Call) RunAndReturn(run func(*v2.Dogu) (*corev1.PersistentVolumeClaim, error)) *MockDoguResourceGenerator_CreateDoguPVC_Call {
	_c.Call.Return(run)
	return _c
}

// CreateDoguService provides a mock function with given fields: doguResource, dogu, imageConfig
func (_m *MockDoguResourceGenerator) CreateDoguService(doguResource *v2.Dogu, dogu *core.Dogu, imageConfig *pkgv1.ConfigFile) (*corev1.Service, error) {
	ret := _m.Called(doguResource, dogu, imageConfig)

	if len(ret) == 0 {
		panic("no return value specified for CreateDoguService")
	}

	var r0 *corev1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(*v2.Dogu, *core.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)); ok {
		return rf(doguResource, dogu, imageConfig)
	}
	if rf, ok := ret.Get(0).(func(*v2.Dogu, *core.Dogu, *pkgv1.ConfigFile) *corev1.Service); ok {
		r0 = rf(doguResource, dogu, imageConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(*v2.Dogu, *core.Dogu, *pkgv1.ConfigFile) error); ok {
		r1 = rf(doguResource, dogu, imageConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguResourceGenerator_CreateDoguService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateDoguService'
type MockDoguResourceGenerator_CreateDoguService_Call struct {
	*mock.Call
}

// CreateDoguService is a helper method to define mock.On call
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
//   - imageConfig *pkgv1.ConfigFile
func (_e *MockDoguResourceGenerator_Expecter) CreateDoguService(doguResource interface{}, dogu interface{}, imageConfig interface{}) *MockDoguResourceGenerator_CreateDoguService_Call {
	return &MockDoguResourceGenerator_CreateDoguService_Call{Call: _e.mock.On("CreateDoguService", doguResource, dogu, imageConfig)}
}

func (_c *MockDoguResourceGenerator_CreateDoguService_Call) Run(run func(doguResource *v2.Dogu, dogu *core.Dogu, imageConfig *pkgv1.ConfigFile)) *MockDoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v2.Dogu), args[1].(*core.Dogu), args[2].(*pkgv1.ConfigFile))
	})
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguService_Call) Return(_a0 *corev1.Service, _a1 error) *MockDoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguResourceGenerator_CreateDoguService_Call) RunAndReturn(run func(*v2.Dogu, *core.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)) *MockDoguResourceGenerator_CreateDoguService_Call {
	_c.Call.Return(run)
	return _c
}

// GetPodTemplate provides a mock function with given fields: ctx, doguResource, dogu
func (_m *MockDoguResourceGenerator) GetPodTemplate(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu) (*corev1.PodTemplateSpec, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for GetPodTemplate")
	}

	var r0 *corev1.PodTemplateSpec
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.PodTemplateSpec, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) *corev1.PodTemplateSpec); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PodTemplateSpec)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguResourceGenerator_GetPodTemplate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPodTemplate'
type MockDoguResourceGenerator_GetPodTemplate_Call struct {
	*mock.Call
}

// GetPodTemplate is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *MockDoguResourceGenerator_Expecter) GetPodTemplate(ctx interface{}, doguResource interface{}, dogu interface{}) *MockDoguResourceGenerator_GetPodTemplate_Call {
	return &MockDoguResourceGenerator_GetPodTemplate_Call{Call: _e.mock.On("GetPodTemplate", ctx, doguResource, dogu)}
}

func (_c *MockDoguResourceGenerator_GetPodTemplate_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu)) *MockDoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *MockDoguResourceGenerator_GetPodTemplate_Call) Return(_a0 *corev1.PodTemplateSpec, _a1 error) *MockDoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguResourceGenerator_GetPodTemplate_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.PodTemplateSpec, error)) *MockDoguResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDoguResourceGenerator creates a new instance of MockDoguResourceGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDoguResourceGenerator(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDoguResourceGenerator {
	mock := &MockDoguResourceGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
