// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// DoguManager is an autogenerated mock type for the DoguManager type
type DoguManager struct {
	mock.Mock
}

type DoguManager_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguManager) EXPECT() *DoguManager_Expecter {
	return &DoguManager_Expecter{mock: &_m.Mock}
}

// Delete provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) Delete(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguManager_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type DoguManager_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) Delete(ctx interface{}, doguResource interface{}) *DoguManager_Delete_Call {
	return &DoguManager_Delete_Call{Call: _e.mock.On("Delete", ctx, doguResource)}
}

func (_c *DoguManager_Delete_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_Delete_Call) Return(_a0 error) *DoguManager_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguManager_Delete_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguManager_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// HandleSupportMode provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) HandleSupportMode(ctx context.Context, doguResource *v1.Dogu) (bool, error) {
	ret := _m.Called(ctx, doguResource)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) (bool, error)); ok {
		return rf(ctx, doguResource)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) bool); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu) error); ok {
		r1 = rf(ctx, doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguManager_HandleSupportMode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HandleSupportMode'
type DoguManager_HandleSupportMode_Call struct {
	*mock.Call
}

// HandleSupportMode is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) HandleSupportMode(ctx interface{}, doguResource interface{}) *DoguManager_HandleSupportMode_Call {
	return &DoguManager_HandleSupportMode_Call{Call: _e.mock.On("HandleSupportMode", ctx, doguResource)}
}

func (_c *DoguManager_HandleSupportMode_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_HandleSupportMode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_HandleSupportMode_Call) Return(_a0 bool, _a1 error) *DoguManager_HandleSupportMode_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguManager_HandleSupportMode_Call) RunAndReturn(run func(context.Context, *v1.Dogu) (bool, error)) *DoguManager_HandleSupportMode_Call {
	_c.Call.Return(run)
	return _c
}

// Install provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) Install(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguManager_Install_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Install'
type DoguManager_Install_Call struct {
	*mock.Call
}

// Install is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) Install(ctx interface{}, doguResource interface{}) *DoguManager_Install_Call {
	return &DoguManager_Install_Call{Call: _e.mock.On("Install", ctx, doguResource)}
}

func (_c *DoguManager_Install_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_Install_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_Install_Call) Return(_a0 error) *DoguManager_Install_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguManager_Install_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguManager_Install_Call {
	_c.Call.Return(run)
	return _c
}

// SetDoguAdditionalIngressAnnotations provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguManager_SetDoguAdditionalIngressAnnotations_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguAdditionalIngressAnnotations'
type DoguManager_SetDoguAdditionalIngressAnnotations_Call struct {
	*mock.Call
}

// SetDoguAdditionalIngressAnnotations is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) SetDoguAdditionalIngressAnnotations(ctx interface{}, doguResource interface{}) *DoguManager_SetDoguAdditionalIngressAnnotations_Call {
	return &DoguManager_SetDoguAdditionalIngressAnnotations_Call{Call: _e.mock.On("SetDoguAdditionalIngressAnnotations", ctx, doguResource)}
}

func (_c *DoguManager_SetDoguAdditionalIngressAnnotations_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_SetDoguAdditionalIngressAnnotations_Call) Return(_a0 error) *DoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguManager_SetDoguAdditionalIngressAnnotations_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Return(run)
	return _c
}

// SetDoguDataVolumeSize provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguManager_SetDoguDataVolumeSize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguDataVolumeSize'
type DoguManager_SetDoguDataVolumeSize_Call struct {
	*mock.Call
}

// SetDoguDataVolumeSize is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) SetDoguDataVolumeSize(ctx interface{}, doguResource interface{}) *DoguManager_SetDoguDataVolumeSize_Call {
	return &DoguManager_SetDoguDataVolumeSize_Call{Call: _e.mock.On("SetDoguDataVolumeSize", ctx, doguResource)}
}

func (_c *DoguManager_SetDoguDataVolumeSize_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_SetDoguDataVolumeSize_Call) Return(_a0 error) *DoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguManager_SetDoguDataVolumeSize_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(run)
	return _c
}

// Upgrade provides a mock function with given fields: ctx, doguResource
func (_m *DoguManager) Upgrade(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguManager_Upgrade_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Upgrade'
type DoguManager_Upgrade_Call struct {
	*mock.Call
}

// Upgrade is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguManager_Expecter) Upgrade(ctx interface{}, doguResource interface{}) *DoguManager_Upgrade_Call {
	return &DoguManager_Upgrade_Call{Call: _e.mock.On("Upgrade", ctx, doguResource)}
}

func (_c *DoguManager_Upgrade_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguManager_Upgrade_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguManager_Upgrade_Call) Return(_a0 error) *DoguManager_Upgrade_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguManager_Upgrade_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguManager_Upgrade_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguManager creates a new instance of DoguManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguManager(t mockConstructorTestingTNewDoguManager) *DoguManager {
	mock := &DoguManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
