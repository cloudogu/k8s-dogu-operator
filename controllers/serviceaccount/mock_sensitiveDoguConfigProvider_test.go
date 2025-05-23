// Code generated by mockery v2.53.3. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockSensitiveDoguConfigProvider is an autogenerated mock type for the sensitiveDoguConfigProvider type
type mockSensitiveDoguConfigProvider struct {
	mock.Mock
}

type mockSensitiveDoguConfigProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSensitiveDoguConfigProvider) EXPECT() *mockSensitiveDoguConfigProvider_Expecter {
	return &mockSensitiveDoguConfigProvider_Expecter{mock: &_m.Mock}
}

// GetSensitiveDoguConfig provides a mock function with given fields: ctx, doguName
func (_m *mockSensitiveDoguConfigProvider) GetSensitiveDoguConfig(ctx context.Context, doguName string) (sensitiveDoguConfig, error) {
	ret := _m.Called(ctx, doguName)

	if len(ret) == 0 {
		panic("no return value specified for GetSensitiveDoguConfig")
	}

	var r0 sensitiveDoguConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (sensitiveDoguConfig, error)); ok {
		return rf(ctx, doguName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) sensitiveDoguConfig); ok {
		r0 = rf(ctx, doguName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(sensitiveDoguConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, doguName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSensitiveDoguConfig'
type mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call struct {
	*mock.Call
}

// GetSensitiveDoguConfig is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName string
func (_e *mockSensitiveDoguConfigProvider_Expecter) GetSensitiveDoguConfig(ctx interface{}, doguName interface{}) *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call {
	return &mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call{Call: _e.mock.On("GetSensitiveDoguConfig", ctx, doguName)}
}

func (_c *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call) Run(run func(ctx context.Context, doguName string)) *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call) Return(_a0 sensitiveDoguConfig, _a1 error) *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call) RunAndReturn(run func(context.Context, string) (sensitiveDoguConfig, error)) *mockSensitiveDoguConfigProvider_GetSensitiveDoguConfig_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSensitiveDoguConfigProvider creates a new instance of mockSensitiveDoguConfigProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSensitiveDoguConfigProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSensitiveDoguConfigProvider {
	mock := &mockSensitiveDoguConfigProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
