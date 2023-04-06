// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// ServiceAccountRemover is an autogenerated mock type for the ServiceAccountRemover type
type ServiceAccountRemover struct {
	mock.Mock
}

type ServiceAccountRemover_Expecter struct {
	mock *mock.Mock
}

func (_m *ServiceAccountRemover) EXPECT() *ServiceAccountRemover_Expecter {
	return &ServiceAccountRemover_Expecter{mock: &_m.Mock}
}

// RemoveAll provides a mock function with given fields: ctx, dogu
func (_m *ServiceAccountRemover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ServiceAccountRemover_RemoveAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAll'
type ServiceAccountRemover_RemoveAll_Call struct {
	*mock.Call
}

// RemoveAll is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *ServiceAccountRemover_Expecter) RemoveAll(ctx interface{}, dogu interface{}) *ServiceAccountRemover_RemoveAll_Call {
	return &ServiceAccountRemover_RemoveAll_Call{Call: _e.mock.On("RemoveAll", ctx, dogu)}
}

func (_c *ServiceAccountRemover_RemoveAll_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *ServiceAccountRemover_RemoveAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *ServiceAccountRemover_RemoveAll_Call) Return(_a0 error) *ServiceAccountRemover_RemoveAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ServiceAccountRemover_RemoveAll_Call) RunAndReturn(run func(context.Context, *core.Dogu) error) *ServiceAccountRemover_RemoveAll_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewServiceAccountRemover interface {
	mock.TestingT
	Cleanup(func())
}

// NewServiceAccountRemover creates a new instance of ServiceAccountRemover. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewServiceAccountRemover(t mockConstructorTestingTNewServiceAccountRemover) *ServiceAccountRemover {
	mock := &ServiceAccountRemover{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}