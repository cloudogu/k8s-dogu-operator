// Code generated by mockery v2.53.2. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// MockDoguStartStopManager is an autogenerated mock type for the DoguStartStopManager type
type MockDoguStartStopManager struct {
	mock.Mock
}

type MockDoguStartStopManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDoguStartStopManager) EXPECT() *MockDoguStartStopManager_Expecter {
	return &MockDoguStartStopManager_Expecter{mock: &_m.Mock}
}

// CheckStarted provides a mock function with given fields: ctx, doguResource
func (_m *MockDoguStartStopManager) CheckStarted(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for CheckStarted")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguStartStopManager_CheckStarted_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStarted'
type MockDoguStartStopManager_CheckStarted_Call struct {
	*mock.Call
}

// CheckStarted is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockDoguStartStopManager_Expecter) CheckStarted(ctx interface{}, doguResource interface{}) *MockDoguStartStopManager_CheckStarted_Call {
	return &MockDoguStartStopManager_CheckStarted_Call{Call: _e.mock.On("CheckStarted", ctx, doguResource)}
}

func (_c *MockDoguStartStopManager_CheckStarted_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockDoguStartStopManager_CheckStarted_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockDoguStartStopManager_CheckStarted_Call) Return(_a0 error) *MockDoguStartStopManager_CheckStarted_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguStartStopManager_CheckStarted_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockDoguStartStopManager_CheckStarted_Call {
	_c.Call.Return(run)
	return _c
}

// CheckStopped provides a mock function with given fields: ctx, doguResource
func (_m *MockDoguStartStopManager) CheckStopped(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for CheckStopped")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguStartStopManager_CheckStopped_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStopped'
type MockDoguStartStopManager_CheckStopped_Call struct {
	*mock.Call
}

// CheckStopped is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockDoguStartStopManager_Expecter) CheckStopped(ctx interface{}, doguResource interface{}) *MockDoguStartStopManager_CheckStopped_Call {
	return &MockDoguStartStopManager_CheckStopped_Call{Call: _e.mock.On("CheckStopped", ctx, doguResource)}
}

func (_c *MockDoguStartStopManager_CheckStopped_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockDoguStartStopManager_CheckStopped_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockDoguStartStopManager_CheckStopped_Call) Return(_a0 error) *MockDoguStartStopManager_CheckStopped_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguStartStopManager_CheckStopped_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockDoguStartStopManager_CheckStopped_Call {
	_c.Call.Return(run)
	return _c
}

// StartDogu provides a mock function with given fields: ctx, doguResource
func (_m *MockDoguStartStopManager) StartDogu(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for StartDogu")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguStartStopManager_StartDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartDogu'
type MockDoguStartStopManager_StartDogu_Call struct {
	*mock.Call
}

// StartDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockDoguStartStopManager_Expecter) StartDogu(ctx interface{}, doguResource interface{}) *MockDoguStartStopManager_StartDogu_Call {
	return &MockDoguStartStopManager_StartDogu_Call{Call: _e.mock.On("StartDogu", ctx, doguResource)}
}

func (_c *MockDoguStartStopManager_StartDogu_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockDoguStartStopManager_StartDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockDoguStartStopManager_StartDogu_Call) Return(_a0 error) *MockDoguStartStopManager_StartDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguStartStopManager_StartDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockDoguStartStopManager_StartDogu_Call {
	_c.Call.Return(run)
	return _c
}

// StopDogu provides a mock function with given fields: ctx, doguResource
func (_m *MockDoguStartStopManager) StopDogu(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for StopDogu")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguStartStopManager_StopDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopDogu'
type MockDoguStartStopManager_StopDogu_Call struct {
	*mock.Call
}

// StopDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *MockDoguStartStopManager_Expecter) StopDogu(ctx interface{}, doguResource interface{}) *MockDoguStartStopManager_StopDogu_Call {
	return &MockDoguStartStopManager_StopDogu_Call{Call: _e.mock.On("StopDogu", ctx, doguResource)}
}

func (_c *MockDoguStartStopManager_StopDogu_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *MockDoguStartStopManager_StopDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *MockDoguStartStopManager_StopDogu_Call) Return(_a0 error) *MockDoguStartStopManager_StopDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguStartStopManager_StopDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *MockDoguStartStopManager_StopDogu_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDoguStartStopManager creates a new instance of MockDoguStartStopManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDoguStartStopManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDoguStartStopManager {
	mock := &MockDoguStartStopManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
