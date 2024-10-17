// Code generated by mockery v2.46.2. DO NOT EDIT.

package health

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	types "k8s.io/apimachinery/pkg/types"
)

// mockDoguHealthChecker is an autogenerated mock type for the doguHealthChecker type
type mockDoguHealthChecker struct {
	mock.Mock
}

type mockDoguHealthChecker_Expecter struct {
	mock *mock.Mock
}

func (_m *mockDoguHealthChecker) EXPECT() *mockDoguHealthChecker_Expecter {
	return &mockDoguHealthChecker_Expecter{mock: &_m.Mock}
}

// CheckByName provides a mock function with given fields: ctx, doguName
func (_m *mockDoguHealthChecker) CheckByName(ctx context.Context, doguName types.NamespacedName) error {
	ret := _m.Called(ctx, doguName)

	if len(ret) == 0 {
		panic("no return value specified for CheckByName")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) error); ok {
		r0 = rf(ctx, doguName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockDoguHealthChecker_CheckByName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckByName'
type mockDoguHealthChecker_CheckByName_Call struct {
	*mock.Call
}

// CheckByName is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName types.NamespacedName
func (_e *mockDoguHealthChecker_Expecter) CheckByName(ctx interface{}, doguName interface{}) *mockDoguHealthChecker_CheckByName_Call {
	return &mockDoguHealthChecker_CheckByName_Call{Call: _e.mock.On("CheckByName", ctx, doguName)}
}

func (_c *mockDoguHealthChecker_CheckByName_Call) Run(run func(ctx context.Context, doguName types.NamespacedName)) *mockDoguHealthChecker_CheckByName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.NamespacedName))
	})
	return _c
}

func (_c *mockDoguHealthChecker_CheckByName_Call) Return(_a0 error) *mockDoguHealthChecker_CheckByName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockDoguHealthChecker_CheckByName_Call) RunAndReturn(run func(context.Context, types.NamespacedName) error) *mockDoguHealthChecker_CheckByName_Call {
	_c.Call.Return(run)
	return _c
}

// newMockDoguHealthChecker creates a new instance of mockDoguHealthChecker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDoguHealthChecker(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockDoguHealthChecker {
	mock := &mockDoguHealthChecker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
