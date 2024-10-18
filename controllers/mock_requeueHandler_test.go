// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	reconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// mockRequeueHandler is an autogenerated mock type for the requeueHandler type
type mockRequeueHandler struct {
	mock.Mock
}

type mockRequeueHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRequeueHandler) EXPECT() *mockRequeueHandler_Expecter {
	return &mockRequeueHandler_Expecter{mock: &_m.Mock}
}

// Handle provides a mock function with given fields: ctx, contextMessage, doguResource, err, onRequeue
func (_m *mockRequeueHandler) Handle(ctx context.Context, contextMessage string, doguResource *v2.Dogu, err error, onRequeue func(*v2.Dogu) error) (reconcile.Result, error) {
	ret := _m.Called(ctx, contextMessage, doguResource, err, onRequeue)

	if len(ret) == 0 {
		panic("no return value specified for Handle")
	}

	var r0 reconcile.Result
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *v2.Dogu, error, func(*v2.Dogu) error) (reconcile.Result, error)); ok {
		return rf(ctx, contextMessage, doguResource, err, onRequeue)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *v2.Dogu, error, func(*v2.Dogu) error) reconcile.Result); ok {
		r0 = rf(ctx, contextMessage, doguResource, err, onRequeue)
	} else {
		r0 = ret.Get(0).(reconcile.Result)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *v2.Dogu, error, func(*v2.Dogu) error) error); ok {
		r1 = rf(ctx, contextMessage, doguResource, err, onRequeue)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockRequeueHandler_Handle_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Handle'
type mockRequeueHandler_Handle_Call struct {
	*mock.Call
}

// Handle is a helper method to define mock.On call
//   - ctx context.Context
//   - contextMessage string
//   - doguResource *v2.Dogu
//   - err error
//   - onRequeue func(*v2.Dogu) error
func (_e *mockRequeueHandler_Expecter) Handle(ctx interface{}, contextMessage interface{}, doguResource interface{}, err interface{}, onRequeue interface{}) *mockRequeueHandler_Handle_Call {
	return &mockRequeueHandler_Handle_Call{Call: _e.mock.On("Handle", ctx, contextMessage, doguResource, err, onRequeue)}
}

func (_c *mockRequeueHandler_Handle_Call) Run(run func(ctx context.Context, contextMessage string, doguResource *v2.Dogu, err error, onRequeue func(*v2.Dogu) error)) *mockRequeueHandler_Handle_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*v2.Dogu), args[3].(error), args[4].(func(*v2.Dogu) error))
	})
	return _c
}

func (_c *mockRequeueHandler_Handle_Call) Return(result reconcile.Result, requeueErr error) *mockRequeueHandler_Handle_Call {
	_c.Call.Return(result, requeueErr)
	return _c
}

func (_c *mockRequeueHandler_Handle_Call) RunAndReturn(run func(context.Context, string, *v2.Dogu, error, func(*v2.Dogu) error) (reconcile.Result, error)) *mockRequeueHandler_Handle_Call {
	_c.Call.Return(run)
	return _c
}

// newMockRequeueHandler creates a new instance of mockRequeueHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRequeueHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRequeueHandler {
	mock := &mockRequeueHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
