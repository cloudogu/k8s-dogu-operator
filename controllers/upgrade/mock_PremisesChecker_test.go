// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

// MockPremisesChecker is an autogenerated mock type for the PremisesChecker type
type MockPremisesChecker struct {
	mock.Mock
}

type MockPremisesChecker_Expecter struct {
	mock *mock.Mock
}

func (_m *MockPremisesChecker) EXPECT() *MockPremisesChecker_Expecter {
	return &MockPremisesChecker_Expecter{mock: &_m.Mock}
}

// Check provides a mock function with given fields: ctx, toDoguResource, fromDogu, toDogu
func (_m *MockPremisesChecker) Check(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu) error {
	ret := _m.Called(ctx, toDoguResource, fromDogu, toDogu)

	if len(ret) == 0 {
		panic("no return value specified for Check")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu, *core.Dogu) error); ok {
		r0 = rf(ctx, toDoguResource, fromDogu, toDogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockPremisesChecker_Check_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Check'
type MockPremisesChecker_Check_Call struct {
	*mock.Call
}

// Check is a helper method to define mock.On call
//   - ctx context.Context
//   - toDoguResource *v2.Dogu
//   - fromDogu *core.Dogu
//   - toDogu *core.Dogu
func (_e *MockPremisesChecker_Expecter) Check(ctx interface{}, toDoguResource interface{}, fromDogu interface{}, toDogu interface{}) *MockPremisesChecker_Check_Call {
	return &MockPremisesChecker_Check_Call{Call: _e.mock.On("Check", ctx, toDoguResource, fromDogu, toDogu)}
}

func (_c *MockPremisesChecker_Check_Call) Run(run func(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu)) *MockPremisesChecker_Check_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu), args[3].(*core.Dogu))
	})
	return _c
}

func (_c *MockPremisesChecker_Check_Call) Return(_a0 error) *MockPremisesChecker_Check_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockPremisesChecker_Check_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu, *core.Dogu) error) *MockPremisesChecker_Check_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockPremisesChecker creates a new instance of MockPremisesChecker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockPremisesChecker(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockPremisesChecker {
	mock := &MockPremisesChecker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
