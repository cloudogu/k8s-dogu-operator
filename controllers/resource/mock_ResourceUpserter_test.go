// Code generated by mockery v2.46.2. DO NOT EDIT.

package resource

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	pkgv1 "github.com/google/go-containerregistry/pkg/v1"

	v1 "k8s.io/api/apps/v1"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// MockResourceUpserter is an autogenerated mock type for the ResourceUpserter type
type MockResourceUpserter struct {
	mock.Mock
}

type MockResourceUpserter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockResourceUpserter) EXPECT() *MockResourceUpserter_Expecter {
	return &MockResourceUpserter_Expecter{mock: &_m.Mock}
}

// UpsertDoguDeployment provides a mock function with given fields: ctx, doguResource, dogu, deploymentPatch
func (_m *MockResourceUpserter) UpsertDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu, deploymentPatch func(*v1.Deployment)) (*v1.Deployment, error) {
	ret := _m.Called(ctx, doguResource, dogu, deploymentPatch)

	if len(ret) == 0 {
		panic("no return value specified for UpsertDoguDeployment")
	}

	var r0 *v1.Deployment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu, func(*v1.Deployment)) (*v1.Deployment, error)); ok {
		return rf(ctx, doguResource, dogu, deploymentPatch)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu, func(*v1.Deployment)) *v1.Deployment); ok {
		r0 = rf(ctx, doguResource, dogu, deploymentPatch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Deployment)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu, func(*v1.Deployment)) error); ok {
		r1 = rf(ctx, doguResource, dogu, deploymentPatch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockResourceUpserter_UpsertDoguDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpsertDoguDeployment'
type MockResourceUpserter_UpsertDoguDeployment_Call struct {
	*mock.Call
}

// UpsertDoguDeployment is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
//   - deploymentPatch func(*v1.Deployment)
func (_e *MockResourceUpserter_Expecter) UpsertDoguDeployment(ctx interface{}, doguResource interface{}, dogu interface{}, deploymentPatch interface{}) *MockResourceUpserter_UpsertDoguDeployment_Call {
	return &MockResourceUpserter_UpsertDoguDeployment_Call{Call: _e.mock.On("UpsertDoguDeployment", ctx, doguResource, dogu, deploymentPatch)}
}

func (_c *MockResourceUpserter_UpsertDoguDeployment_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu, deploymentPatch func(*v1.Deployment))) *MockResourceUpserter_UpsertDoguDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu), args[3].(func(*v1.Deployment)))
	})
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguDeployment_Call) Return(_a0 *v1.Deployment, _a1 error) *MockResourceUpserter_UpsertDoguDeployment_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguDeployment_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu, func(*v1.Deployment)) (*v1.Deployment, error)) *MockResourceUpserter_UpsertDoguDeployment_Call {
	_c.Call.Return(run)
	return _c
}

// UpsertDoguExposedService provides a mock function with given fields: ctx, doguResource, dogu
func (_m *MockResourceUpserter) UpsertDoguExposedService(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu) (*corev1.Service, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for UpsertDoguExposedService")
	}

	var r0 *corev1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.Service, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) *corev1.Service); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockResourceUpserter_UpsertDoguExposedService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpsertDoguExposedService'
type MockResourceUpserter_UpsertDoguExposedService_Call struct {
	*mock.Call
}

// UpsertDoguExposedService is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *MockResourceUpserter_Expecter) UpsertDoguExposedService(ctx interface{}, doguResource interface{}, dogu interface{}) *MockResourceUpserter_UpsertDoguExposedService_Call {
	return &MockResourceUpserter_UpsertDoguExposedService_Call{Call: _e.mock.On("UpsertDoguExposedService", ctx, doguResource, dogu)}
}

func (_c *MockResourceUpserter_UpsertDoguExposedService_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu)) *MockResourceUpserter_UpsertDoguExposedService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguExposedService_Call) Return(_a0 *corev1.Service, _a1 error) *MockResourceUpserter_UpsertDoguExposedService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguExposedService_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.Service, error)) *MockResourceUpserter_UpsertDoguExposedService_Call {
	_c.Call.Return(run)
	return _c
}

// UpsertDoguPVCs provides a mock function with given fields: ctx, doguResource, dogu
func (_m *MockResourceUpserter) UpsertDoguPVCs(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu) (*corev1.PersistentVolumeClaim, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for UpsertDoguPVCs")
	}

	var r0 *corev1.PersistentVolumeClaim
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.PersistentVolumeClaim, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) *corev1.PersistentVolumeClaim); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeClaim)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockResourceUpserter_UpsertDoguPVCs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpsertDoguPVCs'
type MockResourceUpserter_UpsertDoguPVCs_Call struct {
	*mock.Call
}

// UpsertDoguPVCs is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *MockResourceUpserter_Expecter) UpsertDoguPVCs(ctx interface{}, doguResource interface{}, dogu interface{}) *MockResourceUpserter_UpsertDoguPVCs_Call {
	return &MockResourceUpserter_UpsertDoguPVCs_Call{Call: _e.mock.On("UpsertDoguPVCs", ctx, doguResource, dogu)}
}

func (_c *MockResourceUpserter_UpsertDoguPVCs_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu)) *MockResourceUpserter_UpsertDoguPVCs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguPVCs_Call) Return(_a0 *corev1.PersistentVolumeClaim, _a1 error) *MockResourceUpserter_UpsertDoguPVCs_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguPVCs_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu) (*corev1.PersistentVolumeClaim, error)) *MockResourceUpserter_UpsertDoguPVCs_Call {
	_c.Call.Return(run)
	return _c
}

// UpsertDoguService provides a mock function with given fields: ctx, doguResource, image
func (_m *MockResourceUpserter) UpsertDoguService(ctx context.Context, doguResource *v2.Dogu, image *pkgv1.ConfigFile) (*corev1.Service, error) {
	ret := _m.Called(ctx, doguResource, image)

	if len(ret) == 0 {
		panic("no return value specified for UpsertDoguService")
	}

	var r0 *corev1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)); ok {
		return rf(ctx, doguResource, image)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *pkgv1.ConfigFile) *corev1.Service); ok {
		r0 = rf(ctx, doguResource, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *pkgv1.ConfigFile) error); ok {
		r1 = rf(ctx, doguResource, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockResourceUpserter_UpsertDoguService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpsertDoguService'
type MockResourceUpserter_UpsertDoguService_Call struct {
	*mock.Call
}

// UpsertDoguService is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - image *pkgv1.ConfigFile
func (_e *MockResourceUpserter_Expecter) UpsertDoguService(ctx interface{}, doguResource interface{}, image interface{}) *MockResourceUpserter_UpsertDoguService_Call {
	return &MockResourceUpserter_UpsertDoguService_Call{Call: _e.mock.On("UpsertDoguService", ctx, doguResource, image)}
}

func (_c *MockResourceUpserter_UpsertDoguService_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, image *pkgv1.ConfigFile)) *MockResourceUpserter_UpsertDoguService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*pkgv1.ConfigFile))
	})
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguService_Call) Return(_a0 *corev1.Service, _a1 error) *MockResourceUpserter_UpsertDoguService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockResourceUpserter_UpsertDoguService_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *pkgv1.ConfigFile) (*corev1.Service, error)) *MockResourceUpserter_UpsertDoguService_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockResourceUpserter creates a new instance of MockResourceUpserter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockResourceUpserter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockResourceUpserter {
	mock := &MockResourceUpserter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
