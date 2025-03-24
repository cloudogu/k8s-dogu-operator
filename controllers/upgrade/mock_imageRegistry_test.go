// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockImageRegistry is an autogenerated mock type for the imageRegistry type
type mockImageRegistry struct {
	mock.Mock
}

type mockImageRegistry_Expecter struct {
	mock *mock.Mock
}

func (_m *mockImageRegistry) EXPECT() *mockImageRegistry_Expecter {
	return &mockImageRegistry_Expecter{mock: &_m.Mock}
}

// PullImageConfig provides a mock function with given fields: ctx, image
func (_m *mockImageRegistry) PullImageConfig(ctx context.Context, image string) (*v1.ConfigFile, error) {
	ret := _m.Called(ctx, image)

	if len(ret) == 0 {
		panic("no return value specified for PullImageConfig")
	}

	var r0 *v1.ConfigFile
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1.ConfigFile, error)); ok {
		return rf(ctx, image)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1.ConfigFile); ok {
		r0 = rf(ctx, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.ConfigFile)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockImageRegistry_PullImageConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PullImageConfig'
type mockImageRegistry_PullImageConfig_Call struct {
	*mock.Call
}

// PullImageConfig is a helper method to define mock.On call
//   - ctx context.Context
//   - image string
func (_e *mockImageRegistry_Expecter) PullImageConfig(ctx interface{}, image interface{}) *mockImageRegistry_PullImageConfig_Call {
	return &mockImageRegistry_PullImageConfig_Call{Call: _e.mock.On("PullImageConfig", ctx, image)}
}

func (_c *mockImageRegistry_PullImageConfig_Call) Run(run func(ctx context.Context, image string)) *mockImageRegistry_PullImageConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockImageRegistry_PullImageConfig_Call) Return(_a0 *v1.ConfigFile, _a1 error) *mockImageRegistry_PullImageConfig_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockImageRegistry_PullImageConfig_Call) RunAndReturn(run func(context.Context, string) (*v1.ConfigFile, error)) *mockImageRegistry_PullImageConfig_Call {
	_c.Call.Return(run)
	return _c
}

// newMockImageRegistry creates a new instance of mockImageRegistry. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockImageRegistry(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockImageRegistry {
	mock := &mockImageRegistry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
