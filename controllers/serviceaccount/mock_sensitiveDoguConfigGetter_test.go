// Code generated by mockery v2.46.2. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockSensitiveDoguConfigGetter is an autogenerated mock type for the sensitiveDoguConfigGetter type
type mockSensitiveDoguConfigGetter struct {
	mock.Mock
}

type mockSensitiveDoguConfigGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSensitiveDoguConfigGetter) EXPECT() *mockSensitiveDoguConfigGetter_Expecter {
	return &mockSensitiveDoguConfigGetter_Expecter{mock: &_m.Mock}
}

// Exists provides a mock function with given fields: ctx, key
func (_m *mockSensitiveDoguConfigGetter) Exists(ctx context.Context, key string) (bool, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Exists")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (bool, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockSensitiveDoguConfigGetter_Exists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exists'
type mockSensitiveDoguConfigGetter_Exists_Call struct {
	*mock.Call
}

// Exists is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *mockSensitiveDoguConfigGetter_Expecter) Exists(ctx interface{}, key interface{}) *mockSensitiveDoguConfigGetter_Exists_Call {
	return &mockSensitiveDoguConfigGetter_Exists_Call{Call: _e.mock.On("Exists", ctx, key)}
}

func (_c *mockSensitiveDoguConfigGetter_Exists_Call) Run(run func(ctx context.Context, key string)) *mockSensitiveDoguConfigGetter_Exists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigGetter_Exists_Call) Return(_a0 bool, _a1 error) *mockSensitiveDoguConfigGetter_Exists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigGetter_Exists_Call) RunAndReturn(run func(context.Context, string) (bool, error)) *mockSensitiveDoguConfigGetter_Exists_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, key
func (_m *mockSensitiveDoguConfigGetter) Get(ctx context.Context, key string) (string, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (string, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) string); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockSensitiveDoguConfigGetter_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockSensitiveDoguConfigGetter_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *mockSensitiveDoguConfigGetter_Expecter) Get(ctx interface{}, key interface{}) *mockSensitiveDoguConfigGetter_Get_Call {
	return &mockSensitiveDoguConfigGetter_Get_Call{Call: _e.mock.On("Get", ctx, key)}
}

func (_c *mockSensitiveDoguConfigGetter_Get_Call) Run(run func(ctx context.Context, key string)) *mockSensitiveDoguConfigGetter_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigGetter_Get_Call) Return(_a0 string, _a1 error) *mockSensitiveDoguConfigGetter_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigGetter_Get_Call) RunAndReturn(run func(context.Context, string) (string, error)) *mockSensitiveDoguConfigGetter_Get_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSensitiveDoguConfigGetter creates a new instance of mockSensitiveDoguConfigGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSensitiveDoguConfigGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSensitiveDoguConfigGetter {
	mock := &mockSensitiveDoguConfigGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
