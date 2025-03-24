// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

// MockUpgradeExecutor is an autogenerated mock type for the UpgradeExecutor type
type MockUpgradeExecutor struct {
	mock.Mock
}

type MockUpgradeExecutor_Expecter struct {
	mock *mock.Mock
}

func (_m *MockUpgradeExecutor) EXPECT() *MockUpgradeExecutor_Expecter {
	return &MockUpgradeExecutor_Expecter{mock: &_m.Mock}
}

// Upgrade provides a mock function with given fields: ctx, toDoguResource, fromDogu, toDogu
func (_m *MockUpgradeExecutor) Upgrade(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu) error {
	ret := _m.Called(ctx, toDoguResource, fromDogu, toDogu)

	if len(ret) == 0 {
		panic("no return value specified for Upgrade")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu, *core.Dogu) error); ok {
		r0 = rf(ctx, toDoguResource, fromDogu, toDogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockUpgradeExecutor_Upgrade_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Upgrade'
type MockUpgradeExecutor_Upgrade_Call struct {
	*mock.Call
}

// Upgrade is a helper method to define mock.On call
//   - ctx context.Context
//   - toDoguResource *v2.Dogu
//   - fromDogu *core.Dogu
//   - toDogu *core.Dogu
func (_e *MockUpgradeExecutor_Expecter) Upgrade(ctx interface{}, toDoguResource interface{}, fromDogu interface{}, toDogu interface{}) *MockUpgradeExecutor_Upgrade_Call {
	return &MockUpgradeExecutor_Upgrade_Call{Call: _e.mock.On("Upgrade", ctx, toDoguResource, fromDogu, toDogu)}
}

func (_c *MockUpgradeExecutor_Upgrade_Call) Run(run func(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu)) *MockUpgradeExecutor_Upgrade_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu), args[3].(*core.Dogu))
	})
	return _c
}

func (_c *MockUpgradeExecutor_Upgrade_Call) Return(_a0 error) *MockUpgradeExecutor_Upgrade_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockUpgradeExecutor_Upgrade_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu, *core.Dogu) error) *MockUpgradeExecutor_Upgrade_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockUpgradeExecutor creates a new instance of MockUpgradeExecutor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUpgradeExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUpgradeExecutor {
	mock := &MockUpgradeExecutor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
