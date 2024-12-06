// Code generated by mockery v2.46.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// mockDoguRegistrator is an autogenerated mock type for the doguRegistrator type
type mockDoguRegistrator struct {
	mock.Mock
}

type mockDoguRegistrator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockDoguRegistrator) EXPECT() *mockDoguRegistrator_Expecter {
	return &mockDoguRegistrator_Expecter{mock: &_m.Mock}
}

// RegisterDoguVersion provides a mock function with given fields: ctx, dogu
func (_m *mockDoguRegistrator) RegisterDoguVersion(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	if len(ret) == 0 {
		panic("no return value specified for RegisterDoguVersion")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockDoguRegistrator_RegisterDoguVersion_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RegisterDoguVersion'
type mockDoguRegistrator_RegisterDoguVersion_Call struct {
	*mock.Call
}

// RegisterDoguVersion is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *mockDoguRegistrator_Expecter) RegisterDoguVersion(ctx interface{}, dogu interface{}) *mockDoguRegistrator_RegisterDoguVersion_Call {
	return &mockDoguRegistrator_RegisterDoguVersion_Call{Call: _e.mock.On("RegisterDoguVersion", ctx, dogu)}
}

func (_c *mockDoguRegistrator_RegisterDoguVersion_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *mockDoguRegistrator_RegisterDoguVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *mockDoguRegistrator_RegisterDoguVersion_Call) Return(_a0 error) *mockDoguRegistrator_RegisterDoguVersion_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockDoguRegistrator_RegisterDoguVersion_Call) RunAndReturn(run func(context.Context, *core.Dogu) error) *mockDoguRegistrator_RegisterDoguVersion_Call {
	_c.Call.Return(run)
	return _c
}

// newMockDoguRegistrator creates a new instance of mockDoguRegistrator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDoguRegistrator(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockDoguRegistrator {
	mock := &mockDoguRegistrator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
