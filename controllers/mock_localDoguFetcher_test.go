// Code generated by mockery v2.53.3. DO NOT EDIT.

package controllers

import (
	context "context"

	dogu "github.com/cloudogu/ces-commons-lib/dogu"
	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"
)

// mockLocalDoguFetcher is an autogenerated mock type for the localDoguFetcher type
type mockLocalDoguFetcher struct {
	mock.Mock
}

type mockLocalDoguFetcher_Expecter struct {
	mock *mock.Mock
}

func (_m *mockLocalDoguFetcher) EXPECT() *mockLocalDoguFetcher_Expecter {
	return &mockLocalDoguFetcher_Expecter{mock: &_m.Mock}
}

// Enabled provides a mock function with given fields: ctx, doguName
func (_m *mockLocalDoguFetcher) Enabled(ctx context.Context, doguName dogu.SimpleName) (bool, error) {
	ret := _m.Called(ctx, doguName)

	if len(ret) == 0 {
		panic("no return value specified for Enabled")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) (bool, error)); ok {
		return rf(ctx, doguName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) bool); ok {
		r0 = rf(ctx, doguName)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.SimpleName) error); ok {
		r1 = rf(ctx, doguName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockLocalDoguFetcher_Enabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enabled'
type mockLocalDoguFetcher_Enabled_Call struct {
	*mock.Call
}

// Enabled is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName dogu.SimpleName
func (_e *mockLocalDoguFetcher_Expecter) Enabled(ctx interface{}, doguName interface{}) *mockLocalDoguFetcher_Enabled_Call {
	return &mockLocalDoguFetcher_Enabled_Call{Call: _e.mock.On("Enabled", ctx, doguName)}
}

func (_c *mockLocalDoguFetcher_Enabled_Call) Run(run func(ctx context.Context, doguName dogu.SimpleName)) *mockLocalDoguFetcher_Enabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleName))
	})
	return _c
}

func (_c *mockLocalDoguFetcher_Enabled_Call) Return(_a0 bool, _a1 error) *mockLocalDoguFetcher_Enabled_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockLocalDoguFetcher_Enabled_Call) RunAndReturn(run func(context.Context, dogu.SimpleName) (bool, error)) *mockLocalDoguFetcher_Enabled_Call {
	_c.Call.Return(run)
	return _c
}

// FetchInstalled provides a mock function with given fields: ctx, doguName
func (_m *mockLocalDoguFetcher) FetchInstalled(ctx context.Context, doguName dogu.SimpleName) (*core.Dogu, error) {
	ret := _m.Called(ctx, doguName)

	if len(ret) == 0 {
		panic("no return value specified for FetchInstalled")
	}

	var r0 *core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) (*core.Dogu, error)); ok {
		return rf(ctx, doguName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) *core.Dogu); ok {
		r0 = rf(ctx, doguName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.SimpleName) error); ok {
		r1 = rf(ctx, doguName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockLocalDoguFetcher_FetchInstalled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FetchInstalled'
type mockLocalDoguFetcher_FetchInstalled_Call struct {
	*mock.Call
}

// FetchInstalled is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName dogu.SimpleName
func (_e *mockLocalDoguFetcher_Expecter) FetchInstalled(ctx interface{}, doguName interface{}) *mockLocalDoguFetcher_FetchInstalled_Call {
	return &mockLocalDoguFetcher_FetchInstalled_Call{Call: _e.mock.On("FetchInstalled", ctx, doguName)}
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) Run(run func(ctx context.Context, doguName dogu.SimpleName)) *mockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleName))
	})
	return _c
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) Return(installedDogu *core.Dogu, err error) *mockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Return(installedDogu, err)
	return _c
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) RunAndReturn(run func(context.Context, dogu.SimpleName) (*core.Dogu, error)) *mockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Return(run)
	return _c
}

// newMockLocalDoguFetcher creates a new instance of mockLocalDoguFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockLocalDoguFetcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockLocalDoguFetcher {
	mock := &mockLocalDoguFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
