// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// AsyncStep is an autogenerated mock type for the AsyncStep type
type AsyncStep struct {
	mock.Mock
}

type AsyncStep_Expecter struct {
	mock *mock.Mock
}

func (_m *AsyncStep) EXPECT() *AsyncStep_Expecter {
	return &AsyncStep_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: ctx, dogu
func (_m *AsyncStep) Execute(ctx context.Context, dogu *v1.Dogu) (string, error) {
	ret := _m.Called(ctx, dogu)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) (string, error)); ok {
		return rf(ctx, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) string); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu) error); ok {
		r1 = rf(ctx, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AsyncStep_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type AsyncStep_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
func (_e *AsyncStep_Expecter) Execute(ctx interface{}, dogu interface{}) *AsyncStep_Execute_Call {
	return &AsyncStep_Execute_Call{Call: _e.mock.On("Execute", ctx, dogu)}
}

func (_c *AsyncStep_Execute_Call) Run(run func(ctx context.Context, dogu *v1.Dogu)) *AsyncStep_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *AsyncStep_Execute_Call) Return(_a0 string, _a1 error) *AsyncStep_Execute_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *AsyncStep_Execute_Call) RunAndReturn(run func(context.Context, *v1.Dogu) (string, error)) *AsyncStep_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// GetStartCondition provides a mock function with given fields:
func (_m *AsyncStep) GetStartCondition() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// AsyncStep_GetStartCondition_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStartCondition'
type AsyncStep_GetStartCondition_Call struct {
	*mock.Call
}

// GetStartCondition is a helper method to define mock.On call
func (_e *AsyncStep_Expecter) GetStartCondition() *AsyncStep_GetStartCondition_Call {
	return &AsyncStep_GetStartCondition_Call{Call: _e.mock.On("GetStartCondition")}
}

func (_c *AsyncStep_GetStartCondition_Call) Run(run func()) *AsyncStep_GetStartCondition_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *AsyncStep_GetStartCondition_Call) Return(_a0 string) *AsyncStep_GetStartCondition_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *AsyncStep_GetStartCondition_Call) RunAndReturn(run func() string) *AsyncStep_GetStartCondition_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewAsyncStep interface {
	mock.TestingT
	Cleanup(func())
}

// NewAsyncStep creates a new instance of AsyncStep. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAsyncStep(t mockConstructorTestingTNewAsyncStep) *AsyncStep {
	mock := &AsyncStep{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
