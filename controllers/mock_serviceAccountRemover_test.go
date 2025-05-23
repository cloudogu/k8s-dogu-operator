// Code generated by mockery v2.53.3. DO NOT EDIT.

package controllers

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// mockServiceAccountRemover is an autogenerated mock type for the serviceAccountRemover type
type mockServiceAccountRemover struct {
	mock.Mock
}

type mockServiceAccountRemover_Expecter struct {
	mock *mock.Mock
}

func (_m *mockServiceAccountRemover) EXPECT() *mockServiceAccountRemover_Expecter {
	return &mockServiceAccountRemover_Expecter{mock: &_m.Mock}
}

// RemoveAll provides a mock function with given fields: ctx, dogu
func (_m *mockServiceAccountRemover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	if len(ret) == 0 {
		panic("no return value specified for RemoveAll")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockServiceAccountRemover_RemoveAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAll'
type mockServiceAccountRemover_RemoveAll_Call struct {
	*mock.Call
}

// RemoveAll is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *mockServiceAccountRemover_Expecter) RemoveAll(ctx interface{}, dogu interface{}) *mockServiceAccountRemover_RemoveAll_Call {
	return &mockServiceAccountRemover_RemoveAll_Call{Call: _e.mock.On("RemoveAll", ctx, dogu)}
}

func (_c *mockServiceAccountRemover_RemoveAll_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *mockServiceAccountRemover_RemoveAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *mockServiceAccountRemover_RemoveAll_Call) Return(_a0 error) *mockServiceAccountRemover_RemoveAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockServiceAccountRemover_RemoveAll_Call) RunAndReturn(run func(context.Context, *core.Dogu) error) *mockServiceAccountRemover_RemoveAll_Call {
	_c.Call.Return(run)
	return _c
}

// newMockServiceAccountRemover creates a new instance of mockServiceAccountRemover. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockServiceAccountRemover(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockServiceAccountRemover {
	mock := &mockServiceAccountRemover{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
