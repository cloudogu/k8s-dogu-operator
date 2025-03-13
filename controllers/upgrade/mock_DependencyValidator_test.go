// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// MockDependencyValidator is an autogenerated mock type for the DependencyValidator type
type MockDependencyValidator struct {
	mock.Mock
}

type MockDependencyValidator_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDependencyValidator) EXPECT() *MockDependencyValidator_Expecter {
	return &MockDependencyValidator_Expecter{mock: &_m.Mock}
}

// ValidateDependencies provides a mock function with given fields: ctx, dogu
func (_m *MockDependencyValidator) ValidateDependencies(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	if len(ret) == 0 {
		panic("no return value specified for ValidateDependencies")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDependencyValidator_ValidateDependencies_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateDependencies'
type MockDependencyValidator_ValidateDependencies_Call struct {
	*mock.Call
}

// ValidateDependencies is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *MockDependencyValidator_Expecter) ValidateDependencies(ctx interface{}, dogu interface{}) *MockDependencyValidator_ValidateDependencies_Call {
	return &MockDependencyValidator_ValidateDependencies_Call{Call: _e.mock.On("ValidateDependencies", ctx, dogu)}
}

func (_c *MockDependencyValidator_ValidateDependencies_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *MockDependencyValidator_ValidateDependencies_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *MockDependencyValidator_ValidateDependencies_Call) Return(_a0 error) *MockDependencyValidator_ValidateDependencies_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDependencyValidator_ValidateDependencies_Call) RunAndReturn(run func(context.Context, *core.Dogu) error) *MockDependencyValidator_ValidateDependencies_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDependencyValidator creates a new instance of MockDependencyValidator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDependencyValidator(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDependencyValidator {
	mock := &MockDependencyValidator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
