// Code generated by mockery v2.53.2. DO NOT EDIT.

package cesregistry

import (
	context "context"

	dogu "github.com/cloudogu/ces-commons-lib/dogu"
	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"
)

// MockLocalDoguFetcher is an autogenerated mock type for the LocalDoguFetcher type
type MockLocalDoguFetcher struct {
	mock.Mock
}

type MockLocalDoguFetcher_Expecter struct {
	mock *mock.Mock
}

func (_m *MockLocalDoguFetcher) EXPECT() *MockLocalDoguFetcher_Expecter {
	return &MockLocalDoguFetcher_Expecter{mock: &_m.Mock}
}

// Enabled provides a mock function with given fields: ctx, doguName
func (_m *MockLocalDoguFetcher) Enabled(ctx context.Context, doguName dogu.SimpleName) (bool, error) {
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

// MockLocalDoguFetcher_Enabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enabled'
type MockLocalDoguFetcher_Enabled_Call struct {
	*mock.Call
}

// Enabled is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName dogu.SimpleName
func (_e *MockLocalDoguFetcher_Expecter) Enabled(ctx interface{}, doguName interface{}) *MockLocalDoguFetcher_Enabled_Call {
	return &MockLocalDoguFetcher_Enabled_Call{Call: _e.mock.On("Enabled", ctx, doguName)}
}

func (_c *MockLocalDoguFetcher_Enabled_Call) Run(run func(ctx context.Context, doguName dogu.SimpleName)) *MockLocalDoguFetcher_Enabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleName))
	})
	return _c
}

func (_c *MockLocalDoguFetcher_Enabled_Call) Return(_a0 bool, _a1 error) *MockLocalDoguFetcher_Enabled_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockLocalDoguFetcher_Enabled_Call) RunAndReturn(run func(context.Context, dogu.SimpleName) (bool, error)) *MockLocalDoguFetcher_Enabled_Call {
	_c.Call.Return(run)
	return _c
}

// FetchInstalled provides a mock function with given fields: ctx, doguName
func (_m *MockLocalDoguFetcher) FetchInstalled(ctx context.Context, doguName dogu.SimpleName) (*core.Dogu, error) {
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

// MockLocalDoguFetcher_FetchInstalled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FetchInstalled'
type MockLocalDoguFetcher_FetchInstalled_Call struct {
	*mock.Call
}

// FetchInstalled is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName dogu.SimpleName
func (_e *MockLocalDoguFetcher_Expecter) FetchInstalled(ctx interface{}, doguName interface{}) *MockLocalDoguFetcher_FetchInstalled_Call {
	return &MockLocalDoguFetcher_FetchInstalled_Call{Call: _e.mock.On("FetchInstalled", ctx, doguName)}
}

func (_c *MockLocalDoguFetcher_FetchInstalled_Call) Run(run func(ctx context.Context, doguName dogu.SimpleName)) *MockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleName))
	})
	return _c
}

func (_c *MockLocalDoguFetcher_FetchInstalled_Call) Return(installedDogu *core.Dogu, err error) *MockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Return(installedDogu, err)
	return _c
}

func (_c *MockLocalDoguFetcher_FetchInstalled_Call) RunAndReturn(run func(context.Context, dogu.SimpleName) (*core.Dogu, error)) *MockLocalDoguFetcher_FetchInstalled_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockLocalDoguFetcher creates a new instance of MockLocalDoguFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockLocalDoguFetcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLocalDoguFetcher {
	mock := &MockLocalDoguFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
