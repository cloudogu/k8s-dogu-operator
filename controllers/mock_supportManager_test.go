// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// mockSupportManager is an autogenerated mock type for the supportManager type
type mockSupportManager struct {
	mock.Mock
}

type mockSupportManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSupportManager) EXPECT() *mockSupportManager_Expecter {
	return &mockSupportManager_Expecter{mock: &_m.Mock}
}

// HandleSupportMode provides a mock function with given fields: ctx, doguResource
func (_m *mockSupportManager) HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for HandleSupportMode")
	}

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

// mockSupportManager_HandleSupportMode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HandleSupportMode'
type mockSupportManager_HandleSupportMode_Call struct {
	*mock.Call
}

// HandleSupportMode is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockSupportManager_Expecter) HandleSupportMode(ctx interface{}, doguResource interface{}) *mockSupportManager_HandleSupportMode_Call {
	return &mockSupportManager_HandleSupportMode_Call{Call: _e.mock.On("HandleSupportMode", ctx, doguResource)}
}

func (_c *mockSupportManager_HandleSupportMode_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockSupportManager_HandleSupportMode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockSupportManager_HandleSupportMode_Call) Return(_a0 bool, _a1 error) *mockSupportManager_HandleSupportMode_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSupportManager_HandleSupportMode_Call) RunAndReturn(run func(context.Context, *v2.Dogu) (bool, error)) *mockSupportManager_HandleSupportMode_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSupportManager creates a new instance of mockSupportManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSupportManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSupportManager {
	mock := &mockSupportManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
