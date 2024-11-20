// Code generated by mockery v2.20.0. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// MockCombinedDoguManager is an autogenerated mock type for the CombinedDoguManager type
type MockCombinedDoguManager struct {
	mock.Mock
}

type MockCombinedDoguManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCombinedDoguManager) EXPECT() *MockCombinedDoguManager_Expecter {
	return &MockCombinedDoguManager_Expecter{mock: &_m.Mock}
}

// CheckStarted provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) CheckStarted(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_CheckStarted_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStarted'
type MockCombinedDoguManager_CheckStarted_Call struct {
	*mock.Call
}

// CheckStarted is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) CheckStarted(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_CheckStarted_Call {
	return &MockCombinedDoguManager_CheckStarted_Call{Call: _e.mock.On("CheckStarted", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_CheckStarted_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_CheckStarted_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_CheckStarted_Call) Return(_a0 error) *MockCombinedDoguManager_CheckStarted_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_CheckStarted_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_CheckStarted_Call {
	_c.Call.Return(run)
	return _c
}

// CheckStopped provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) CheckStopped(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_CheckStopped_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStopped'
type MockCombinedDoguManager_CheckStopped_Call struct {
	*mock.Call
}

// CheckStopped is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) CheckStopped(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_CheckStopped_Call {
	return &MockCombinedDoguManager_CheckStopped_Call{Call: _e.mock.On("CheckStopped", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_CheckStopped_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_CheckStopped_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_CheckStopped_Call) Return(_a0 error) *MockCombinedDoguManager_CheckStopped_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_CheckStopped_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_CheckStopped_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) Delete(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type MockCombinedDoguManager_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) Delete(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_Delete_Call {
	return &MockCombinedDoguManager_Delete_Call{Call: _e.mock.On("Delete", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_Delete_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_Delete_Call) Return(_a0 error) *MockCombinedDoguManager_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_Delete_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// HandleSupportMode provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	ret := _m.Called(ctx, doguResource)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) (bool, error)); ok {
		return rf(ctx, doguResource)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) bool); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu) error); ok {
		r1 = rf(ctx, doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCombinedDoguManager_HandleSupportMode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HandleSupportMode'
type MockCombinedDoguManager_HandleSupportMode_Call struct {
	*mock.Call
}

// HandleSupportMode is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) HandleSupportMode(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_HandleSupportMode_Call {
	return &MockCombinedDoguManager_HandleSupportMode_Call{Call: _e.mock.On("HandleSupportMode", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_HandleSupportMode_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_HandleSupportMode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_HandleSupportMode_Call) Return(_a0 bool, _a1 error) *MockCombinedDoguManager_HandleSupportMode_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCombinedDoguManager_HandleSupportMode_Call) RunAndReturn(run func(context.Context, *v2.Dogu) (bool, error)) *MockCombinedDoguManager_HandleSupportMode_Call {
	_c.Call.Return(run)
	return _c
}

// Install provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) Install(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_Install_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Install'
type MockCombinedDoguManager_Install_Call struct {
	*mock.Call
}

// Install is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) Install(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_Install_Call {
	return &MockCombinedDoguManager_Install_Call{Call: _e.mock.On("Install", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_Install_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_Install_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_Install_Call) Return(_a0 error) *MockCombinedDoguManager_Install_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_Install_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_Install_Call {
	_c.Call.Return(run)
	return _c
}

// SetDoguAdditionalIngressAnnotations provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguAdditionalIngressAnnotations'
type MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call struct {
	*mock.Call
}

// SetDoguAdditionalIngressAnnotations is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) SetDoguAdditionalIngressAnnotations(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call {
	return &MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call{Call: _e.mock.On("SetDoguAdditionalIngressAnnotations", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call) Return(_a0 error) *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_SetDoguAdditionalIngressAnnotations_Call {
	_c.Call.Return(run)
	return _c
}

// SetDoguDataVolumeSize provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_SetDoguDataVolumeSize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguDataVolumeSize'
type MockCombinedDoguManager_SetDoguDataVolumeSize_Call struct {
	*mock.Call
}

// SetDoguDataVolumeSize is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) SetDoguDataVolumeSize(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_SetDoguDataVolumeSize_Call {
	return &MockCombinedDoguManager_SetDoguDataVolumeSize_Call{Call: _e.mock.On("SetDoguDataVolumeSize", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_SetDoguDataVolumeSize_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_SetDoguDataVolumeSize_Call) Return(_a0 error) *MockCombinedDoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_SetDoguDataVolumeSize_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(run)
	return _c
}

// StartDogu provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) StartDogu(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_StartDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartDogu'
type MockCombinedDoguManager_StartDogu_Call struct {
	*mock.Call
}

// StartDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) StartDogu(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_StartDogu_Call {
	return &MockCombinedDoguManager_StartDogu_Call{Call: _e.mock.On("StartDogu", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_StartDogu_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_StartDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_StartDogu_Call) Return(_a0 error) *MockCombinedDoguManager_StartDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_StartDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_StartDogu_Call {
	_c.Call.Return(run)
	return _c
}

// StopDogu provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) StopDogu(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_StopDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopDogu'
type MockCombinedDoguManager_StopDogu_Call struct {
	*mock.Call
}

// StopDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) StopDogu(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_StopDogu_Call {
	return &MockCombinedDoguManager_StopDogu_Call{Call: _e.mock.On("StopDogu", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_StopDogu_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_StopDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_StopDogu_Call) Return(_a0 error) *MockCombinedDoguManager_StopDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_StopDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_StopDogu_Call {
	_c.Call.Return(run)
	return _c
}

// Upgrade provides a mock function with given fields: ctx, doguResource
func (_m *MockCombinedDoguManager) Upgrade(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCombinedDoguManager_Upgrade_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Upgrade'
type MockCombinedDoguManager_Upgrade_Call struct {
	*mock.Call
}

// Upgrade is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockCombinedDoguManager_Expecter) Upgrade(ctx interface{}, doguResource interface{}) *MockCombinedDoguManager_Upgrade_Call {
	return &MockCombinedDoguManager_Upgrade_Call{Call: _e.mock.On("Upgrade", ctx, doguResource)}
}

func (_c *MockCombinedDoguManager_Upgrade_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockCombinedDoguManager_Upgrade_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCombinedDoguManager_Upgrade_Call) Return(_a0 error) *MockCombinedDoguManager_Upgrade_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCombinedDoguManager_Upgrade_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockCombinedDoguManager_Upgrade_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockCombinedDoguManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockCombinedDoguManager creates a new instance of MockCombinedDoguManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockCombinedDoguManager(t mockConstructorTestingTNewMockCombinedDoguManager) *MockCombinedDoguManager {
	mock := &MockCombinedDoguManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
