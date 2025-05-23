// Code generated by mockery v2.53.3. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockSensitiveDoguConfig is an autogenerated mock type for the sensitiveDoguConfig type
type mockSensitiveDoguConfig struct {
	mock.Mock
}

type mockSensitiveDoguConfig_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSensitiveDoguConfig) EXPECT() *mockSensitiveDoguConfig_Expecter {
	return &mockSensitiveDoguConfig_Expecter{mock: &_m.Mock}
}

// DeleteRecursive provides a mock function with given fields: ctx, key
func (_m *mockSensitiveDoguConfig) DeleteRecursive(ctx context.Context, key string) error {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for DeleteRecursive")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockSensitiveDoguConfig_DeleteRecursive_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteRecursive'
type mockSensitiveDoguConfig_DeleteRecursive_Call struct {
	*mock.Call
}

// DeleteRecursive is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *mockSensitiveDoguConfig_Expecter) DeleteRecursive(ctx interface{}, key interface{}) *mockSensitiveDoguConfig_DeleteRecursive_Call {
	return &mockSensitiveDoguConfig_DeleteRecursive_Call{Call: _e.mock.On("DeleteRecursive", ctx, key)}
}

func (_c *mockSensitiveDoguConfig_DeleteRecursive_Call) Run(run func(ctx context.Context, key string)) *mockSensitiveDoguConfig_DeleteRecursive_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfig_DeleteRecursive_Call) Return(_a0 error) *mockSensitiveDoguConfig_DeleteRecursive_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockSensitiveDoguConfig_DeleteRecursive_Call) RunAndReturn(run func(context.Context, string) error) *mockSensitiveDoguConfig_DeleteRecursive_Call {
	_c.Call.Return(run)
	return _c
}

// Exists provides a mock function with given fields: ctx, key
func (_m *mockSensitiveDoguConfig) Exists(ctx context.Context, key string) (bool, error) {
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

// mockSensitiveDoguConfig_Exists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exists'
type mockSensitiveDoguConfig_Exists_Call struct {
	*mock.Call
}

// Exists is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *mockSensitiveDoguConfig_Expecter) Exists(ctx interface{}, key interface{}) *mockSensitiveDoguConfig_Exists_Call {
	return &mockSensitiveDoguConfig_Exists_Call{Call: _e.mock.On("Exists", ctx, key)}
}

func (_c *mockSensitiveDoguConfig_Exists_Call) Run(run func(ctx context.Context, key string)) *mockSensitiveDoguConfig_Exists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfig_Exists_Call) Return(_a0 bool, _a1 error) *mockSensitiveDoguConfig_Exists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfig_Exists_Call) RunAndReturn(run func(context.Context, string) (bool, error)) *mockSensitiveDoguConfig_Exists_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, key
func (_m *mockSensitiveDoguConfig) Get(ctx context.Context, key string) (string, error) {
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

// mockSensitiveDoguConfig_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockSensitiveDoguConfig_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *mockSensitiveDoguConfig_Expecter) Get(ctx interface{}, key interface{}) *mockSensitiveDoguConfig_Get_Call {
	return &mockSensitiveDoguConfig_Get_Call{Call: _e.mock.On("Get", ctx, key)}
}

func (_c *mockSensitiveDoguConfig_Get_Call) Run(run func(ctx context.Context, key string)) *mockSensitiveDoguConfig_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfig_Get_Call) Return(_a0 string, _a1 error) *mockSensitiveDoguConfig_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfig_Get_Call) RunAndReturn(run func(context.Context, string) (string, error)) *mockSensitiveDoguConfig_Get_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: ctx, key, value
func (_m *mockSensitiveDoguConfig) Set(ctx context.Context, key string, value string) error {
	ret := _m.Called(ctx, key, value)

	if len(ret) == 0 {
		panic("no return value specified for Set")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockSensitiveDoguConfig_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type mockSensitiveDoguConfig_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
//   - value string
func (_e *mockSensitiveDoguConfig_Expecter) Set(ctx interface{}, key interface{}, value interface{}) *mockSensitiveDoguConfig_Set_Call {
	return &mockSensitiveDoguConfig_Set_Call{Call: _e.mock.On("Set", ctx, key, value)}
}

func (_c *mockSensitiveDoguConfig_Set_Call) Run(run func(ctx context.Context, key string, value string)) *mockSensitiveDoguConfig_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfig_Set_Call) Return(_a0 error) *mockSensitiveDoguConfig_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockSensitiveDoguConfig_Set_Call) RunAndReturn(run func(context.Context, string, string) error) *mockSensitiveDoguConfig_Set_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSensitiveDoguConfig creates a new instance of mockSensitiveDoguConfig. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSensitiveDoguConfig(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSensitiveDoguConfig {
	mock := &mockSensitiveDoguConfig{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
