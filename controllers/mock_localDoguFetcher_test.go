// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

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

// FetchInstalled provides a mock function with given fields: ctx, doguName
func (_m *mockLocalDoguFetcher) FetchInstalled(ctx context.Context, doguName string) (*core.Dogu, error) {
	ret := _m.Called(ctx, doguName)

	if len(ret) == 0 {
		panic("no return value specified for FetchInstalled")
	}

	var r0 *core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*core.Dogu, error)); ok {
		return rf(ctx, doguName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *core.Dogu); ok {
		r0 = rf(ctx, doguName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
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
//   - doguName string
func (_e *mockLocalDoguFetcher_Expecter) FetchInstalled(ctx interface{}, doguName interface{}) *mockLocalDoguFetcher_FetchInstalled_Call {
	return &mockLocalDoguFetcher_FetchInstalled_Call{Call: _e.mock.On("FetchInstalled", ctx, doguName)}
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) Run(run func(ctx context.Context, doguName string)) *mockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) Return(installedDogu *core.Dogu, err error) *mockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Return(installedDogu, err)
	return _c
}

func (_c *mockLocalDoguFetcher_FetchInstalled_Call) RunAndReturn(run func(context.Context, string) (*core.Dogu, error)) *mockLocalDoguFetcher_FetchInstalled_Call {
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
