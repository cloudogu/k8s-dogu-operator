// Code generated by mockery v2.53.2. DO NOT EDIT.

package resource

import (
	context "context"

	config "github.com/cloudogu/k8s-registry-lib/config"

	mock "github.com/stretchr/testify/mock"

	repository "github.com/cloudogu/k8s-registry-lib/repository"
)

// mockGlobalConfigurationWatcher is an autogenerated mock type for the globalConfigurationWatcher type
type mockGlobalConfigurationWatcher struct {
	mock.Mock
}

type mockGlobalConfigurationWatcher_Expecter struct {
	mock *mock.Mock
}

func (_m *mockGlobalConfigurationWatcher) EXPECT() *mockGlobalConfigurationWatcher_Expecter {
	return &mockGlobalConfigurationWatcher_Expecter{mock: &_m.Mock}
}

// Watch provides a mock function with given fields: ctx, filters
func (_m *mockGlobalConfigurationWatcher) Watch(ctx context.Context, filters ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error) {
	_va := make([]interface{}, len(filters))
	for _i := range filters {
		_va[_i] = filters[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Watch")
	}

	var r0 <-chan repository.GlobalConfigWatchResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error)); ok {
		return rf(ctx, filters...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...config.WatchFilter) <-chan repository.GlobalConfigWatchResult); ok {
		r0 = rf(ctx, filters...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan repository.GlobalConfigWatchResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...config.WatchFilter) error); ok {
		r1 = rf(ctx, filters...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockGlobalConfigurationWatcher_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockGlobalConfigurationWatcher_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - filters ...config.WatchFilter
func (_e *mockGlobalConfigurationWatcher_Expecter) Watch(ctx interface{}, filters ...interface{}) *mockGlobalConfigurationWatcher_Watch_Call {
	return &mockGlobalConfigurationWatcher_Watch_Call{Call: _e.mock.On("Watch",
		append([]interface{}{ctx}, filters...)...)}
}

func (_c *mockGlobalConfigurationWatcher_Watch_Call) Run(run func(ctx context.Context, filters ...config.WatchFilter)) *mockGlobalConfigurationWatcher_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]config.WatchFilter, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(config.WatchFilter)
			}
		}
		run(args[0].(context.Context), variadicArgs...)
	})
	return _c
}

func (_c *mockGlobalConfigurationWatcher_Watch_Call) Return(_a0 <-chan repository.GlobalConfigWatchResult, _a1 error) *mockGlobalConfigurationWatcher_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockGlobalConfigurationWatcher_Watch_Call) RunAndReturn(run func(context.Context, ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error)) *mockGlobalConfigurationWatcher_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// newMockGlobalConfigurationWatcher creates a new instance of mockGlobalConfigurationWatcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockGlobalConfigurationWatcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockGlobalConfigurationWatcher {
	mock := &mockGlobalConfigurationWatcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
