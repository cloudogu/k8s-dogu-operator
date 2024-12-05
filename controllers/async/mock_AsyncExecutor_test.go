// Code generated by mockery v2.20.0. DO NOT EDIT.

package async

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// MockAsyncExecutor is an autogenerated mock type for the AsyncExecutor type
type MockAsyncExecutor struct {
	mock.Mock
}

type MockAsyncExecutor_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAsyncExecutor) EXPECT() *MockAsyncExecutor_Expecter {
	return &MockAsyncExecutor_Expecter{mock: &_m.Mock}
}

// AddStep provides a mock function with given fields: step
func (_m *MockAsyncExecutor) AddStep(step AsyncStep) {
	_m.Called(step)
}

// MockAsyncExecutor_AddStep_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddStep'
type MockAsyncExecutor_AddStep_Call struct {
	*mock.Call
}

// AddStep is a helper method to define mock.On call
//   - step AsyncStep
func (_e *MockAsyncExecutor_Expecter) AddStep(step interface{}) *MockAsyncExecutor_AddStep_Call {
	return &MockAsyncExecutor_AddStep_Call{Call: _e.mock.On("AddStep", step)}
}

func (_c *MockAsyncExecutor_AddStep_Call) Run(run func(step AsyncStep)) *MockAsyncExecutor_AddStep_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(AsyncStep))
	})
	return _c
}

func (_c *MockAsyncExecutor_AddStep_Call) Return() *MockAsyncExecutor_AddStep_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockAsyncExecutor_AddStep_Call) RunAndReturn(run func(AsyncStep)) *MockAsyncExecutor_AddStep_Call {
	_c.Call.Return(run)
	return _c
}

// Execute provides a mock function with given fields: ctx, dogu, currentState
func (_m *MockAsyncExecutor) Execute(ctx context.Context, dogu *v2.Dogu, currentState string) error {
	ret := _m.Called(ctx, dogu, currentState)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, string) error); ok {
		r0 = rf(ctx, dogu, currentState)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAsyncExecutor_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type MockAsyncExecutor_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - currentState string
func (_e *MockAsyncExecutor_Expecter) Execute(ctx interface{}, dogu interface{}, currentState interface{}) *MockAsyncExecutor_Execute_Call {
	return &MockAsyncExecutor_Execute_Call{Call: _e.mock.On("Execute", ctx, dogu, currentState)}
}

func (_c *MockAsyncExecutor_Execute_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, currentState string)) *MockAsyncExecutor_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(string))
	})
	return _c
}

func (_c *MockAsyncExecutor_Execute_Call) Return(_a0 error) *MockAsyncExecutor_Execute_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAsyncExecutor_Execute_Call) RunAndReturn(run func(context.Context, *v2.Dogu, string) error) *MockAsyncExecutor_Execute_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockAsyncExecutor interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockAsyncExecutor creates a new instance of MockAsyncExecutor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockAsyncExecutor(t mockConstructorTestingTNewMockAsyncExecutor) *MockAsyncExecutor {
	mock := &MockAsyncExecutor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
