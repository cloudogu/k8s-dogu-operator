// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// DoguStartStopManager is an autogenerated mock type for the DoguStartStopManager type
type DoguStartStopManager struct {
	mock.Mock
}

type DoguStartStopManager_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguStartStopManager) EXPECT() *DoguStartStopManager_Expecter {
	return &DoguStartStopManager_Expecter{mock: &_m.Mock}
}

// CheckStarted provides a mock function with given fields: ctx, doguResource
func (_m *DoguStartStopManager) CheckStarted(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguStartStopManager_CheckStarted_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStarted'
type DoguStartStopManager_CheckStarted_Call struct {
	*mock.Call
}

// CheckStarted is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguStartStopManager_Expecter) CheckStarted(ctx interface{}, doguResource interface{}) *DoguStartStopManager_CheckStarted_Call {
	return &DoguStartStopManager_CheckStarted_Call{Call: _e.mock.On("CheckStarted", ctx, doguResource)}
}

func (_c *DoguStartStopManager_CheckStarted_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguStartStopManager_CheckStarted_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguStartStopManager_CheckStarted_Call) Return(_a0 error) *DoguStartStopManager_CheckStarted_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguStartStopManager_CheckStarted_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguStartStopManager_CheckStarted_Call {
	_c.Call.Return(run)
	return _c
}

// CheckStopped provides a mock function with given fields: ctx, doguResource
func (_m *DoguStartStopManager) CheckStopped(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguStartStopManager_CheckStopped_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStopped'
type DoguStartStopManager_CheckStopped_Call struct {
	*mock.Call
}

// CheckStopped is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguStartStopManager_Expecter) CheckStopped(ctx interface{}, doguResource interface{}) *DoguStartStopManager_CheckStopped_Call {
	return &DoguStartStopManager_CheckStopped_Call{Call: _e.mock.On("CheckStopped", ctx, doguResource)}
}

func (_c *DoguStartStopManager_CheckStopped_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguStartStopManager_CheckStopped_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguStartStopManager_CheckStopped_Call) Return(_a0 error) *DoguStartStopManager_CheckStopped_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguStartStopManager_CheckStopped_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguStartStopManager_CheckStopped_Call {
	_c.Call.Return(run)
	return _c
}

// StartDogu provides a mock function with given fields: ctx, doguResource
func (_m *DoguStartStopManager) StartDogu(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguStartStopManager_StartDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartDogu'
type DoguStartStopManager_StartDogu_Call struct {
	*mock.Call
}

// StartDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguStartStopManager_Expecter) StartDogu(ctx interface{}, doguResource interface{}) *DoguStartStopManager_StartDogu_Call {
	return &DoguStartStopManager_StartDogu_Call{Call: _e.mock.On("StartDogu", ctx, doguResource)}
}

func (_c *DoguStartStopManager_StartDogu_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguStartStopManager_StartDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguStartStopManager_StartDogu_Call) Return(_a0 error) *DoguStartStopManager_StartDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguStartStopManager_StartDogu_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguStartStopManager_StartDogu_Call {
	_c.Call.Return(run)
	return _c
}

// StopDogu provides a mock function with given fields: ctx, doguResource
func (_m *DoguStartStopManager) StopDogu(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguStartStopManager_StopDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopDogu'
type DoguStartStopManager_StopDogu_Call struct {
	*mock.Call
}

// StopDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *DoguStartStopManager_Expecter) StopDogu(ctx interface{}, doguResource interface{}) *DoguStartStopManager_StopDogu_Call {
	return &DoguStartStopManager_StopDogu_Call{Call: _e.mock.On("StopDogu", ctx, doguResource)}
}

func (_c *DoguStartStopManager_StopDogu_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *DoguStartStopManager_StopDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *DoguStartStopManager_StopDogu_Call) Return(_a0 error) *DoguStartStopManager_StopDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguStartStopManager_StopDogu_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *DoguStartStopManager_StopDogu_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguStartStopManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguStartStopManager creates a new instance of DoguStartStopManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguStartStopManager(t mockConstructorTestingTNewDoguStartStopManager) *DoguStartStopManager {
	mock := &DoguStartStopManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
