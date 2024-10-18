// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// mockStartDoguManager is an autogenerated mock type for the startDoguManager type
type mockStartDoguManager struct {
	mock.Mock
}

type mockStartDoguManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockStartDoguManager) EXPECT() *mockStartDoguManager_Expecter {
	return &mockStartDoguManager_Expecter{mock: &_m.Mock}
}

// CheckStarted provides a mock function with given fields: ctx, doguResource
func (_m *mockStartDoguManager) CheckStarted(ctx context.Context, doguResource *v2.Dogu) error {
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

// mockStartDoguManager_CheckStarted_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckStarted'
type mockStartDoguManager_CheckStarted_Call struct {
	*mock.Call
}

// CheckStarted is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockStartDoguManager_Expecter) CheckStarted(ctx interface{}, doguResource interface{}) *mockStartDoguManager_CheckStarted_Call {
	return &mockStartDoguManager_CheckStarted_Call{Call: _e.mock.On("CheckStarted", ctx, doguResource)}
}

func (_c *mockStartDoguManager_CheckStarted_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockStartDoguManager_CheckStarted_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockStartDoguManager_CheckStarted_Call) Return(_a0 error) *mockStartDoguManager_CheckStarted_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockStartDoguManager_CheckStarted_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *mockStartDoguManager_CheckStarted_Call {
	_c.Call.Return(run)
	return _c
}

// StartDogu provides a mock function with given fields: ctx, doguResource
func (_m *mockStartDoguManager) StartDogu(ctx context.Context, doguResource *v2.Dogu) error {
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

// mockStartDoguManager_StartDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartDogu'
type mockStartDoguManager_StartDogu_Call struct {
	*mock.Call
}

// StartDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockStartDoguManager_Expecter) StartDogu(ctx interface{}, doguResource interface{}) *mockStartDoguManager_StartDogu_Call {
	return &mockStartDoguManager_StartDogu_Call{Call: _e.mock.On("StartDogu", ctx, doguResource)}
}

func (_c *mockStartDoguManager_StartDogu_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockStartDoguManager_StartDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockStartDoguManager_StartDogu_Call) Return(_a0 error) *mockStartDoguManager_StartDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockStartDoguManager_StartDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *mockStartDoguManager_StartDogu_Call {
	_c.Call.Return(run)
	return _c
}

// newMockStartDoguManager creates a new instance of mockStartDoguManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockStartDoguManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockStartDoguManager {
	mock := &mockStartDoguManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}